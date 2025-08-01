// Package gotel provides OpenTelemetry metrics capabilities with real-time delivery.
// It offers a simple API for applications to send metrics to OpenTelemetry Collector
// and compatible backends.
package gotel

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GetSimpl/gotel/pkg/client"
	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/GetSimpl/gotel/pkg/metrics"
	"github.com/panjf2000/ants/v2"
)

// GoTel is the main client for sending metrics to OpenTelemetry Collector
type GoTel struct {
	config          *config.Config
	otelClient      *client.OtelClient
	metricsRegistry *metrics.Registry
	ctx             context.Context
	cancel          context.CancelFunc

	// Rate limiting for metrics sending
	lastMetricsSent time.Time
	metricsMutex    sync.RWMutex

	// Channel for async metrics sending
	metricsChan chan struct{}

	// Goroutine pool for fallback sync sends
	pool *ants.Pool

	// Metrics tracking
	droppedRequests int64 // atomic counter
}

// New creates a new GoTel client with the provided configuration
func New(cfg *config.Config) (*GoTel, error) {
	if cfg == nil {
		cfg = config.Default()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create OpenTelemetry client
	otelClient, err := client.NewOtelClient(cfg)
	if err != nil {
		cancel() // Clean up context
		return nil, fmt.Errorf("failed to create OTEL client: %w", err)
	}

	// Create metrics registry with OTEL client
	metricsRegistry := metrics.NewRegistry(otelClient, ctx)

	// Create goroutine pool for fallback sync sends
	poolSize := 10 // Default pool size for fallback sends
	if cfg.MetricBufferSize > 10 {
		poolSize = cfg.MetricBufferSize / 10 // Pool size is 1/10th of buffer size
	}

	pool, err := ants.NewPool(poolSize, ants.WithNonblocking(false))
	if err != nil {
		cancel() // Clean up context
		otelClient.Close()
		return nil, fmt.Errorf("failed to create goroutine pool: %w", err)
	}

	gotel := &GoTel{
		config:          cfg,
		otelClient:      otelClient,
		metricsRegistry: metricsRegistry,
		ctx:             ctx,
		cancel:          cancel,
		metricsChan:     make(chan struct{}, cfg.MetricBufferSize),
		pool:            pool,
		droppedRequests: 0,
	}

	// Start background metrics worker if async is enabled
	if cfg.EnableAsyncMetrics {
		go gotel.metricsWorker()
	}

	if cfg.LogLevel == "debug" || cfg.EnableDebug {
		log.Printf("GoTel initialized with OTEL endpoint: %s", cfg.OtelEndpoint)
		log.Printf("Async metrics: %v, Buffer size: %d", cfg.EnableAsyncMetrics, cfg.MetricBufferSize)
	}

	return gotel, nil
}

// NewWithDefaults creates a new GoTel client with default configuration
// This is the simplest way to get started with GoTel
func NewWithDefaults() (*GoTel, error) {
	cfg := config.FromEnv()
	return New(cfg)
}

// Counter creates or retrieves a counter metric
func (g *GoTel) Counter(name string, labels map[string]string) *metrics.Counter {
	return g.metricsRegistry.GetOrCreateCounter(name, labels)
}

// Gauge creates or retrieves a gauge metric
func (g *GoTel) Gauge(name string, labels map[string]string) *metrics.Gauge {
	return g.metricsRegistry.GetOrCreateGauge(name, labels)
}

// SendMetricsSync sends all registered metrics synchronously with rate limiting
func (g *GoTel) SendMetricsSync() error {
	g.metricsMutex.Lock()
	defer g.metricsMutex.Unlock()

	// Check if enough time has passed since last send (rate limiting)
	now := time.Now()
	if now.Sub(g.lastMetricsSent) < g.config.MinSendInterval {
		return nil // Skip sending due to rate limit
	}

	g.lastMetricsSent = now

	// Force flush OTEL metrics
	return g.otelClient.ForceFlush()
}

// SendMetricsAsync sends metrics asynchronously via buffered channel with fallback to sync
// If channel is full, falls back to pooled goroutine to ensure no metrics are lost
func (g *GoTel) SendMetricsAsync() {
	select {
	case g.metricsChan <- struct{}{}:
		// Successfully queued for async worker
	default:
		// Channel full, fallback to pooled goroutine for immediate send
		if g.config.LogLevel == "debug" || g.config.EnableDebug {
			log.Printf("Metrics channel full, falling back to pooled sync send")
		}

		// Submit to goroutine pool for fallback sync send
		err := g.pool.Submit(func() {
			g.sendMetricsWithRateLimit()
		})

		if err != nil {
			// Pool is also full or closed, count as dropped
			atomic.AddInt64(&g.droppedRequests, 1)
			if g.config.LogLevel == "debug" || g.config.EnableDebug {
				log.Printf("Goroutine pool full, dropped requests: %d", atomic.LoadInt64(&g.droppedRequests))
			}
		}
	}
}

// IncrementCounter is a convenience method to increment a counter by 1
func (g *GoTel) IncrementCounter(name string, labels map[string]string) error {
	counter := g.Counter(name, labels)
	counter.Inc()

	if g.config.EnableAsyncMetrics {
		g.SendMetricsAsync()
		return nil
	}

	return g.SendMetricsSync()
}

// SetGauge is a convenience method to set a gauge value
func (g *GoTel) SetGauge(name string, value float64, labels map[string]string) error {
	gauge := g.Gauge(name, labels)
	gauge.Set(value)

	if g.config.EnableAsyncMetrics {
		g.SendMetricsAsync()
		return nil
	}

	return g.SendMetricsSync()
}

// GetRegistry returns the internal metrics registry for advanced usage
func (g *GoTel) GetRegistry() *metrics.Registry {
	return g.metricsRegistry
}

// GetConfig returns the current configuration
func (g *GoTel) GetConfig() *config.Config {
	return g.config
}

// Close gracefully shuts down the GoTel client
func (g *GoTel) Close() error {
	// Send any remaining metrics
	if err := g.SendMetricsSync(); err != nil {
		log.Printf("Failed to send final metrics: %v", err)
	}

	// Cancel context to stop any background goroutines
	g.cancel()

	// Close the goroutine pool
	g.pool.Release()

	// Close OTEL client
	if err := g.otelClient.Close(); err != nil {
		log.Printf("Failed to close OTEL client: %v", err)
	}

	if g.config.LogLevel == "debug" || g.config.EnableDebug {
		dropped := atomic.LoadInt64(&g.droppedRequests)
		log.Printf("GoTel client shut down. Dropped requests: %d", dropped)
	}

	return nil
}

// Health returns basic health information about the GoTel client
func (g *GoTel) Health() map[string]interface{} {
	return map[string]interface{}{
		"status":           "healthy",
		"otel_endpoint":    g.config.OtelEndpoint,
		"metrics_count":    g.metricsRegistry.MetricsCount(),
		"async_enabled":    g.config.EnableAsyncMetrics,
		"config_valid":     g.config.Validate() == nil,
		"pool_running":     g.pool.Running(),
		"pool_free":        g.pool.Free(),
		"pool_capacity":    g.pool.Cap(),
		"dropped_requests": atomic.LoadInt64(&g.droppedRequests),
	}
}

// metricsWorker runs in background to send metrics asynchronously with rate limiting
func (g *GoTel) metricsWorker() {
	ticker := time.NewTicker(g.config.SendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			// Send final metrics before shutdown
			g.sendMetricsWithRateLimit()
			return
		case <-ticker.C:
			// Periodic send
			g.sendMetricsWithRateLimit()
		case <-g.metricsChan:
			// Immediate send requested from SendMetricsAsync()
			g.sendMetricsWithRateLimit()
		}
	}
}

// sendMetricsWithRateLimit sends metrics with rate limiting to prevent duplicate timestamps
func (g *GoTel) sendMetricsWithRateLimit() {
	g.metricsMutex.Lock()
	defer g.metricsMutex.Unlock()

	// Check if enough time has passed since last send
	now := time.Now()
	if now.Sub(g.lastMetricsSent) < g.config.MinSendInterval {
		return
	}

	g.lastMetricsSent = now

	// Force flush OTEL metrics
	if err := g.otelClient.ForceFlush(); err != nil {
		if g.config.LogLevel == "debug" || g.config.EnableDebug {
			log.Printf("Failed to flush metrics: %v", err)
		}
	}
}
