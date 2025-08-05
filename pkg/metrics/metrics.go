package metrics

import (
	"context"
	"errors"
	"sync"

	"github.com/swagftw/gotel/pkg/client"
)

var (
	ErrCreatingMetric         = errors.New("failed to create metric")
	ErrHistBucketSizeTooLarge = errors.New("histogram bucket size is too large")
)

type (
	MetricName string
	Unit       string
)

const (
	MetricCounterHttpRequestsTotal MetricName = "http.server.requests.total"
	MetricHistHttpRequestDuration  MetricName = "http.server.request.duration"

	UnitPercent      Unit = "%"
	UnitSeconds      Unit = "s"
	UnitMilliseconds Unit = "ms"
	UnitBytes        Unit = "By"
	UnitRequest      Unit = "{request}"
)

// Counter represents a metrics counter that wraps OTEL counter
type Counter struct {
	name        MetricName
	unit        Unit
	labels      map[string]string
	otelCounter client.Counter
	ctx         context.Context
	value       int64
	mutex       sync.Mutex
}

// Gauge represents a gauge metric that wraps OTEL gauge
type Gauge struct {
	name      MetricName
	labels    map[string]string
	otelGauge client.Gauge // underlying sdk gauge
	ctx       context.Context
	value     float64
	mutex     sync.Mutex
}

// Histogram represents a histogram metric that wraps OTEL histogram
type Histogram struct {
	name          MetricName
	labels        map[string]string
	otelHistogram client.Histogram
	ctx           context.Context
	mutex         sync.Mutex
}

// registry holds all metrics and interfaces with OTEL client
type registry struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	otelClient client.OTelClient
	ctx        context.Context
	mutex      sync.RWMutex
}

// Registry is the public interface for metrics registry
type Registry interface {
	GetOrCreateCounter(name MetricName, unit Unit, labels map[string]string) (*Counter, error)
	GetOrCreateGauge(name MetricName, unit Unit, labels map[string]string) (*Gauge, error)
	GetOrCreateHistogram(name MetricName, unit Unit, buckets []float64, labels map[string]string) (*Histogram, error)
	Close() error
}

// NewRegistry creates a new metrics registry
func NewRegistry(otelClient client.OTelClient, ctx context.Context) Registry {
	return &registry{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		otelClient: otelClient,
		ctx:        ctx,
		mutex:      sync.RWMutex{},
	}
}

// GetOrCreateCounter gets an existing counter or creates a new one
func (r *registry) GetOrCreateCounter(name MetricName, unit Unit, labels map[string]string) (*Counter, error) {
	key := metricKey(string(name), labels)

	r.mutex.RLock()
	if counter, exists := r.counters[key]; exists {
		r.mutex.RUnlock()
		return counter, nil
	}
	r.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check again after acquiring write lock
	if counter, exists := r.counters[key]; exists {
		return counter, nil
	}

	// Create OTEL counter
	otelCounter, err := r.otelClient.CreateCounter(string(name), string(unit))
	if err != nil {
		return nil, ErrCreatingMetric
	}

	counter := &Counter{
		name:        name,
		unit:        unit,
		labels:      labels,
		otelCounter: otelCounter,
		ctx:         r.ctx,
		mutex:       sync.Mutex{},
		value:       0, // default value
	}

	r.counters[key] = counter

	return counter, nil
}

// GetOrCreateGauge gets an existing gauge or creates a new one
func (r *registry) GetOrCreateGauge(name MetricName, unit Unit, labels map[string]string) (*Gauge, error) {
	key := metricKey(string(name), labels)

	r.mutex.RLock()
	if gauge, exists := r.gauges[key]; exists {
		r.mutex.RUnlock()
		return gauge, nil
	}
	r.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check again after acquiring write lock
	if gauge, exists := r.gauges[key]; exists {
		return gauge, nil
	}

	// Create OTEL gauge
	otelGauge, err := r.otelClient.CreateGauge(string(name), string(unit))
	if err != nil {
		// Log error but don't fail - return a dummy gauge
		return nil, ErrCreatingMetric
	}

	gauge := &Gauge{
		name:      name,
		labels:    labels,
		otelGauge: otelGauge,
		ctx:       r.ctx,
		mutex:     sync.Mutex{},
		value:     0, // default value
	}

	r.gauges[key] = gauge

	return gauge, nil
}

// GetOrCreateHistogram gets an existing histogram or creates a new one
func (r *registry) GetOrCreateHistogram(name MetricName, unit Unit, buckets []float64, labels map[string]string) (*Histogram, error) {
	if len(buckets) > 20 {
		return nil, ErrHistBucketSizeTooLarge
	}

	key := metricKey(string(name), labels)

	r.mutex.RLock()
	if histogram, exists := r.histograms[key]; exists {
		r.mutex.RUnlock()
		return histogram, nil
	}
	r.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check again after acquiring write lock
	if histogram, exists := r.histograms[key]; exists {
		return histogram, nil
	}

	// Create OTEL histogram
	otelHistogram, err := r.otelClient.CreateHistogram(string(name), string(unit), buckets)
	if err != nil {
		return nil, ErrCreatingMetric
	}

	histogram := &Histogram{
		name:          name,
		labels:        labels,
		otelHistogram: otelHistogram,
		ctx:           r.ctx,
		mutex:         sync.Mutex{},
	}

	r.histograms[key] = histogram

	return histogram, nil
}

func (r *registry) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Close all OTEL clients if needed
	if r.otelClient != nil {
		if err := r.otelClient.Close(); err != nil {
			return err
		}
	}

	// Clear all metrics
	r.counters = make(map[string]*Counter)
	r.gauges = make(map[string]*Gauge)
	r.histograms = make(map[string]*Histogram)

	return nil
}

// Inc increments the counter by 1 and returns the new value
func (c *Counter) Inc() int64 {
	return c.Add(1)
}

// Add adds the given value to the counter and returns the new value
func (c *Counter) Add(delta int64) int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.value += delta
	newValue := c.value

	// Record to OTEL
	if c.otelCounter != nil {
		c.otelCounter.Add(delta, c.labels)
	}

	return newValue
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.value = value

	// Record to OTEL
	if g.otelGauge != nil {
		g.otelGauge.Set(value, g.labels)
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
	defer g.mutex.Unlock()

	g.value += delta

	// Record to OTEL
	if g.otelGauge != nil {
		g.otelGauge.Set(g.value, g.labels)
	}
}

// Record records a value for the histogram
func (h *Histogram) Record(value float64) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.otelHistogram != nil {
		h.otelHistogram.Record(value, h.labels)
	}
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
