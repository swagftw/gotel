package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GetSimpl/gotel/pkg/config"
)

func TestNewServer(t *testing.T) {
	// Create a mock Prometheus server
	promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer promServer.Close()

	cfg := config.Default()
	cfg.PrometheusEndpoint = promServer.URL
	cfg.EnableAsyncMetrics = false // Disable for simpler testing
	cfg.MetricBufferSize = 100

	server := NewServer(cfg, ":0") // Let the OS choose a port

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config != cfg {
		t.Error("Expected server config to match provided config")
	}

	if server.prometheusClient == nil {
		t.Error("Expected Prometheus client to be initialized")
	}

	if server.metricsRegistry == nil {
		t.Error("Expected metrics registry to be initialized")
	}

	if server.engine == nil {
		t.Error("Expected Gin engine to be initialized")
	}
}

func TestHealthHandler(t *testing.T) {
	// Create a mock Prometheus server
	promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer promServer.Close()

	cfg := config.Default()
	cfg.PrometheusEndpoint = promServer.URL
	cfg.EnableAsyncMetrics = false
	cfg.MetricBufferSize = 100

	server := NewServer(cfg, ":0")

	// Create a test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	server.engine.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check response fields
	if response["status"] != "healthy" {
		t.Error("Expected status to be 'healthy'")
	}
}

func TestMainHandler(t *testing.T) {
	// Create a mock Prometheus server
	promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer promServer.Close()

	cfg := config.Default()
	cfg.PrometheusEndpoint = promServer.URL
	cfg.EnableAsyncMetrics = false
	cfg.MetricBufferSize = 100

	server := NewServer(cfg, ":0")

	// Create a test request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	server.engine.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check that request_count is present and is a number
	if _, ok := response["request_count"]; !ok {
		t.Error("Expected request_count in response")
	}
}

func TestMetricsInfoHandler(t *testing.T) {
	// Create a mock Prometheus server
	promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer promServer.Close()

	cfg := config.Default()
	cfg.PrometheusEndpoint = promServer.URL
	cfg.EnableAsyncMetrics = false
	cfg.MetricBufferSize = 100

	server := NewServer(cfg, ":0")

	// Create a test request
	req, err := http.NewRequest("GET", "/metrics-info", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	server.engine.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check response fields
	if _, ok := response["request_count"]; !ok {
		t.Error("Expected request_count in response")
	}
	if _, ok := response["active_requests"]; !ok {
		t.Error("Expected active_requests in response")
	}
	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}
}

func TestServerShutdown(t *testing.T) {
	// Create a mock Prometheus server
	promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer promServer.Close()

	cfg := config.Default()
	cfg.PrometheusEndpoint = promServer.URL
	cfg.EnableAsyncMetrics = false
	cfg.MetricBufferSize = 100

	server := NewServer(cfg, ":0")

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected shutdown to succeed, got error: %v", err)
	}
}

func ExampleNewServer() {
	cfg := config.Default()
	cfg.PrometheusEndpoint = "http://localhost:9090/api/v1/write"
	cfg.EnableAsyncMetrics = true
	cfg.MetricBufferSize = 1000

	server := NewServer(cfg, ":8080")

	// In a real application, you would call server.Start()
	fmt.Printf("Server created with endpoint: %s\n", cfg.PrometheusEndpoint)

	// For this example, we'll just show that the server was created
	if server != nil {
		fmt.Println("Server initialized successfully")
	}

	// Output:
	// Server created with endpoint: http://localhost:9090/api/v1/write
	// Server initialized successfully
}
