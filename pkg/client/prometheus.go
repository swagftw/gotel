package client

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/go-resty/resty/v2"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

// PrometheusClient handles direct communication with Prometheus remote write endpoint
type PrometheusClient struct {
	config *config.Config
	client *resty.Client
}

// NewPrometheusClient creates a new Prometheus client with optimized transport settings
func NewPrometheusClient(cfg *config.Config) *PrometheusClient {
	// Create custom HTTP transport with optimized settings
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConnections,
		MaxIdleConnsPerHost: cfg.MaxConnectionsPerHost,
		IdleConnTimeout:     cfg.IdleConnectionTimeout,
		DisableCompression:  true, // We handle compression ourselves with snappy
	}

	// Create Resty client with custom transport
	client := resty.New()
	client.SetTransport(transport)

	// Configure retry policy with backoff for 429 and 5xx errors
	client.SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(cfg.RetryWaitTime).
		SetRetryMaxWaitTime(cfg.RetryMaxWaitTime).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry on network errors
			if err != nil {
				return true
			}
			// Retry on 429 (Too Many Requests) and 5xx server errors
			return r.StatusCode() == 429 || (r.StatusCode() >= 500 && r.StatusCode() < 600)
		}).
		SetTimeout(cfg.HTTPTimeout)

	// Set default headers for Prometheus remote write
	client.SetHeaders(map[string]string{
		"Content-Encoding":                  "snappy",
		"Content-Type":                      "application/x-protobuf",
		"X-Prometheus-Remote-Write-Version": "0.1.0",
		"User-Agent":                        "gotel/1.0.0",
	})

	return &PrometheusClient{
		config: cfg,
		client: client,
	}
}

// SendMetrics sends metrics directly to Prometheus using remote write protocol
func (p *PrometheusClient) SendMetrics(samples []prompb.TimeSeries) error {
	if len(samples) == 0 {
		return fmt.Errorf("no metrics to send")
	}

	// Create WriteRequest
	writeReq := &prompb.WriteRequest{
		Timeseries: samples,
	}

	// Marshal to protobuf
	data, err := writeReq.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	// Compress with snappy
	compressed := snappy.Encode(nil, data)

	// Send request using Resty with automatic retries
	resp, err := p.client.R().
		SetBody(bytes.NewReader(compressed)).
		Post(p.config.PrometheusEndpoint)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode(), resp.String())
	}

	log.Printf("Metrics sent successfully to Prometheus (status: %d, samples: %d)", resp.StatusCode(), len(samples))
	return nil
}

// Close cleans up the client resources
func (p *PrometheusClient) Close() error {
	// Close idle connections
	if transport, ok := p.client.GetClient().Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// GetStats returns client statistics
func (p *PrometheusClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"endpoint":            p.config.PrometheusEndpoint,
		"timeout":             p.config.HTTPTimeout.String(),
		"retry_count":         p.config.RetryCount,
		"max_idle_conns":      p.config.MaxIdleConnections,
		"max_idle_conns_host": p.config.MaxConnectionsPerHost,
		"idle_conn_timeout":   p.config.IdleConnectionTimeout.String(),
		"keep_alive":          p.config.KeepAlive.String(),
	}
}
