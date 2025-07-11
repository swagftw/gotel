package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var (
	meter        metric.Meter
	requestCount metric.Int64Counter
)

func main() {
	// Setup OpenTelemetry
	ctx := context.Background()
	shutdown := setupMeterProvider(ctx)
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}()

	// Initialize metrics
	initMetrics()

	engine := gin.Default()

	engine.GET("/", func(c *gin.Context) {
		// Increment request counter
		requestCount.Add(c.Request.Context(), 1)
		c.JSON(200, gin.H{"message": "Hello, World!"})
	})

	engine.GET("/metrics", func(c *gin.Context) {
		requestCount.Add(c.Request.Context(), 1)
		c.JSON(200, gin.H{"endpoint": "metrics", "status": "ok"})
	})

	if err := engine.Run(":8080"); err != nil {
		panic(err)
	}
}

// setupMeterProvider initializes the OpenTelemetry meter provider
func setupMeterProvider(ctx context.Context) func(context.Context) error {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("gotel-service"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Create OTLP HTTP metric exporter
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint("localhost:4318"),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to create metric exporter: %v", err)
	}

	// Create meter provider
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(10*time.Second))),
	)

	// Set global meter provider
	otel.SetMeterProvider(provider)

	return provider.Shutdown
}

// initMetrics initializes application-specific metrics
func initMetrics() {
	meter = otel.Meter("gotel-service")

	var err error
	requestCount, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		log.Fatalf("failed to create request counter: %v", err)
	}
}
