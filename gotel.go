// Package gotel provides direct Prometheus metrics push capabilities with real-time delivery.
// It offers a simple API for applications to send metrics directly to Prometheus
// without requiring OpenTelemetry Collector middleware.
package gotel

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/GetSimpl/gotel/pkg/client"
	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/GetSimpl/gotel/pkg/metrics"
)

// GoTel is the main client for sending metrics to Prometheus
type GoTel struct {
	config           *config.Config
	prometheusClient *client.PrometheusClient
	metricsRegistry  *metrics.Registry
	ctx              context.Context
	cancel           context.CancelFunc
}

// New creates a new GoTel client with the provided configuration
func New(cfg *config.Config) (*GoTel, error) {
	if cfg == nil {
		cfg = config.Default()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create Prometheus client using unified configuration
	promClient := client.NewPrometheusClient(cfg)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	gotel := &GoTel{
		config:           cfg,
		prometheusClient: promClient,
		metricsRegistry:  metrics.NewRegistry(),
		ctx:              ctx,
		cancel:           cancel,
	}

	if cfg.LogLevel == "debug" || cfg.EnableDebug {
		log.Printf("GoTel initialized with Prometheus endpoint: %s", cfg.PrometheusEndpoint)
		log.Printf("Async metrics: %v, Buffer size: %d", cfg.EnableAsyncMetrics, cfg.MetricBufferSize)
	}

	return gotel, nil
}

// NewWithDefaults creates a new GoTel client with default configuration
// This is the simplest way to get started with GoTel
func NewWithDefaults() (*GoTel, error) {
	cfg := config.FromEnv()
	return New(cfg)
}

// Counter creates or retrieves a counter metric
func (g *GoTel) Counter(name string, labels map[string]string) *metrics.Counter {
	return g.metricsRegistry.GetOrCreateCounter(name, labels)
}

// Gauge creates or retrieves a gauge metric
func (g *GoTel) Gauge(name string, labels map[string]string) *metrics.Gauge {
	return g.metricsRegistry.GetOrCreateGauge(name, labels)
}

// SendMetrics sends all registered metrics to Prometheus immediately
func (g *GoTel) SendMetrics() error {
	timeSeries := g.metricsRegistry.Collect()
	if len(timeSeries) == 0 {
		return nil
	}

	return g.prometheusClient.SendMetrics(timeSeries)
}

// SendMetricsAsync sends metrics asynchronously in a separate goroutine
func (g *GoTel) SendMetricsAsync() {
	go func() {
		if err := g.SendMetrics(); err != nil {
			if g.config.LogLevel == "debug" || g.config.EnableDebug {
				log.Printf("Failed to send metrics: %v", err)
			}
		}
	}()
}

// StartPeriodicSender starts a background goroutine that sends metrics at regular intervals
func (g *GoTel) StartPeriodicSender(interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second // Default interval
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := g.SendMetrics(); err != nil {
					if g.config.LogLevel == "debug" || g.config.EnableDebug {
						log.Printf("Periodic metrics send failed: %v", err)
					}
				}
			case <-g.ctx.Done():
				return
			}
		}
	}()

	if g.config.LogLevel == "debug" || g.config.EnableDebug {
		log.Printf("Started periodic metrics sender with interval: %v", interval)
	}
}

// IncrementCounter is a convenience method to increment a counter by 1
func (g *GoTel) IncrementCounter(name string, labels map[string]string) error {
	counter := g.Counter(name, labels)
	counter.Inc()

	if g.config.EnableAsyncMetrics {
		g.SendMetricsAsync()
		return nil
	}

	return g.SendMetrics()
}

// SetGauge is a convenience method to set a gauge value
func (g *GoTel) SetGauge(name string, value float64, labels map[string]string) error {
	gauge := g.Gauge(name, labels)
	gauge.Set(value)

	if g.config.EnableAsyncMetrics {
		g.SendMetricsAsync()
		return nil
	}

	return g.SendMetrics()
}

// GetRegistry returns the internal metrics registry for advanced usage
func (g *GoTel) GetRegistry() *metrics.Registry {
	return g.metricsRegistry
}

// GetConfig returns the current configuration
func (g *GoTel) GetConfig() *config.Config {
	return g.config
}

// Close gracefully shuts down the GoTel client
func (g *GoTel) Close() error {
	// Send any remaining metrics
	if err := g.SendMetrics(); err != nil {
		log.Printf("Failed to send final metrics: %v", err)
	}

	// Cancel context to stop any background goroutines
	g.cancel()

	if g.config.LogLevel == "debug" || g.config.EnableDebug {
		log.Println("GoTel client shut down")
	}

	return nil
}

// Health returns basic health information about the GoTel client
func (g *GoTel) Health() map[string]interface{} {
	return map[string]interface{}{
		"status":              "healthy",
		"prometheus_endpoint": g.config.PrometheusEndpoint,
		"metrics_count":       g.metricsRegistry.MetricsCount(),
		"async_enabled":       g.config.EnableAsyncMetrics,
		"config_valid":        g.config.Validate() == nil,
	}
}
