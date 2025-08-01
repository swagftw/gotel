package main

import (
	"log"
	"time"

	"github.com/GetSimpl/gotel"
	"github.com/GetSimpl/gotel/pkg/config"
)

func main() {
	// Example 1: Simple usage with default configuration
	log.Println("=== Example 1: Simple Usage ===")

	// Create client with defaults (reads from environment variables)
	client, err := gotel.NewWithDefaults()
	if err != nil {
		log.Fatalf("Failed to create GoTel client: %v", err)
	}
	defer client.Close()

	// Create and increment a counter
	counter := client.Counter("api_requests_total", map[string]string{
		"service": "example",
		"method":  "GET",
	})
	counter.Inc()

	// Create and set a gauge
	gauge := client.Gauge("queue_size", map[string]string{
		"queue": "processing",
	})
	gauge.Set(42.5)

	// Send metrics synchronously
	if err := client.SendMetricsSync(); err != nil {
		log.Printf("Failed to send metrics: %v", err)
	}

	log.Println("Metrics sent successfully!")

	// Example 2: Custom configuration with async metrics
	log.Println("\n=== Example 2: Custom Configuration ===")

	cfg := config.Default() // Start with defaults
	cfg.OtelEndpoint = "localhost:4318"
	cfg.EnableAsyncMetrics = true
	cfg.SendInterval = 2 * time.Second
	cfg.MinSendInterval = time.Millisecond // 1ms rate limiting
	cfg.MetricBufferSize = 100
	cfg.EnableDebug = true

	asyncClient, err := gotel.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create async GoTel client: %v", err)
	}
	defer asyncClient.Close()

	// Use convenience methods with async sending
	for i := 0; i < 5; i++ {
		// These will send asynchronously via the buffered channel
		asyncClient.IncrementCounter("requests_processed", map[string]string{
			"worker": "worker-1",
			"status": "success",
		})

		asyncClient.SetGauge("active_connections", 15.0, map[string]string{
			"server": "api-server",
		})

		time.Sleep(500 * time.Millisecond)
	}

	// Example 3: Manual metric management
	log.Println("\n=== Example 3: Manual Metric Management ===")

	// Create metrics manually and control when they're sent
	errorCounter := asyncClient.Counter("errors_total", map[string]string{
		"service": "payment",
		"type":    "timeout",
	})

	responseTimeGauge := asyncClient.Gauge("response_time_seconds", map[string]string{
		"endpoint": "/api/v1/payment",
	})

	// Simulate some operations
	for i := 0; i < 3; i++ {
		errorCounter.Add(2)                             // Add 2 errors
		responseTimeGauge.Set(0.150 + float64(i)*0.050) // Increasing response time

		// Send async (will block if buffer is full, ensuring no loss)
		asyncClient.SendMetricsAsync()

		time.Sleep(300 * time.Millisecond)
	}

	// Example 4: Health check
	log.Println("\n=== Example 4: Health Check ===")
	health := asyncClient.Health()
	log.Printf("Client health: %+v", health)

	log.Println("\n=== All examples completed ===")

	// Give async worker time to send final metrics
	time.Sleep(1 * time.Second)
}
