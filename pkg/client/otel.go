package client

import (
	"context"
	"fmt"
	"log"
	"sync"
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

// OtelClient handles communication with OpenTelemetry Collector
type OtelClient struct {
	config        *config.Config
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter
	exporter      *otlpmetrichttp.Exporter
	counters      map[string]metric.Int64Counter
	gauges        map[string]metric.Float64Gauge
	histograms    map[string]metric.Float64Histogram
	mutex         sync.RWMutex
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
				sdkmetric.WithInterval(time.Duration(cfg.SendInterval)),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create meter
	meter := meterProvider.Meter(cfg.ServiceName)

	client := &OtelClient{
		config:        cfg,
		meterProvider: meterProvider,
		meter:         meter,
		exporter:      exporter,
		counters:      make(map[string]metric.Int64Counter),
		gauges:        make(map[string]metric.Float64Gauge),
		histograms:    make(map[string]metric.Float64Histogram),
		ctx:           ctx,
		cancel:        cancel,
		resource:      res,
	}

	if cfg.EnableDebug {
		log.Printf("OTEL client initialized with endpoint: %s", cfg.OtelEndpoint)
		log.Printf("Send interval: %v", 30*time.Second)
	}

	return client, nil
}

// GetOrCreateCounter gets an existing counter or creates a new one
func (o *OtelClient) GetOrCreateCounter(name string, labels map[string]string) (metric.Int64Counter, error) {
	key := metricKey(name, labels)

	// Try to get existing counter with read lock
	o.mutex.RLock()
	if counter, exists := o.counters[key]; exists {
		o.mutex.RUnlock()
		return counter, nil
	}
	o.mutex.RUnlock()

	// Need to create new counter, acquire write lock
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Double-check in case another goroutine created it while we waited
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

	// Try to get existing gauge with read lock
	o.mutex.RLock()
	if gauge, exists := o.gauges[key]; exists {
		o.mutex.RUnlock()
		return gauge, nil
	}
	o.mutex.RUnlock()

	// Need to create new gauge, acquire write lock
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Double-check in case another goroutine created it while we waited
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
	// Create a copy of labels to avoid modifying the original map
	labelsCopy := make(map[string]string, len(labels)+2)
	for k, v := range labels {
		labelsCopy[k] = v
	}
	labelsCopy["service_name"] = o.config.ServiceName
	labelsCopy["service_environment"] = o.config.Environment

	counter, err := o.GetOrCreateCounter(name, labelsCopy)
	if err != nil {
		return err
	}

	attrs := labelsToAttributes(labelsCopy)
	counter.Add(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// SetGauge sets a gauge to the specified value
func (o *OtelClient) SetGauge(name string, labels map[string]string, value float64) error {
	// Create a copy of labels to avoid modifying the original map
	labelsCopy := make(map[string]string, len(labels)+2)
	for k, v := range labels {
		labelsCopy[k] = v
	}
	labelsCopy["service_name"] = o.config.ServiceName
	labelsCopy["service_environment"] = o.config.Environment

	gauge, err := o.GetOrCreateGauge(name, labelsCopy)
	if err != nil {
		return err
	}

	attrs := labelsToAttributes(labelsCopy)
	gauge.Record(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// GetOrCreateHistogram gets an existing histogram or creates a new one
func (o *OtelClient) GetOrCreateHistogram(name string, labels map[string]string) (metric.Float64Histogram, error) {
	key := metricKey(name, labels)

	// Try to get existing histogram with read lock
	o.mutex.RLock()
	if histogram, exists := o.histograms[key]; exists {
		o.mutex.RUnlock()
		return histogram, nil
	}
	o.mutex.RUnlock()

	// Need to create new histogram, acquire write lock
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Double-check in case another goroutine created it while we waited
	if histogram, exists := o.histograms[key]; exists {
		return histogram, nil
	}

	histogram, err := o.meter.Float64Histogram(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram %s: %w", name, err)
	}

	o.histograms[key] = histogram
	return histogram, nil
}

// RecordHistogram records a value in a histogram
func (o *OtelClient) RecordHistogram(name string, labels map[string]string, value float64) error {
	// Create a copy of labels to avoid modifying the original map
	labelsCopy := make(map[string]string, len(labels)+2)
	for k, v := range labels {
		labelsCopy[k] = v
	}
	labelsCopy["service_name"] = o.config.ServiceName
	labelsCopy["service_environment"] = o.config.Environment

	histogram, err := o.GetOrCreateHistogram(name, labelsCopy)
	if err != nil {
		return err
	}

	attrs := labelsToAttributes(labelsCopy)
	histogram.Record(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// ForceFlush forces all pending metrics to be sent
func (o *OtelClient) ForceFlush() error {
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second) // Default timeout
	defer cancel()

	return o.meterProvider.ForceFlush(ctx)
}

// Close gracefully shuts down the client
func (o *OtelClient) Close() error {
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

// GetStats returns client statistics
func (o *OtelClient) GetStats() map[string]interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	return map[string]interface{}{
		"otel_endpoint":    o.config.OtelEndpoint,
		"counters_count":   len(o.counters),
		"gauges_count":     len(o.gauges),
		"histograms_count": len(o.histograms),
		"service_name":     o.config.ServiceName,
		"service_version":  o.config.ServiceVersion,
		"environment":      o.config.Environment,
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
