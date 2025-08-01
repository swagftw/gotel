package metrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/metric"
)

// Mock OTEL client for testing
type mockOtelClient struct {
	counters map[string]metric.Int64Counter
	gauges   map[string]metric.Float64Gauge
}

func (m *mockOtelClient) GetOrCreateCounter(name string, labels map[string]string) (metric.Int64Counter, error) {
	key := metricKey(name, labels)
	if counter, exists := m.counters[key]; exists {
		return counter, nil
	}
	// Return nil for testing - we just want to test the registry logic
	return nil, nil
}

func (m *mockOtelClient) GetOrCreateGauge(name string, labels map[string]string) (metric.Float64Gauge, error) {
	key := metricKey(name, labels)
	if gauge, exists := m.gauges[key]; exists {
		return gauge, nil
	}
	// Return nil for testing - we just want to test the registry logic
	return nil, nil
}

func newMockOtelClient() *mockOtelClient {
	return &mockOtelClient{
		counters: make(map[string]metric.Int64Counter),
		gauges:   make(map[string]metric.Float64Gauge),
	}
}

func TestRegistry(t *testing.T) {
	mockClient := newMockOtelClient()
	ctx := context.Background()
	registry := NewRegistry(mockClient, ctx)

	// Test GetOrCreateCounter
	counter := registry.GetOrCreateCounter("test_counter", map[string]string{"method": "GET"})
	if counter == nil {
		t.Error("Expected counter to be created")
	}

	// Test GetOrCreateGauge
	gauge := registry.GetOrCreateGauge("test_gauge", map[string]string{"cpu": "0"})
	if gauge == nil {
		t.Error("Expected gauge to be created")
	}

	// Test GetCounter
	retrievedCounter, exists := registry.GetCounter("test_counter")
	if !exists {
		t.Error("Expected counter to exist in registry")
	}
	if retrievedCounter == nil {
		t.Error("Expected retrieved counter to not be nil")
	}

	// Test GetGauge
	retrievedGauge, exists := registry.GetGauge("test_gauge")
	if !exists {
		t.Error("Expected gauge to exist in registry")
	}
	if retrievedGauge == nil {
		t.Error("Expected retrieved gauge to not be nil")
	}

	// Test MetricsCount
	count := registry.MetricsCount()
	if count != 2 {
		t.Errorf("Expected 2 metrics, got %d", count)
	}

	// Test Collect
	metrics := registry.Collect()
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics in collection, got %d", len(metrics))
	}
}

func TestCounter(t *testing.T) {
	ctx := context.Background()
	counter := NewCounter("test_counter", map[string]string{"label1": "value1"}, nil, ctx)

	// Test initial value
	if counter.Get() != 0 {
		t.Errorf("Expected initial counter value to be 0, got %d", counter.Get())
	}

	// Test Inc
	val := counter.Inc()
	if val != 1 {
		t.Errorf("Expected counter value to be 1 after Inc(), got %d", val)
	}

	// Test Add
	val = counter.Add(5)
	if val != 6 {
		t.Errorf("Expected counter value to be 6 after Add(5), got %d", val)
	}

	// Test Get
	if counter.Get() != 6 {
		t.Errorf("Expected counter value to be 6, got %d", counter.Get())
	}
}

func TestGauge(t *testing.T) {
	ctx := context.Background()
	gauge := NewGauge("test_gauge", map[string]string{"instance": "localhost"}, nil, ctx)

	const tolerance = 0.001 // tolerance for floating point comparison

	// Test initial value
	if gauge.Get() != 0.0 {
		t.Errorf("Expected initial gauge value to be 0.0, got %f", gauge.Get())
	}

	// Test Set
	gauge.Set(3.14)
	if abs(gauge.Get()-3.14) > tolerance {
		t.Errorf("Expected gauge value to be 3.14, got %f", gauge.Get())
	}

	// Test Inc
	gauge.Inc()
	if abs(gauge.Get()-4.14) > tolerance {
		t.Errorf("Expected gauge value to be 4.14 after Inc(), got %f", gauge.Get())
	}

	// Test Dec
	gauge.Dec()
	if abs(gauge.Get()-3.14) > tolerance {
		t.Errorf("Expected gauge value to be 3.14 after Dec(), got %f", gauge.Get())
	}

	// Test Add
	gauge.Add(1.86)
	if abs(gauge.Get()-5.0) > tolerance {
		t.Errorf("Expected gauge value to be 5.0 after Add(1.86), got %f", gauge.Get())
	}
}

// abs returns the absolute value of x
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	counter := NewCounter("concurrent_test", map[string]string{}, nil, ctx)
	gauge := NewGauge("concurrent_gauge", map[string]string{}, nil, ctx)

	// Test concurrent access to counter
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			counter.Inc()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	if counter.Get() != 100 {
		t.Errorf("Expected counter value to be 100, got %d", counter.Get())
	}

	// Test concurrent access to gauge
	for i := 0; i < 100; i++ {
		go func(val float64) {
			gauge.Set(val)
			done <- true
		}(float64(i))
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Gauge value should be one of the set values (race condition expected)
	val := gauge.Get()
	if val < 0 || val >= 100 {
		t.Errorf("Expected gauge value to be between 0 and 99, got %f", val)
	}
}
