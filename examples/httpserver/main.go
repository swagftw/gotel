package main

import (
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/GetSimpl/gotel"
	"github.com/GetSimpl/gotel/pkg/config"
	"github.com/GetSimpl/gotel/pkg/metrics"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	otelClient, err := gotel.New(cfg)
	if err != nil {
		slog.Error("failed to create OpenTelemetry client", "err", err)
		os.Exit(1)
	}

	defer otelClient.Close()

	svr := gin.New()

	svr.GET("/", func(context *gin.Context) {
		startTime := time.Now()

		//Record request counter
		otelClient.IncrementCounter(metrics.MetricCounterHttpRequestsTotal, metrics.UnitRequest, map[string]string{
			"http_method": context.Request.Method,
			"http_route":  context.FullPath(),
			"http_status": "200",
		})
		if err != nil {
			slog.Error("failed to increment counter", "err", err)
		}

		// simulate random ms of work from 1-10
		time.Sleep(time.Duration(1+rand.Intn(10)) * time.Millisecond)

		// Record latency
		duration := time.Since(startTime).Seconds()

		buckets := []float64{0.001, 0.01, 0.1, 0.5, 1.0} // 1ms to 1s buckets
		otelClient.RecordHistogram(duration, metrics.MetricHistHttpRequestDuration, metrics.UnitSeconds, buckets, map[string]string{
			"http_method": context.Request.Method,
			"http_route":  context.FullPath(),
			"http_status": "200",
		})
		if err != nil {
			slog.Error("failed to record histogram", "err", err)
		}

		context.Status(http.StatusOK)
	})

	err = svr.Run(":4000")
	if err != nil {
		slog.Error("failed to start server", "err", err)
		return
	}
}
