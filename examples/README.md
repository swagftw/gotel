# GoTel Examples

This directory contains example applications demonstrating various usage patterns of the GoTel metrics library.

## Prerequisites

Before running any examples, make sure you have:

1. **Go 1.24 or later** installed
2. **Prometheus running locally** on port 9090, or set the `PROMETHEUS_ENDPOINT` environment variable to your Prometheus remote write endpoint
3. **Docker Compose** (optional, for running the full observability stack)

### Quick Setup with Docker Compose

To run Prometheus, Grafana, and OpenTelemetry Collector locally:

```bash
cd setup
docker-compose up -d
```

This will start:
- Prometheus on http://localhost:9090
- Grafana on http://localhost:3000 (admin/admin)
- OpenTelemetry Collector on http://localhost:4318

## Examples

### 1. Simple Usage (`simple_usage/`)

**What it demonstrates:**
- Basic GoTel client creation with defaults
- Counter and gauge metric creation
- Both synchronous and asynchronous metric sending
- Custom configuration with async metrics
- Manual metric management
- Health monitoring

**How to run:**
```bash
cd /path/to/gotel
go run examples/simple_usage/main.go
```

**Expected output:**
- Demonstrates basic metric creation and sending
- Shows async metric buffering and sending
- Displays client health statistics
- All metrics are successfully sent to Prometheus

### 2. Stress Testing (`stress_demo/`)

**What it demonstrates:**
- High-load metric sending with small buffer
- Goroutine pool fallback mechanism
- Zero data loss under extreme load
- Pool utilization and resource management

**How to run:**
```bash
cd /path/to/gotel
go run examples/stress_demo/main.go
```

**Expected output:**
- 200 concurrent requests (20 goroutines × 10 requests each)
- "Metrics channel full, falling back to pooled sync send" messages
- Pool utilization statistics (running: 10, free: 0)
- Zero dropped requests
- Proper cleanup after completion

**Key features tested:**
- Three-tier reliability system (channel → pool → drop counting)
- Managed goroutine pool with ants library
- Non-blocking async operations with fallback

### 3. HTTP Server Integration (`httpserver/`)

**What it demonstrates:**
- GoTel integration with Gin web framework
- Real-time metrics publishing
- HTTP endpoints for health and metrics information
- Graceful shutdown with proper resource cleanup

**How to run:**
```bash
cd /path/to/gotel
go run examples/httpserver/main.go
```

**Available endpoints:**
- `http://localhost:8080/` - Main page with usage examples
- `http://localhost:8080/health` - Health check endpoint
- `http://localhost:8080/metrics-info` - Current metrics information
- `http://localhost:8080/client-stats` - GoTel client statistics

**How to test:**
```bash
# Test health endpoint
curl http://localhost:8080/health

# Test metrics info
curl http://localhost:8080/metrics-info

# Test client stats
curl http://localhost:8080/client-stats
```

**Expected behavior:**
- Server starts on port 8080
- Metrics are sent to Prometheus every 5 seconds
- Endpoints return JSON responses with current status
- Graceful shutdown on Ctrl+C

## Running Tests

To run the httpserver tests:

```bash
cd /path/to/gotel
go test ./examples/httpserver/server
```

## Configuration

All examples support configuration via environment variables:

```bash
# Set custom Prometheus endpoint
export PROMETHEUS_ENDPOINT="http://your-prometheus:9090/api/v1/write"

# Set custom port for httpserver
export PORT="3000"

# Enable debug mode
export DEBUG="true"
```

## Troubleshooting

### Common Issues

1. **Connection refused to Prometheus:**
   - Ensure Prometheus is running on the configured endpoint
   - Check if Docker Compose stack is up: `docker-compose ps`
   - Verify endpoint URL format includes `/api/v1/write`

2. **Examples don't compile:**
   - Ensure you're in the gotel root directory
   - Run `go mod tidy` to resolve dependencies
   - Check Go version: `go version` (requires 1.24+)

3. **Metrics not appearing in Prometheus:**
   - Check Prometheus logs for remote write errors
   - Verify the remote write configuration in `prometheus.yml`
   - Check network connectivity to Prometheus endpoint

### Debug Mode

Enable debug logging in any example by setting the debug flag:

```go
cfg.EnableDebug = true
```

This will show detailed information about:
- Metric creation and sending
- Pool utilization
- Channel operations
- HTTP requests to Prometheus

## Performance Notes

- The `stress_demo/main.go` example intentionally uses a small buffer (5) to demonstrate fallback behavior
- In production, use larger buffers (100-1000) for better performance
- The goroutine pool prevents resource exhaustion under high load
- Rate limiting (1ms default) prevents overwhelming Prometheus

## Next Steps

After running these examples:

1. **Integrate with your application**: Use patterns from `simple_usage/main.go`
2. **Add custom metrics**: Follow the counter/gauge creation patterns
3. **Configure for production**: Adjust buffer sizes and intervals
4. **Set up monitoring**: Use the health endpoints for observability
5. **Scale testing**: Use `stress_demo/main.go` patterns for load testing
