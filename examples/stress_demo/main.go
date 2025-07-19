package main

import (
	"log"
	"sync"
	"time"

	"github.com/GetSimpl/gotel"
	"github.com/GetSimpl/gotel/pkg/config"
)

func main() {
	// Create configuration with small buffer to trigger fallback
	cfg := config.Default()
	cfg.PrometheusEndpoint = "http://localhost:9090/api/v1/write"
	cfg.EnableAsyncMetrics = true
	cfg.SendInterval = 10 * time.Second // Long interval to fill buffer
	cfg.MinSendInterval = time.Millisecond
	cfg.MetricBufferSize = 5 // Very small buffer to trigger fallback quickly
	cfg.EnableDebug = true

	client, err := gotel.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create GoTel client: %v", err)
	}
	defer client.Close()

	log.Printf("Starting stress test with buffer size: %d", cfg.MetricBufferSize)
	log.Printf("Pool capacity: %d", client.Health()["pool_capacity"])

	// Create metrics
	counter := client.Counter("stress_test_requests", map[string]string{
		"test": "fallback",
	})

	gauge := client.Gauge("stress_test_value", map[string]string{
		"test": "fallback",
	})

	// Stress test: send lots of async requests quickly
	var wg sync.WaitGroup
	numGoroutines := 20
	requestsPerGoroutine := 10

	log.Printf("Starting %d goroutines, each sending %d requests", numGoroutines, requestsPerGoroutine)

	start := time.Now()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				counter.Inc()
				gauge.Set(float64(goroutineID*requestsPerGoroutine + j))

				// Send async - should trigger fallback when buffer is full
				client.SendMetricsAsync()

				// Small delay to see the pattern
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	log.Printf("Stress test completed in %v", duration)

	// Check health to see pool usage and dropped requests
	health := client.Health()
	log.Printf("Final health stats:")
	log.Printf("  Pool capacity: %v", health["pool_capacity"])
	log.Printf("  Pool running: %v", health["pool_running"])
	log.Printf("  Pool free: %v", health["pool_free"])
	log.Printf("  Dropped requests: %v", health["dropped_requests"])
	log.Printf("  Metrics count: %v", health["metrics_count"])

	// Give some time for final sends
	log.Println("Waiting for final metric sends...")
	time.Sleep(3 * time.Second)

	// Check final health
	finalHealth := client.Health()
	log.Printf("After cleanup:")
	log.Printf("  Pool running: %v", finalHealth["pool_running"])
	log.Printf("  Dropped requests: %v", finalHealth["dropped_requests"])
}
