// Package main provides an example of how to use the GoTel metrics library
// This demonstrates integration with a web server for publishing metrics to Prometheus
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/GetSimpl/gotel/pkg/server"
)

func main() {
	log.Println("GoTel Example - Metrics Publishing Demo")
	log.Println("======================================")

	// Load configuration from environment variables
	cfg := config.FromEnv()

	// Override with command-line environment variables if present
	if port := os.Getenv("PORT"); port != "" {
		log.Printf("Using port from environment: %s", port)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Show configuration being used
	log.Printf("Configuration loaded:")
	log.Printf("  Prometheus Endpoint: %s", cfg.PrometheusEndpoint)
	log.Printf("  App Name: %s", cfg.AppName)
	log.Printf("  App Version: %s", cfg.AppVersion)
	log.Printf("  Environment: %s", cfg.Environment)
	log.Printf("  Debug Mode: %v", cfg.EnableDebug)
	log.Printf("  Async Metrics: %v", cfg.EnableAsyncMetrics)

	runWebServerExample(cfg)
}

// runWebServerExample shows how to integrate GoTel with a web server
func runWebServerExample(cfg *config.Config) {
	log.Println("\n--- Example 2: Web Server with Metrics ---")

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	// Create server with GoTel integration
	srv := server.NewServer(cfg, port)

	// Setup graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Received shutdown signal")
		cancel()

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Shutdown server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
			os.Exit(1)
		}

		os.Exit(0)
	}()

	// Start server
	log.Printf("Starting web server example on %s", port)
	log.Println("You can test the endpoints:")
	log.Printf("  - Health: http://localhost%s/health", port)
	log.Printf("  - Main: http://localhost%s/", port)
	log.Printf("  - Metrics Info: http://localhost%s/metrics-info", port)
	log.Println("\nPress Ctrl+C to stop the server")

	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Web server example completed!")
}
