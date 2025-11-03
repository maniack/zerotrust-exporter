package magicwan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vinistoisr/zerotrust-exporter/internal/appmetrics"
	"github.com/vinistoisr/zerotrust-exporter/internal/config"
)

const getTunnelHealthCheckResults = `
query GetTunnelHealthCheckResults($accountTag: String!, $datetimeStart: Time!, $datetimeEnd: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      magicTransitTunnelHealthChecksAdaptiveGroups(
        limit: 100,
        filter: {
          datetime_geq: $datetimeStart,
          datetime_lt: $datetimeEnd
        }
      ) {
        avg {
          tunnelState
        }
        dimensions {
          tunnelName
          edgeColoName
        }
      }
    }
  }
}
`

type TunnelHealthCheckResults struct {
	Data struct {
		Viewer struct {
			Accounts []struct {
				MagicTransitTunnelHealthChecksAdaptiveGroups []struct {
					Avg struct {
						TunnelState float64 `json:"tunnelState"`
					} `json:"avg"`
					Dimensions struct {
						TunnelName   string `json:"tunnelName"`
						EdgeColoName string `json:"edgeColoName"`
					} `json:"dimensions"`
				} `json:"magicTransitTunnelHealthChecksAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []interface{} `json:"errors"`
}

func CollectMagicWANState(ctx context.Context) {
	appmetrics.IncApiCallCounter()

	startTime := time.Now()

	end := time.Now().UTC().Truncate(time.Minute * 5)
	start := end.Add(-time.Minute * 5)
	apiKey := config.ApiKey

	vars := variables{
		AccountTag:    config.AccountID,
		DatetimeStart: start.Format(time.RFC3339),
		DatetimeEnd:   end.Format(time.RFC3339),
	}

	bodyMap := map[string]interface{}{
		"query":     getTunnelHealthCheckResults,
		"variables": vars,
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		log.Println("GraphQL request failed:", err)
		appmetrics.SetUpMetric(0)
		return
	}

	respBytes, err := doGraphQLRequest(ctx, apiKey, bodyBytes)
	if err != nil {
		log.Println("GraphQL request failed:", err)
		appmetrics.SetUpMetric(0)
		return
	}

	var gqlResp TunnelHealthCheckResults
	if err := json.Unmarshal(respBytes, &gqlResp); err != nil {
		log.Println("Failed to parse GraphQL response:", err)
		appmetrics.SetUpMetric(0)
		return
	}

	if len(gqlResp.Errors) > 0 {
		log.Println("GraphQL returned errors:", gqlResp.Errors)
		appmetrics.SetUpMetric(0)
	}

	tunnels := 0
	// Collect metrics for each tunnel
	for _, acc := range gqlResp.Data.Viewer.Accounts {
		for _, rec := range acc.MagicTransitTunnelHealthChecksAdaptiveGroups {
			tunnel := rec.Dimensions.TunnelName
			colo := rec.Dimensions.EdgeColoName
			state := rec.Avg.TunnelState
			metrics.GetOrCreateGauge(fmt.Sprintf(`zerotrust_magic_wan_tunnels_up{name="%s", colo="%s"}`, tunnel, colo), func() float64 { return float64(state) })
			tunnels++
		}
	}

	if config.Debug {
		log.Printf("Fetched %d MagicWAN tunnels in %v", tunnels, time.Since(startTime))
	}

}
