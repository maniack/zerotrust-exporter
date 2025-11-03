package collector

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vinistoisr/zerotrust-exporter/internal/appmetrics"
	"github.com/vinistoisr/zerotrust-exporter/internal/config"
	"github.com/vinistoisr/zerotrust-exporter/internal/devices"
	"github.com/vinistoisr/zerotrust-exporter/internal/dex"
	"github.com/vinistoisr/zerotrust-exporter/internal/magicwan"
	"github.com/vinistoisr/zerotrust-exporter/internal/tunnels"
	"github.com/vinistoisr/zerotrust-exporter/internal/users"
)

// Register metrics handler
func RegisterHandler() {
	http.HandleFunc("/metrics", MetricsHandler)
}

// StartServer starts the HTTP server
func StartServer(addr string) {
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// metricsHandler handles the /metrics endpoint
func MetricsHandler(w http.ResponseWriter, req *http.Request) {
	// Start timer for scrape duration
	startTime := time.Now()

	// Derive context with timeout from request context for all collectors
	ctx, cancel := context.WithTimeout(req.Context(), 25*time.Second)
	defer cancel()

	// create a channel between device metrics and user metrics
	deviceMetricsChan := make(chan map[string]devices.DeviceStatus, 1)

	appmetrics.SetUpMetric(1)

	// Create a wait group to wait for all goroutines to complete
	wg := new(sync.WaitGroup)

	// Collect device metrics
	if config.EnableDevices {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Collecting device metrics...")
			deviceMetrics := devices.CollectDeviceMetrics()
			deviceMetricsChan <- deviceMetrics
			close(deviceMetricsChan)
		}()
	} else {
		// No devices collection; close the channel so users collector won't block
		close(deviceMetricsChan)
	}

	// Collect user metrics
	if config.EnableUsers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Waiting for device metrics...")
			if deviceMetrics, ok := <-deviceMetricsChan; ok {
				users.CollectUserMetrics(deviceMetrics)
			} else {
				log.Println("Failed to read device metrics from channel.")
			}
		}()
	}

	// Collect tunnel metrics
	if config.EnableTunnels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Collecting tunnel metrics...")
			tunnels.CollectTunnelMetrics()
		}()
	}

	// Collect dex metrics
	if config.EnableDex {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Collecting dex metrics...")
			dex.CollectDexMetrics(ctx, config.AccountID)
		}()
	}

	// Collect magicwan metrics
	if config.EnableMagicWAN {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("Collecting magic wan metrics...")
			magicwan.CollectMagicWANState(ctx)
			magicwan.CollectMagicWANBandwidth(ctx)
		}()
	}

	// Wait for all metrics collection to complete
	log.Println("Waiting for all metrics collection to complete...")
	wg.Wait()
	log.Println("All metrics collection completed.")

	// Update scrape duration metric
	appmetrics.ScrapeDuration.UpdateDuration(startTime)
	// Write metrics to the response
	metrics.WritePrometheus(w, true)

	// Print debug information if enabled
	if config.Debug {
		log.Printf("Scrape completed in %v", time.Since(startTime))
		log.Printf("API calls made: %d", appmetrics.ApiCallCounter.Get())
		log.Printf("API errors encountered: %d", appmetrics.ApiErrorsCounter.Get())
	}
}
