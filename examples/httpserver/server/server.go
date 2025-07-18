package server

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/GetSimpl/gotel/pkg/client"
	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/GetSimpl/gotel/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server with metrics
type Server struct {
	config           *config.Config
	port             string
	engine           *gin.Engine
	prometheusClient *client.PrometheusClient
	metricsRegistry  *metrics.Registry
	metricsChan      chan struct{}
	httpServer       *http.Server

	// Rate limiting for metrics sending
	lastMetricsSent time.Time
	metricsMutex    sync.RWMutex

	// Metrics
	requestCounter *metrics.Counter
	activeGauge    *metrics.Gauge
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, port string) *Server {
	if port == "" {
		port = ":8080"
	}

	// Initialize Prometheus client using unified configuration
	promClient := client.NewPrometheusClient(cfg)

	// Initialize metrics registry
	registry := metrics.NewRegistry()

	// Create server
	server := &Server{
		config:           cfg,
		port:             port,
		engine:           gin.Default(),
		prometheusClient: promClient,
		metricsRegistry:  registry,
		metricsChan:      make(chan struct{}, cfg.MetricBufferSize),
	}

	// Initialize metrics
	server.initMetrics()

	// Setup routes
	server.setupRoutes()

	// Start async metrics sender if enabled
	if cfg.EnableAsyncMetrics {
		go server.metricsWorker()
	}

	return server
}

// initMetrics initializes the server metrics
func (s *Server) initMetrics() {
	labels := s.config.GetLabels()

	s.requestCounter = s.metricsRegistry.GetOrCreateCounter("gotel_http_requests_total", labels)
	s.activeGauge = s.metricsRegistry.GetOrCreateGauge("gotel_active_requests", labels)
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Add middleware for request tracking
	s.engine.Use(s.requestMiddleware())

	// Health check endpoint
	s.engine.GET("/health", s.healthHandler)

	// Main endpoint
	s.engine.GET("/", s.mainHandler)

	// Metrics info endpoint
	s.engine.GET("/metrics-info", s.metricsInfoHandler)

	// Client stats endpoint
	s.engine.GET("/client-stats", s.clientStatsHandler)
}

// requestMiddleware tracks active requests
func (s *Server) requestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Increment active requests
		s.activeGauge.Inc()
		defer s.activeGauge.Dec()

		c.Next()
	}
}

// mainHandler handles the main endpoint
func (s *Server) mainHandler(c *gin.Context) {
	// Increment request counter
	count := s.requestCounter.Inc()

	// Send metrics
	s.sendMetrics()

	c.JSON(200, gin.H{
		"message":       "Hello, World!",
		"request_count": count,
		"timestamp":     time.Now().Unix(),
	})
}

// healthHandler handles health check requests
func (s *Server) healthHandler(c *gin.Context) {
	stats := s.prometheusClient.GetStats()

	c.JSON(200, gin.H{
		"status":           "healthy",
		"timestamp":        time.Now().Unix(),
		"prometheus_stats": stats,
	})
}

// metricsInfoHandler returns current metrics values
func (s *Server) metricsInfoHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"request_count":   s.requestCounter.Get(),
		"active_requests": s.activeGauge.Get(),
		"timestamp":       time.Now().Unix(),
	})
}

// clientStatsHandler returns Prometheus client statistics
func (s *Server) clientStatsHandler(c *gin.Context) {
	stats := s.prometheusClient.GetStats()
	c.JSON(200, stats)
}

// sendMetrics sends metrics to Prometheus
func (s *Server) sendMetrics() {
	if s.config.EnableAsyncMetrics {
		// Send signal to metrics worker
		select {
		case s.metricsChan <- struct{}{}:
		default:
			log.Printf("Metrics channel full, dropping metric send request")
		}
	} else {
		// Send synchronously
		s.sendMetricsSync()
	}
}

// sendMetricsSync sends metrics synchronously with 1ms rate limiting
func (s *Server) sendMetricsSync() {
	// Rate limit to 1ms to prevent duplicate timestamp errors
	s.metricsMutex.RLock()
	timeSinceLastSend := time.Since(s.lastMetricsSent)
	s.metricsMutex.RUnlock()

	// Don't send metrics more frequently than 1ms
	if timeSinceLastSend < time.Millisecond {
		return
	}

	timeSeries := s.metricsRegistry.Export()
	if len(timeSeries) == 0 {
		return
	}

	if err := s.prometheusClient.SendMetrics(timeSeries); err != nil {
		log.Printf("Failed to send metrics: %v", err)
	} else {
		// Update last sent timestamp on success
		s.metricsMutex.Lock()
		s.lastMetricsSent = time.Now()
		s.metricsMutex.Unlock()
	}
}

// metricsWorker handles async metrics sending
func (s *Server) metricsWorker() {
	ticker := time.NewTicker(5 * time.Second) // Send metrics every 5 seconds or on demand
	defer ticker.Stop()

	for {
		select {
		case <-s.metricsChan:
			s.sendMetricsSync()
		case <-ticker.C:
			// Periodic send to ensure metrics don't get lost
			s.sendMetricsSync()
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.port,
		Handler:      s.engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting GoTel server on %s", s.port)
	log.Printf("Prometheus endpoint: %s", s.config.PrometheusEndpoint)
	log.Printf("Async metrics: %v", s.config.EnableAsyncMetrics)

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Shutdown HTTP server if it exists
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	// Send final metrics
	s.sendMetricsSync()

	// Close Prometheus client
	if err := s.prometheusClient.Close(); err != nil {
		log.Printf("Error closing Prometheus client: %v", err)
	}

	log.Println("Server shutdown complete")
	return nil
}
