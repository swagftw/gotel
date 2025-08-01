package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/GetSimpl/gotel/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// OtelClient handles communication with OpenTelemetry Collector
type OtelClient struct {
	config        *config.Config
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter
	exporter      *otlpmetrichttp.Exporter
	counters      map[string]metric.Int64Counter
	gauges        map[string]metric.Float64Gauge
	ctx           context.Context
	cancel        context.CancelFunc
	resource      *resource.Resource
}

// NewOtelClient creates a new OpenTelemetry client
func NewOtelClient(cfg *config.Config) (*OtelClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.AppName),
			semconv.ServiceVersion(cfg.AppVersion),
			semconv.ServiceInstanceID(cfg.Instance),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure exporter options
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.OtelEndpoint),
		otlpmetrichttp.WithTimeout(cfg.HTTPTimeout),
	}

	// Add insecure option if specified
	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	// Add headers if specified
	if len(cfg.OtelHeaders) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.OtelHeaders))
	}

	// Create OTLP exporter
	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(cfg.SendInterval),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create meter
	meter := meterProvider.Meter("gotel")

	client := &OtelClient{
		config:        cfg,
		meterProvider: meterProvider,
		meter:         meter,
		exporter:      exporter,
		counters:      make(map[string]metric.Int64Counter),
		gauges:        make(map[string]metric.Float64Gauge),
		ctx:           ctx,
		cancel:        cancel,
		resource:      res,
	}

	if cfg.LogLevel == "debug" || cfg.EnableDebug {
		log.Printf("OTEL client initialized with endpoint: %s", cfg.OtelEndpoint)
		log.Printf("Send interval: %v", cfg.SendInterval)
	}

	return client, nil
}

// GetOrCreateCounter gets an existing counter or creates a new one
func (o *OtelClient) GetOrCreateCounter(name string, labels map[string]string) (metric.Int64Counter, error) {
	key := metricKey(name, labels)

	if counter, exists := o.counters[key]; exists {
		return counter, nil
	}

	counter, err := o.meter.Int64Counter(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter %s: %w", name, err)
	}

	o.counters[key] = counter
	return counter, nil
}

// GetOrCreateGauge gets an existing gauge or creates a new one
func (o *OtelClient) GetOrCreateGauge(name string, labels map[string]string) (metric.Float64Gauge, error) {
	key := metricKey(name, labels)

	if gauge, exists := o.gauges[key]; exists {
		return gauge, nil
	}

	gauge, err := o.meter.Float64Gauge(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create gauge %s: %w", name, err)
	}

	o.gauges[key] = gauge
	return gauge, nil
}

// IncrementCounter increments a counter by the specified amount
func (o *OtelClient) IncrementCounter(name string, labels map[string]string, value int64) error {
	counter, err := o.GetOrCreateCounter(name, labels)
	if err != nil {
		return err
	}

	attrs := labelsToAttributes(labels)
	counter.Add(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// SetGauge sets a gauge to the specified value
func (o *OtelClient) SetGauge(name string, labels map[string]string, value float64) error {
	gauge, err := o.GetOrCreateGauge(name, labels)
	if err != nil {
		return err
	}

	attrs := labelsToAttributes(labels)
	gauge.Record(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// ForceFlush forces all pending metrics to be sent
func (o *OtelClient) ForceFlush() error {
	ctx, cancel := context.WithTimeout(o.ctx, o.config.HTTPTimeout)
	defer cancel()

	return o.meterProvider.ForceFlush(ctx)
}

// Close gracefully shuts down the client
func (o *OtelClient) Close() error {
	if o.config.LogLevel == "debug" || o.config.EnableDebug {
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

// GetStats returns client statistics
func (o *OtelClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"otel_endpoint":    o.config.OtelEndpoint,
		"timeout":          o.config.HTTPTimeout.String(),
		"send_interval":    o.config.SendInterval.String(),
		"counters_count":   len(o.counters),
		"gauges_count":     len(o.gauges),
		"service_name":     o.config.AppName,
		"service_version":  o.config.AppVersion,
		"service_instance": o.config.Instance,
		"environment":      o.config.Environment,
		"insecure":         o.config.Insecure,
	}
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

// metricKey creates a unique key for a metric based on name and labels
func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	key := name + "{"
	first := true
	for k, v := range labels {
		if !first {
			key += ","
		}
		key += k + "=" + v
		first = false
	}
	key += "}"
	return key
}
