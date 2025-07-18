package metrics

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// Counter represents an atomic counter metric
type Counter struct {
	name   string
	labels map[string]string
	value  int64
}

// NewCounter creates a new counter metric
func NewCounter(name string, labels map[string]string) *Counter {
	return &Counter{
		name:   name,
		labels: labels,
		value:  0,
	}
}

// Inc increments the counter by 1 and returns the new value
func (c *Counter) Inc() int64 {
	return atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter and returns the new value
func (c *Counter) Add(delta int64) int64 {
	return atomic.AddInt64(&c.value, delta)
}

// Get returns the current value of the counter
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// ToTimeSeries converts the counter to a Prometheus TimeSeries
func (c *Counter) ToTimeSeries(timestamp time.Time) prompb.TimeSeries {
	labels := make([]prompb.Label, 0, len(c.labels)+1)

	// Add the metric name
	labels = append(labels, prompb.Label{
		Name:  "__name__",
		Value: c.name,
	})

	// Add custom labels
	for k, v := range c.labels {
		labels = append(labels, prompb.Label{
			Name:  k,
			Value: v,
		})
	}

	return prompb.TimeSeries{
		Labels: labels,
		Samples: []prompb.Sample{
			{
				Value:     float64(c.Get()),
				Timestamp: timestamp.UnixMilli(),
			},
		},
	}
}

// Gauge represents a gauge metric that can go up and down
type Gauge struct {
	name   string
	labels map[string]string
	value  int64 // Using int64 for atomic operations, will convert to float64 when needed
}

// NewGauge creates a new gauge metric
func NewGauge(name string, labels map[string]string) *Gauge {
	return &Gauge{
		name:   name,
		labels: labels,
		value:  0,
	}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	atomic.StoreInt64(&g.value, int64(value*1000)) // Store as millis for precision
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	atomic.AddInt64(&g.value, 1000) // Add 1000 millis = 1.0
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	atomic.AddInt64(&g.value, -1000) // Subtract 1000 millis = 1.0
}

// Add adds the given value to the gauge
func (g *Gauge) Add(delta float64) {
	atomic.AddInt64(&g.value, int64(delta*1000))
}

// Get returns the current value of the gauge
func (g *Gauge) Get() float64 {
	return float64(atomic.LoadInt64(&g.value)) / 1000.0
}

// ToTimeSeries converts the gauge to a Prometheus TimeSeries
func (g *Gauge) ToTimeSeries(timestamp time.Time) prompb.TimeSeries {
	labels := make([]prompb.Label, 0, len(g.labels)+1)

	// Add the metric name
	labels = append(labels, prompb.Label{
		Name:  "__name__",
		Value: g.name,
	})

	// Add custom labels
	for k, v := range g.labels {
		labels = append(labels, prompb.Label{
			Name:  k,
			Value: v,
		})
	}

	return prompb.TimeSeries{
		Labels: labels,
		Samples: []prompb.Sample{
			{
				Value:     g.Get(),
				Timestamp: timestamp.UnixMilli(),
			},
		},
	}
}

// Registry holds all metrics and can export them
type Registry struct {
	counters map[string]*Counter
	gauges   map[string]*Gauge
}

// NewRegistry creates a new metrics registry
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
	}
}

// NewCounter creates and registers a new counter
func (r *Registry) NewCounter(name string, labels map[string]string) *Counter {
	counter := NewCounter(name, labels)
	r.counters[name] = counter
	return counter
}

// NewGauge creates and registers a new gauge
func (r *Registry) NewGauge(name string, labels map[string]string) *Gauge {
	gauge := NewGauge(name, labels)
	r.gauges[name] = gauge
	return gauge
}

// Export exports all metrics as Prometheus TimeSeries
func (r *Registry) Export() []prompb.TimeSeries {
	// Use precise timestamp to avoid duplicates with 1ms rate limiting
	now := time.Now()
	var timeSeries []prompb.TimeSeries

	// Export counters
	for _, counter := range r.counters {
		timeSeries = append(timeSeries, counter.ToTimeSeries(now))
	}

	// Export gauges
	for _, gauge := range r.gauges {
		timeSeries = append(timeSeries, gauge.ToTimeSeries(now))
	}

	return timeSeries
}

// GetCounter returns a counter by name
func (r *Registry) GetCounter(name string) (*Counter, bool) {
	counter, exists := r.counters[name]
	return counter, exists
}

// GetGauge returns a gauge by name
func (r *Registry) GetGauge(name string) (*Gauge, bool) {
	gauge, exists := r.gauges[name]
	return gauge, exists
}

// GetOrCreateCounter gets an existing counter or creates a new one
func (r *Registry) GetOrCreateCounter(name string, labels map[string]string) *Counter {
	key := metricKey(name, labels)
	if counter, exists := r.counters[key]; exists {
		return counter
	}
	counter := NewCounter(name, labels)
	r.counters[key] = counter
	return counter
}

// GetOrCreateGauge gets an existing gauge or creates a new one
func (r *Registry) GetOrCreateGauge(name string, labels map[string]string) *Gauge {
	key := metricKey(name, labels)
	if gauge, exists := r.gauges[key]; exists {
		return gauge
	}
	gauge := NewGauge(name, labels)
	r.gauges[key] = gauge
	return gauge
}

// Collect exports all metrics as Prometheus TimeSeries (alias for Export)
func (r *Registry) Collect() []prompb.TimeSeries {
	return r.Export()
}

// MetricsCount returns the total number of registered metrics
func (r *Registry) MetricsCount() int {
	return len(r.counters) + len(r.gauges)
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
