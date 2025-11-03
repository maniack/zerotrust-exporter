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

const getTunnelBandwidth = `
query GetTunnelBandwidth($accountTag: string, $datetimeStart: string, $datetimeEnd: string) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      magicTransitTunnelTrafficAdaptiveGroups(
        limit: 100,
        filter: {
          datetime_geq: $datetimeStart,
          datetime_lt:  $datetimeEnd,
          direction: $direction
        }
      ) {
        avg {
          bitRateFiveMinutes
        }
        dimensions {
          tunnelName
          edgeColoName
          datetimeFiveMinutes
        }
      }
    }
  }
}
`

type TunnelBandwidth struct {
	Data struct {
		Viewer struct {
			Accounts []struct {
				MagicTransitTunnelTrafficAdaptiveGroups []struct {
					Avg struct {
						BitRateFiveMinutes float64 `json:"bitRateFiveMinutes"`
					} `json:"avg"`
					Dimensions struct {
						DatetimeFiveMinute string `json:"datetimeFiveMinute"`
						EdgeColoName       string `json:"edgeColoName"`
						TunnelName         string `json:"tunnelName"`
					} `json:"dimensions"`
				} `json:"magicTransitTunnelTrafficAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []interface{} `json:"errors"`
}

func CollectMagicWANBandwidth(ctx context.Context) {
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
		"query":     getTunnelBandwidth,
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

	var gqlResp TunnelBandwidth
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
		for _, rec := range acc.MagicTransitTunnelTrafficAdaptiveGroups {
			tunnel := rec.Dimensions.TunnelName
			colo := rec.Dimensions.EdgeColoName
			bw := rec.Avg.BitRateFiveMinutes
			if len(tunnel) > 0 {
				metrics.GetOrCreateGauge(fmt.Sprintf(`zerotrust_magic_wan_tunnel_bandwidth{name="%s", colo="%s"}`, tunnel, colo), func() float64 { return float64(bw) })
				tunnels++
			}
		}
	}

	if config.Debug {
		log.Printf("Fetched %d MagicWAN tunnels in %v", tunnels, time.Since(startTime))
	}

}
