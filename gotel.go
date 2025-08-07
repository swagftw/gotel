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
	"github.com/GetSimpl/gotel/pkg/logger"
)

// gotel is the main client for sending metrics to OpenTelemetry Collector
// OTEL SDK automatically handles batching, buffering, and reliable delivery
type gotel struct {
	config          *config.Config
	metricsRegistry metrics.Registry
	ctx             context.Context
	cancel          context.CancelFunc
}

type Gotel interface {
	IncrementCounter(name metrics.MetricName, unit metrics.Unit, labels map[string]string)
	AddToCounter(delta int64, name metrics.MetricName, unit metrics.Unit, labels map[string]string)
	SetGauge(value float64, name metrics.MetricName, unit metrics.Unit, labels map[string]string)
	RecordHistogram(value float64, name metrics.MetricName, unit metrics.Unit, buckets []float64, labels map[string]string)
	Close() error
}

// New creates a new gotel client with the provided configuration
func New(cfg *config.Config) (Gotel, error) {
	if cfg == nil {
		cfg = config.Default()
	}

	logger.InitLogger()

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
		metricsRegistry: registry,
		ctx:             ctx,
		cancel:          cancel,
	}

	if cfg.EnableDebug {
		logger.Logger.Info("gotel client initialized", "endpoint", cfg.OtelEndpoint)
		logger.Logger.Info("OTEL SDK will automatically batch and send metrics")
	}

	return g, nil
}

// IncrementCounter is a convenience method to increment a counter by 1
// The metric will be automatically batched and sent by OTEL SDK
func (g *gotel) IncrementCounter(name metrics.MetricName, unit metrics.Unit, labels map[string]string) {
	counter, err := g.metricsRegistry.GetOrCreateCounter(name, unit, g.addDefaultLabels(labels))
	if err != nil {
		return
	}

	counter.Inc()
}

func (g *gotel) AddToCounter(delta int64, name metrics.MetricName, unit metrics.Unit, labels map[string]string) {
	counter, err := g.metricsRegistry.GetOrCreateCounter(name, unit, g.addDefaultLabels(labels))
	if err != nil {
		return
	}

	counter.Add(delta)
}

// SetGauge is a convenience method to set a gauge value
// The metric will be automatically batched and sent by OTEL SDK
func (g *gotel) SetGauge(value float64, name metrics.MetricName, unit metrics.Unit, labels map[string]string) {
	gauge, err := g.metricsRegistry.GetOrCreateGauge(name, unit, g.addDefaultLabels(labels))
	if err != nil {
		return
	}

	gauge.Set(value)
}

// RecordHistogram is a convenience method to record a value in a histogram
// The metric will be automatically batched and sent by OTEL SDK
func (g *gotel) RecordHistogram(value float64, name metrics.MetricName, unit metrics.Unit, buckets []float64, labels map[string]string) {
	histogram, err := g.metricsRegistry.GetOrCreateHistogram(name, unit, buckets, g.addDefaultLabels(labels))
	if err != nil {
		return
	}

	histogram.Record(value)
}

func (g *gotel) addDefaultLabels(labels map[string]string) map[string]string {
	labelsCopy := make(map[string]string, len(labels)+2)
	for k, v := range labels {
		labelsCopy[k] = v
	}

	// Add default labels for service and environment
	labelsCopy["service.name"] = g.config.ServiceName
	labelsCopy["environment"] = g.config.Environment

	return labelsCopy
}

// Close gracefully shuts down the gotel client
func (g *gotel) Close() error {
	g.cancel()

	if g.config.EnableDebug {
		log.Println("Shutting down gotel client...")
	}

	// Force flush any remaining metrics before shutdown
	if err := g.metricsRegistry.Close(); err != nil {
		log.Printf("Failed to flush metrics during shutdown: %v", err)
	}

	if g.config.EnableDebug {
		log.Println("gotel client shut down successfully")
	}

	return nil
}
