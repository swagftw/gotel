# GoTel - OpenTelemetry Metrics Demo

This project demonstrates how to implement OpenTelemetry metrics in a Go application with Gin framework, using Prometheus for storage and Grafana for visualization.

## Architecture

- **Go Application**: Gin web server with OpenTelemetry instrumentation
- **OpenTelemetry Collector**: Receives metrics via OTLP and exports to Prometheus
- **Prometheus**: Time-series database for metrics storage
- **Grafana**: Visualization dashboard with auto-provisioned datasources

## Quick Start

1. **Start the monitoring stack**:
   ```bash
   cd setup
   docker-compose up -d
   ```

2. **Run the Go application**:
   ```bash
   cd playground
   go run main.go
   ```

3. **Access the services**:
   - Application: http://localhost:8080
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (admin/admin)

## Testing Metrics

Generate some traffic to see metrics:
```bash
# Generate requests
curl http://localhost:8080/
curl http://localhost:8080/metrics

# Check metrics in Prometheus
# Go to http://localhost:9090 and query: http_requests_total
```

## Grafana Dashboard

The setup automatically provisions:
- Prometheus datasource
- "GoTel Application Metrics" dashboard
- Panels showing HTTP request rate and total requests

## Metrics Collected

- `http_requests_total`: Counter of total HTTP requests
- Automatic Gin middleware metrics via OpenTelemetry

## Configuration Files

- `otel-collector-config.yaml`: OTel Collector configuration
- `prometheus.yml`: Prometheus scrape configuration
- `grafana/provisioning/`: Auto-provisioning for Grafana

## Stopping the Stack

```bash
cd setup
docker-compose down
```
