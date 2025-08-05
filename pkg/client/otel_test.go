package client

import (
	"testing"

	"github.com/GetSimpl/gotel/pkg/config"
)

func TestNewOtelClient(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Error("Expected client to be created")
		return
	}

	if client.config != cfg {
		t.Error("Expected client config to match provided config")
	}

	if client.meter == nil {
		t.Error("Expected meter to be initialized")
	}

	if client.meterProvider == nil {
		t.Error("Expected meter provider to be initialized")
	}
}

func TestOtelClient_GetOrCreateCounter(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	labels := map[string]string{
		"service": "test",
		"method":  "GET",
	}

	counter1, err := client.GetOrCreateCounter("test_counter", labels)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	if counter1 == nil {
		t.Error("Expected counter to be created")
	}

	// Get the same counter again
	counter2, err := client.GetOrCreateCounter("test_counter", labels)
	if err != nil {
		t.Fatalf("Failed to get existing counter: %v", err)
	}

	// Should be the same instance
	if counter1 != counter2 {
		t.Error("Expected to get the same counter instance")
	}
}

func TestOtelClient_GetOrCreateGauge(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	labels := map[string]string{
		"instance": "localhost",
	}

	gauge1, err := client.GetOrCreateGauge("test_gauge", labels)
	if err != nil {
		t.Fatalf("Failed to create gauge: %v", err)
	}

	if gauge1 == nil {
		t.Error("Expected gauge to be created")
	}

	// Get the same gauge again
	gauge2, err := client.GetOrCreateGauge("test_gauge", labels)
	if err != nil {
		t.Fatalf("Failed to get existing gauge: %v", err)
	}

	// Should be the same instance
	if gauge1 != gauge2 {
		t.Error("Expected to get the same gauge instance")
	}
}

func TestOtelClient_IncrementCounter(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	labels := map[string]string{
		"test": "counter",
	}

	err = client.IncrementCounter("test_requests", labels, 5)
	if err != nil {
		t.Fatalf("Failed to increment counter: %v", err)
	}

	// Increment again
	err = client.IncrementCounter("test_requests", labels, 3)
	if err != nil {
		t.Fatalf("Failed to increment counter again: %v", err)
	}
}

func TestOtelClient_SetGauge(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	labels := map[string]string{
		"test": "gauge",
	}

	err = client.SetGauge("test_value", labels, 42.5)
	if err != nil {
		t.Fatalf("Failed to set gauge: %v", err)
	}

	// Set again with different value
	err = client.SetGauge("test_value", labels, 100.0)
	if err != nil {
		t.Fatalf("Failed to set gauge again: %v", err)
	}
}

func TestOtelClient_ForceFlush(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	// Add some metrics
	err = client.IncrementCounter("flush_test", nil, 1)
	if err != nil {
		t.Fatalf("Failed to increment counter: %v", err)
	}

	// Force flush - expect failure since no collector is running
	err = client.ForceFlush()
	// Don't fail test since collector isn't running
	t.Logf("Force flush result (expected to fail): %v", err)
}

func TestOtelClient_GetStats(t *testing.T) {
	cfg := config.Default()
	cfg.OtelEndpoint = "localhost:4318"
	cfg.ServiceName = "test-app"
	cfg.ServiceVersion = "1.0.0"
	cfg.Environment = "test"
	cfg.Insecure = true

	client, err := NewOtelClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OTEL client: %v", err)
	}
	defer client.Close()

	stats := client.GetStats()

	if stats["otel_endpoint"] != cfg.OtelEndpoint {
		t.Errorf("Expected endpoint %s, got %s", cfg.OtelEndpoint, stats["otel_endpoint"])
	}

	if stats["service_name"] != cfg.ServiceName {
		t.Errorf("Expected service name %s, got %s", cfg.ServiceName, stats["service_name"])
	}

	if stats["service_version"] != cfg.ServiceVersion {
		t.Errorf("Expected service version %s, got %s", cfg.ServiceVersion, stats["service_version"])
	}
}
