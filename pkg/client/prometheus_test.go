package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/prometheus/prometheus/prompb"
)

func TestNewPrometheusClient(t *testing.T) {
	cfg := config.Default()
	cfg.PrometheusEndpoint = "http://test.example.com"
	cfg.HTTPTimeout = 10 * time.Second
	cfg.RetryCount = 2
	cfg.MaxIdleConnections = 50
	cfg.MaxConnectionsPerHost = 5

	client := NewPrometheusClient(cfg)

	if client == nil {
		t.Error("Expected client to be created")
		return
	}

	if client.config != cfg {
		t.Error("Expected client config to match provided config")
	}

	if client.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestSendMetrics_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/write" {
			t.Errorf("Expected path /api/v1/write, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.PrometheusEndpoint = server.URL + "/api/v1/write"
	client := NewPrometheusClient(cfg)

	// Create test metrics
	samples := []prompb.TimeSeries{
		{
			Labels: []prompb.Label{
				{Name: "__name__", Value: "test_metric"},
				{Name: "job", Value: "test"},
			},
			Samples: []prompb.Sample{
				{Value: 1.0, Timestamp: time.Now().UnixMilli()},
			},
		},
	}

	err := client.SendMetrics(samples)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGetStats(t *testing.T) {
	cfg := config.Default()
	cfg.PrometheusEndpoint = "http://test.example.com"
	cfg.HTTPTimeout = 15 * time.Second
	cfg.RetryCount = 5
	client := NewPrometheusClient(cfg)

	stats := client.GetStats()

	if stats["endpoint"] != cfg.PrometheusEndpoint {
		t.Errorf("Expected endpoint %s, got %s", cfg.PrometheusEndpoint, stats["endpoint"])
	}

	if stats["timeout"] != cfg.HTTPTimeout.String() {
		t.Errorf("Expected timeout %s, got %s", cfg.HTTPTimeout.String(), stats["timeout"])
	}

	if stats["retry_count"] != cfg.RetryCount {
		t.Errorf("Expected retry count %d, got %v", cfg.RetryCount, stats["retry_count"])
	}
}
