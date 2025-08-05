package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/GetSimpl/gotel"
	"github.com/GetSimpl/gotel/pkg/config"
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

	svr := gin.Default()

	svr.GET("/", func(context *gin.Context) {
		startTime := time.Now()

		// Record request counter
		err = otelClient.IncrementCounter(HttpRequestsTotalCounterMetricName, map[string]string{
			"http_method": context.Request.Method,
			"http_route":  context.FullPath(),
			"http_status": "200",
		}, 1)
		if err != nil {
			slog.Error("failed to increment counter", "err", err)
		}

		// Simulate some work
		time.Sleep(time.Millisecond * 10) // 10ms of work

		// Record latency
		duration := time.Since(startTime).Seconds()
		err = otelClient.RecordHistogram(HttpRequestDurationHistogramName, map[string]string{
			"http_method": context.Request.Method,
			"http_route":  context.FullPath(),
			"http_status": "200",
		}, duration)
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
