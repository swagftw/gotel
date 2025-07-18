package metrics

import (
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	counter := NewCounter("test_counter", map[string]string{
		"label1": "value1",
		"label2": "value2",
	})

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

func TestCounterToTimeSeries(t *testing.T) {
	counter := NewCounter("test_counter", map[string]string{
		"service": "test",
		"method":  "GET",
	})

	counter.Add(10)

	timestamp := time.Now()
	ts := counter.ToTimeSeries(timestamp)

	// Check metric name
	if ts.Labels[0].Name != "__name__" || ts.Labels[0].Value != "test_counter" {
		t.Errorf("Expected metric name to be 'test_counter', got %s", ts.Labels[0].Value)
	}

	// Check labels count (name + 2 custom labels)
	if len(ts.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(ts.Labels))
	}

	// Check sample value
	if len(ts.Samples) != 1 {
		t.Errorf("Expected 1 sample, got %d", len(ts.Samples))
	}

	if ts.Samples[0].Value != 10.0 {
		t.Errorf("Expected sample value to be 10.0, got %f", ts.Samples[0].Value)
	}

	// Check timestamp
	expectedTimestamp := timestamp.UnixMilli()
	if ts.Samples[0].Timestamp != expectedTimestamp {
		t.Errorf("Expected timestamp %d, got %d", expectedTimestamp, ts.Samples[0].Timestamp)
	}
}

func TestGauge(t *testing.T) {
	gauge := NewGauge("test_gauge", map[string]string{
		"instance": "localhost",
	})

	// Test initial value
	if gauge.Get() != 0.0 {
		t.Errorf("Expected initial gauge value to be 0.0, got %f", gauge.Get())
	}

	// Test Set
	gauge.Set(3.14)
	if gauge.Get() != 3.14 {
		t.Errorf("Expected gauge value to be 3.14, got %f", gauge.Get())
	}

	// Test Inc
	gauge.Inc()
	if gauge.Get() != 4.14 {
		t.Errorf("Expected gauge value to be 4.14 after Inc(), got %f", gauge.Get())
	}

	// Test Dec
	gauge.Dec()
	if gauge.Get() != 3.14 {
		t.Errorf("Expected gauge value to be 3.14 after Dec(), got %f", gauge.Get())
	}

	// Test Add
	gauge.Add(1.86)
	if gauge.Get() != 5.0 {
		t.Errorf("Expected gauge value to be 5.0 after Add(1.86), got %f", gauge.Get())
	}
}

func TestGaugeToTimeSeries(t *testing.T) {
	gauge := NewGauge("test_gauge", map[string]string{
		"instance": "localhost",
	})

	gauge.Set(42.5)

	timestamp := time.Now()
	ts := gauge.ToTimeSeries(timestamp)

	// Check metric name
	if ts.Labels[0].Name != "__name__" || ts.Labels[0].Value != "test_gauge" {
		t.Errorf("Expected metric name to be 'test_gauge', got %s", ts.Labels[0].Value)
	}

	// Check sample value
	if ts.Samples[0].Value != 42.5 {
		t.Errorf("Expected sample value to be 42.5, got %f", ts.Samples[0].Value)
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test NewCounter
	counter := registry.NewCounter("http_requests", map[string]string{"method": "GET"})
	if counter == nil {
		t.Error("Expected counter to be created")
	}

	// Test NewGauge
	gauge := registry.NewGauge("cpu_usage", map[string]string{"cpu": "0"})
	if gauge == nil {
		t.Error("Expected gauge to be created")
	}

	// Test GetCounter
	retrievedCounter, exists := registry.GetCounter("http_requests")
	if !exists {
		t.Error("Expected counter to exist in registry")
	}
	if retrievedCounter != counter {
		t.Error("Expected retrieved counter to be the same instance")
	}

	// Test GetGauge
	retrievedGauge, exists := registry.GetGauge("cpu_usage")
	if !exists {
		t.Error("Expected gauge to exist in registry")
	}
	if retrievedGauge != gauge {
		t.Error("Expected retrieved gauge to be the same instance")
	}

	// Test Export
	counter.Add(10)
	gauge.Set(75.5)

	timeSeries := registry.Export()
	if len(timeSeries) != 2 {
		t.Errorf("Expected 2 time series, got %d", len(timeSeries))
	}
}

func TestConcurrentAccess(t *testing.T) {
	counter := NewCounter("concurrent_test", map[string]string{})
	gauge := NewGauge("concurrent_gauge", map[string]string{})

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
