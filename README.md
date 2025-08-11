# GoTel

A Go library for OpenTelemetry metrics with automatic batching and reliable delivery. GoTel provides a simple API for applications to send counters, gauges, and histograms to OpenTelemetry Collector and compatible backends.

## Features

- Simple interface for OpenTelemetry metrics
- Automatic batching and buffering via OTEL SDK
- Metrics are thread safe and syncing is managed by package itself
- Support for counters, gauges, and histograms
- Configurable via environment variables
- Default service and environment labels
- Debug logging support

## Installation

```bash
go get github.com/GetSimpl/gotel
```

## Quick Start

### 1. Basic Usage

```go
package main

import (
    "log"
    "github.com/GetSimpl/gotel"
    "github.com/GetSimpl/gotel/pkg/config"
    "github.com/GetSimpl/gotel/pkg/metrics"
)

func main() {
    // Load configuration (from env vars or use defaults)
    cfg, err := config.LoadConfig()
    if err != nil { 
        // handle error depending on the setup
        log.Fatal(err)
    }

    // Create gotel client
    // For better usage, you can expose this client globally and use it anywhere
    client, err := gotel.New(cfg)
    if err != nil { 
        // handle error depending on the setup
        log.Fatal(err)
    }
    defer client.Close()

    // Use the client
    client.IncrementCounter(
        metrics.MetricCounterHttpRequestsTotal,
        metrics.UnitRequest,
        map[string]string{"http.method": "GET", "http.status": "200"},
    )
}
```

### 2. Configuration

Configure via environment variables:

```bash
export OTEL_ENDPOINT=http://localhost:4318
export OTEL_SERVICE_NAME=my-service
export OTEL_SERVICE_VERSION=1.0.0
export ENV=production
export OTEL_SEND_INTERVAL=30
export OTEL_DEBUG=false
```

Or use the provided development configuration:

```bash
source development.env
```

## API Reference

The `Gotel` interface provides these methods:

### IncrementCounter
Increments a counter by 1.

```go
IncrementCounter(name metrics.MetricName, unit metrics.Unit, labels map[string]string)
```

### AddToCounter
Adds a specific value to a counter.

```go
AddToCounter(delta int64, name metrics.MetricName, unit metrics.Unit, labels map[string]string)
```

### SetGauge
Sets a gauge to a specific value.

```go
SetGauge(value float64, name metrics.MetricName, unit metrics.Unit, labels map[string]string)
```

### RecordHistogram
Records a value in a histogram with custom buckets.

```go
RecordHistogram(value float64, name metrics.MetricName, unit metrics.Unit, buckets []float64, labels map[string]string)
```

### Close
Gracefully shuts down the client and flushes remaining metrics.

```go
Close() error
```

## Built-in Metric Names and Units

### Metric Names
- `metrics.MetricCounterHttpRequestsTotal` - HTTP request counter
- `metrics.MetricHistHttpRequestDuration` - HTTP request duration histogram

### Units
- `metrics.UnitPercent` - Percentage (%)
- `metrics.UnitSeconds` - Seconds (s)
- `metrics.UnitMilliseconds` - Milliseconds (ms)
- `metrics.UnitBytes` - Bytes (By)
- `metrics.UnitRequest` - Request count ({request})

## Example: HTTP Server

See the complete example in `examples/httpserver/main.go`:

```go
// Record request counter
client.IncrementCounter(
    metrics.MetricCounterHttpRequestsTotal,
    metrics.UnitRequest,
    map[string]string{
        "http_method": "GET",
        "http_route":  "/api/users",
        "http_status": "200",
    },
)

// Record request duration
// use sensible buckets according to average response times
buckets := []float64{0.001, 0.01, 0.1, 1.0}
client.RecordHistogram(
    duration,
    metrics.MetricHistHttpRequestDuration,
    metrics.UnitSeconds,
    buckets,
    map[string]string{
        "http_method": "GET",
        "http_route":  "/api/users",
        "http_status": "200",
    },
)
```

## Running the Example

1. Start the OpenTelemetry stack:
```bash
cd setup
docker-compose up -d
```

2. Run the example HTTP server:
```bash
source development.env
go run examples/httpserver/main.go
```

3. Generate some metrics:
```bash
curl http://localhost:4000/
```

4. View metrics:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

## Configuration Options

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `OTEL_ENDPOINT` | `http://localhost:4318` | OpenTelemetry collector endpoint |
| `OTEL_SERVICE_NAME` | `gotel-app` | Service name for metrics |
| `OTEL_SERVICE_VERSION` | `1.0.0` | Service version |
| `ENV` | `local` | Environment (local, staging, production) |
| `OTEL_SEND_INTERVAL` | `30` | Batch send interval in seconds |
| `OTEL_DEBUG` | `false` | Enable debug logging |

## Default Labels

GoTel automatically adds these labels to all metrics:
- `service.name` - From `OTEL_SERVICE_NAME`
- `environment` - From `ENV`

## Contributing

We welcome contributions to GoTel! Here's how you can help:

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/GetSimpl/gotel.git
   cd gotel
   ```
   
2. Create development.env with following con:
    ```bash
    OTEL_ENDPOINT={{otel-collector-endpoint}}
    OTEL_SERVICE_NAME={{service-name}}
    OTEL_SERVICE_VERSION={{service-version}}
    ENV={{environment}}
    OTEL_SEND_INTERVAL=60
    OTEL_DEBUG=true
    ```

3. Set up the development environment:
   ```bash
   source development.env
   cd setup && docker-compose up -d
   ```

### Development Guidelines

- Write tests for new features
- Follow Go conventions and best practices
- Run tests before submitting: `go test ./...`
- Update documentation for API changes
- Use meaningful commit messages

### Testing Your Changes

1. Run the test suite:
   ```bash
   go test ./...
   ```

2. Test with the example application, update it if needed:
   ```bash
   go run examples/httpserver/main.go
   ```

3. Verify metrics are being sent to the collector

### Submitting Changes

1. Create a feature branch: `git checkout -b feature-name`
2. Make your changes and commit them
3. Push to your fork: `git push origin feature-name`
4. Create a Pull Request with:
   - Clear description of changes
   - Test results
   - Any breaking changes noted
