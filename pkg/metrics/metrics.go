package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Counter represents a metrics counter that wraps OTEL counter
type Counter struct {
	name        string
	labels      map[string]string
	otelCounter metric.Int64Counter
	ctx         context.Context
	attributes  []attribute.KeyValue
	value       int64
	mutex       sync.RWMutex
}

// NewCounter creates a new counter metric
func NewCounter(name string, labels map[string]string, otelCounter metric.Int64Counter, ctx context.Context) *Counter {
	attrs := labelsToAttributes(labels)
	return &Counter{
		name:        name,
		labels:      labels,
		otelCounter: otelCounter,
		ctx:         ctx,
		attributes:  attrs,
		value:       0,
	}
}

// Inc increments the counter by 1 and returns the new value
func (c *Counter) Inc() int64 {
	return c.Add(1)
}

// Add adds the given value to the counter and returns the new value
func (c *Counter) Add(delta int64) int64 {
	c.mutex.Lock()
	c.value += delta
	newValue := c.value
	c.mutex.Unlock()

	// Record to OTEL
	if c.otelCounter != nil {
		c.otelCounter.Add(c.ctx, delta, metric.WithAttributes(c.attributes...))
	}

	return newValue
}

// Get returns the current value of the counter
func (c *Counter) Get() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.value
}

// Gauge represents a gauge metric that wraps OTEL gauge
type Gauge struct {
	name       string
	labels     map[string]string
	otelGauge  metric.Float64Gauge
	ctx        context.Context
	attributes []attribute.KeyValue
	value      float64
	mutex      sync.RWMutex
}

// NewGauge creates a new gauge metric
func NewGauge(name string, labels map[string]string, otelGauge metric.Float64Gauge, ctx context.Context) *Gauge {
	attrs := labelsToAttributes(labels)
	return &Gauge{
		name:       name,
		labels:     labels,
		otelGauge:  otelGauge,
		ctx:        ctx,
		attributes: attrs,
		value:      0,
	}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	g.mutex.Lock()
	g.value = value
	g.mutex.Unlock()

	// Record to OTEL
	if g.otelGauge != nil {
		g.otelGauge.Record(g.ctx, value, metric.WithAttributes(g.attributes...))
	}
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	g.Add(1.0)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	g.Add(-1.0)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(delta float64) {
	g.mutex.Lock()
	g.value += delta
	newValue := g.value
	g.mutex.Unlock()

	// Record to OTEL
	if g.otelGauge != nil {
		g.otelGauge.Record(g.ctx, newValue, metric.WithAttributes(g.attributes...))
	}
}

// Get returns the current value of the gauge
func (g *Gauge) Get() float64 {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.value
}

// Registry holds all metrics and interfaces with OTEL client
type Registry struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	otelClient OtelClient
	ctx        context.Context
	mutex      sync.RWMutex
}

// OtelClient interface for dependency injection
type OtelClient interface {
	GetOrCreateCounter(name string, labels map[string]string) (metric.Int64Counter, error)
	GetOrCreateGauge(name string, labels map[string]string) (metric.Float64Gauge, error)
}

// NewRegistry creates a new metrics registry
func NewRegistry(otelClient OtelClient, ctx context.Context) *Registry {
	return &Registry{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		otelClient: otelClient,
		ctx:        ctx,
	}
}

// GetOrCreateCounter gets an existing counter or creates a new one
func (r *Registry) GetOrCreateCounter(name string, labels map[string]string) *Counter {
	key := metricKey(name, labels)

	r.mutex.RLock()
	if counter, exists := r.counters[key]; exists {
		r.mutex.RUnlock()
		return counter
	}
	r.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check again after acquiring write lock
	if counter, exists := r.counters[key]; exists {
		return counter
	}

	// Create OTEL counter
	otelCounter, err := r.otelClient.GetOrCreateCounter(name, labels)
	if err != nil {
		// Log error but don't fail - return a dummy counter
		counter := &Counter{
			name:   name,
			labels: labels,
			ctx:    r.ctx,
			value:  0,
		}
		r.counters[key] = counter
		return counter
	}

	counter := NewCounter(name, labels, otelCounter, r.ctx)
	r.counters[key] = counter
	return counter
}

// GetOrCreateGauge gets an existing gauge or creates a new one
func (r *Registry) GetOrCreateGauge(name string, labels map[string]string) *Gauge {
	key := metricKey(name, labels)

	r.mutex.RLock()
	if gauge, exists := r.gauges[key]; exists {
		r.mutex.RUnlock()
		return gauge
	}
	r.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check again after acquiring write lock
	if gauge, exists := r.gauges[key]; exists {
		return gauge
	}

	// Create OTEL gauge
	otelGauge, err := r.otelClient.GetOrCreateGauge(name, labels)
	if err != nil {
		// Log error but don't fail - return a dummy gauge
		gauge := &Gauge{
			name:   name,
			labels: labels,
			ctx:    r.ctx,
			value:  0,
		}
		r.gauges[key] = gauge
		return gauge
	}

	gauge := NewGauge(name, labels, otelGauge, r.ctx)
	r.gauges[key] = gauge
	return gauge
}

// MetricsCount returns the total number of registered metrics
func (r *Registry) MetricsCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.counters) + len(r.gauges)
}

// GetCounter returns a counter by name
func (r *Registry) GetCounter(name string) (*Counter, bool) {
	key := metricKey(name, nil)
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Try exact match first
	if counter, exists := r.counters[key]; exists {
		return counter, exists
	}

	// If not found, try finding any counter with that name
	for _, counter := range r.counters {
		if counter.name == name {
			return counter, true
		}
	}

	return nil, false
}

// GetGauge returns a gauge by name
func (r *Registry) GetGauge(name string) (*Gauge, bool) {
	key := metricKey(name, nil)
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Try exact match first
	if gauge, exists := r.gauges[key]; exists {
		return gauge, exists
	}

	// If not found, try finding any gauge with that name
	for _, gauge := range r.gauges {
		if gauge.name == name {
			return gauge, true
		}
	}

	return nil, false
}

// Collect returns basic metrics information (for compatibility)
func (r *Registry) Collect() []MetricInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var metrics []MetricInfo

	for _, counter := range r.counters {
		metrics = append(metrics, MetricInfo{
			Name:   counter.name,
			Type:   "counter",
			Value:  float64(counter.Get()),
			Labels: counter.labels,
		})
	}

	for _, gauge := range r.gauges {
		metrics = append(metrics, MetricInfo{
			Name:   gauge.name,
			Type:   "gauge",
			Value:  gauge.Get(),
			Labels: gauge.labels,
		})
	}

	return metrics
}

// MetricInfo represents basic metric information
type MetricInfo struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
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
