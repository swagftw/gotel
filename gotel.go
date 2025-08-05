// Package gotel provides OpenTelemetry metrics capabilities with automatic batching.
// It offers a simple API for applications to send metrics to OpenTelemetry Collector
// and compatible backends. OTEL SDK handles all batching, buffering, and reliability.
package gotel

import (
	"context"
	"fmt"
	"log"

	"github.com/GetSimpl/gotel/pkg/client"
	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/GetSimpl/gotel/pkg/metrics"
)

type MetricName string

const (
	MetricCounterHttpRequestsTotal MetricName = "http.server.requests.total"
	MetricHistHttpRequestDuration  MetricName = "http.server.request.duration"
)

// gotel is the main client for sending metrics to OpenTelemetry Collector
// OTEL SDK automatically handles batching, buffering, and reliable delivery
type gotel struct {
	config          *config.Config
	otelClient      *client.OtelClient
	metricsRegistry metrics.Registry
	ctx             context.Context
	cancel          context.CancelFunc
}

type Gotel interface {
	IncrementCounter(name MetricName, unit metrics.Unit, labels map[string]interface{})
	SetGauge(name MetricName, unit metrics.Unit, labels map[string]interface{})
	RecordHistogram(name MetricName, unit metrics.Unit, labels map[string]interface{})
}

// New creates a new gotel client with the provided configuration
func New(cfg *config.Config) (Gotel, error) {
	if cfg == nil {
		cfg = config.Default()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create OTEL client
	otelClient, err := client.NewOtelClient(cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create OTEL client: %w", err)
	}

	// Create metrics registry with OTEL client
	registry := metrics.NewRegistry(otelClient, ctx)

	g := &gotel{
		config:          cfg,
		otelClient:      otelClient,
		metricsRegistry: registry,
		ctx:             ctx,
		cancel:          cancel,
	}

	if cfg.EnableDebug {
		log.Printf("gotel client initialized with endpoint: %s", cfg.OtelEndpoint)
		log.Printf("OTEL SDK will automatically batch and send metrics")
	}

	return g, nil
}

// Counter creates or retrieves a counter metric
// Metrics are automatically batched and sent by OTEL SDK
func (g *gotel) Counter(name string, unit metrics.Unit, labels map[string]string) *metrics.Counter {
	return g.metricsRegistry.GetOrCreateCounter(name, labels, unit)
}

// Gauge creates or retrieves a gauge metric
// Metrics are automatically batched and sent by OTEL SDK
func (g *gotel) Gauge(name string, unit metrics.Unit, labels map[string]string) *metrics.Gauge {
	return g.metricsRegistry.GetOrCreateGauge(name, labels, unit)
}

// Histogram creates or retrieves a histogram metric
// Metrics are automatically batched and sent by OTEL SDK
func (g *gotel) Histogram(name string, unit metrics.Unit, labels map[string]string) *metrics.Histogram {
	return g.metricsRegistry.GetOrCreateHistogram(name, labels, unit)
}

// ForceFlush forces immediate export of all pending metrics
// Usually not needed - OTEL SDK automatically exports on schedule
// Use only when you need immediate delivery (e.g., before shutdown)
func (g *gotel) ForceFlush() error {
	return g.otelClient.ForceFlush()
}

// IncrementCounter is a convenience method to increment a counter by 1
// The metric will be automatically batched and sent by OTEL SDK
func (g *gotel) IncrementCounter(name string, unit metrics.Unit, labels map[string]string) {
	counter := g.Counter(name, unit, labels)
	counter.Inc()
}

// SetGauge is a convenience method to set a gauge value
// The metric will be automatically batched and sent by OTEL SDK
func (g *gotel) SetGauge(name string, value float64, unit metrics.Unit, labels map[string]string) {
	gauge := g.Gauge(name, unit, labels)
	gauge.Set(value)
}

// RecordHistogram is a convenience method to record a value in a histogram
// The metric will be automatically batched and sent by OTEL SDK
func (g *gotel) RecordHistogram(name string, value float64, labels map[string]string) {
	histogram := g.Histogram(name, labels)
	histogram.Record(value)
}

// GetRegistry returns the metrics registry for advanced usage
func (g *gotel) GetRegistry() *metrics.Registry {
	return g.metricsRegistry
}

// GetConfig returns the current configuration
func (g *gotel) GetConfig() *config.Config {
	return g.config
}

// Close gracefully shuts down the gotel client
func (g *gotel) Close() error {
	g.cancel()

	if g.config.EnableDebug {
		log.Println("Shutting down gotel client...")
	}

	// Force flush any remaining metrics before shutdown
	if err := g.otelClient.ForceFlush(); err != nil {
		log.Printf("Failed to flush metrics during shutdown: %v", err)
	}

	// Close OTEL client
	if err := g.otelClient.Close(); err != nil {
		return fmt.Errorf("failed to close OTEL client: %w", err)
	}

	if g.config.EnableDebug {
		log.Println("gotel client shut down successfully")
	}

	return nil
}
