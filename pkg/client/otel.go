package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/GetSimpl/gotel/pkg/config"
)

type Counter interface {
	Inc(labels map[string]string)
	Add(delta int64, labels map[string]string)
}

type Gauge interface {
	// Set sets the gauge to a specific value
	Set(value float64, labels map[string]string)
}

type Histogram interface {
	// Record records a value in the histogram
	Record(value float64, labels map[string]string)
}

type OTelClient interface {
	CreateCounter(name, unit string) (Counter, error)
	CreateGauge(name, unit string) (Gauge, error)
	CreateHistogram(name, unit string, buckets []float64) (Histogram, error)
	Close() error
}

// otelClient handles communication with OpenTelemetry Collector
// instrument creation is stateless; caching and mutexes are managed by metrics.Registry
type otelClient struct {
	config        *config.Config
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter
	exporter      *otlpmetrichttp.Exporter
	ctx           context.Context
	cancel        context.CancelFunc
	resource      *resource.Resource
}

type counter struct {
	ctx         context.Context
	otelCounter metric.Int64Counter
}

type gauge struct {
	ctx       context.Context
	otelGauge metric.Float64Gauge
}

type histogram struct {
	ctx           context.Context
	otelHistogram metric.Float64Histogram
}

// NewOtelClient creates a new OpenTelemetry client
func NewOtelClient(cfg *config.Config) (OTelClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure exporter options with sensible defaults
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.OtelEndpoint),
		otlpmetrichttp.WithTimeout(30 * time.Second), // Default timeout
	}

	if cfg.Environment == "local" {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	// Create OTLP exporter
	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create meter provider with configured internal
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(time.Second*time.Duration(cfg.SendInterval)),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create meter
	meter := meterProvider.Meter(cfg.ServiceName)

	client := &otelClient{
		config:        cfg,
		meterProvider: meterProvider,
		meter:         meter,
		exporter:      exporter,
		ctx:           ctx,
		cancel:        cancel,
		resource:      res,
	}

	// TODO: implement logger
	if cfg.EnableDebug {
		log.Printf("OTEL client initialized with endpoint: %s", cfg.OtelEndpoint)
		log.Printf("Send interval: %v", 30*time.Second)
	}

	return client, nil
}

// CreateCounter creates a new counter instrument
func (o *otelClient) CreateCounter(name, unit string) (Counter, error) {
	otelCounter, err := o.meter.Int64Counter(name, metric.WithUnit(unit))
	if err != nil {
		// TODO: add error log
		return nil, err
	}

	return &counter{ctx: o.ctx, otelCounter: otelCounter}, nil
}

// CreateGauge creates a new gauge instrument
func (o *otelClient) CreateGauge(name, unit string) (Gauge, error) {
	otelGauge, err := o.meter.Float64Gauge(name)
	if err != nil {
		// TODO: add error log
		return nil, err
	}

	return &gauge{ctx: o.ctx, otelGauge: otelGauge}, nil
}

// CreateHistogram creates a new histogram instrument
func (o *otelClient) CreateHistogram(name, unit string, buckets []float64) (Histogram, error) {
	otelHistogram, err := o.meter.Float64Histogram(name, metric.WithUnit(unit), metric.WithExplicitBucketBoundaries(buckets...))
	if err != nil {
		// TODO: add error log
		return nil, err
	}

	return &histogram{ctx: o.ctx, otelHistogram: otelHistogram}, nil
}

// Inc is Add(1)
func (c *counter) Inc(labels map[string]string) {
	attrs := labelsToAttributes(labels)
	c.otelCounter.Add(c.ctx, 1, metric.WithAttributes(attrs...))
}

// Add adds a delta to the counter
func (c *counter) Add(delta int64, labels map[string]string) {
	attrs := labelsToAttributes(labels)
	c.otelCounter.Add(c.ctx, delta, metric.WithAttributes(attrs...))
}

// Set sets gauge to the specified value
func (g *gauge) Set(value float64, labels map[string]string) {
	attrs := labelsToAttributes(labels)
	g.otelGauge.Record(g.ctx, value, metric.WithAttributes(attrs...))
}

// Record records a value in a histogram
func (h *histogram) Record(value float64, labels map[string]string) {
	attrs := labelsToAttributes(labels)
	h.otelHistogram.Record(h.ctx, value, metric.WithAttributes(attrs...))
}

// ForceFlush forces all pending metrics to be sent
func (o *otelClient) ForceFlush() error {
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second) // Default timeout
	defer cancel()

	return o.meterProvider.ForceFlush(ctx)
}

// Close gracefully shuts down the client
func (o *otelClient) Close() error {
	if o.config.EnableDebug {
		log.Println("Shutting down OTEL client")
	}

	// Force flush any remaining metrics
	if err := o.ForceFlush(); err != nil {
		log.Printf("Failed to flush metrics during shutdown: %v", err)
	}

	// Cancel context
	o.cancel()

	// Shutdown meter provider
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := o.meterProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown meter provider: %w", err)
	}

	return nil
}

// labelsToAttributes converts a map of labels to OTEL attributes
func labelsToAttributes(labels map[string]string) []attribute.KeyValue {
	if len(labels) == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, len(labels))
	for k, v := range labels {
		attrs = append(attrs, attribute.String(k, v))
	}
	return attrs
}
