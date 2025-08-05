# Latency Monitoring with Histograms and Percentiles

This guide shows how to collect HTTP request latency using histograms and create Grafana dashboards with P50, P90, P95, and P99 percentiles.

## Overview

The GoTel client now supports histogram metrics for latency tracking:
- **Metric Name**: `http_request_duration_seconds`
- **Type**: Histogram
- **Labels**: `http_method`, `http_route`, `http_status`, `service_name`, `service_environment`

## How It Works

1. **Go Application** records request duration using `RecordHistogram()`
2. **OTEL Collector** receives histogram data via OTLP
3. **Prometheus** scrapes histogram buckets from collector
4. **Grafana** calculates percentiles using `histogram_quantile()`

## Using Histograms in Code

```go
package main

import (
    "time"
    "github.com/GetSimpl/gotel/pkg/client"
)

func handleRequest(otelClient *client.OtelClient) {
    startTime := time.Now()
    
    // Your request processing logic here
    
    // Record latency
    duration := time.Since(startTime).Seconds()
    err := otelClient.RecordHistogram("http_request_duration_seconds", map[string]string{
        "http_method": "GET",
        "http_route":  "/api/users",
        "http_status": "200",
    }, duration)
    if err != nil {
        log.Printf("Failed to record latency: %v", err)
    }
}
```

## Grafana Dashboard Setup

### 1. Create a New Dashboard

1. Open Grafana (http://localhost:3000)
2. Click "+" → "Dashboard"
3. Click "Add visualization"
4. Select "Prometheus" as data source

### 2. P99 Latency Panel

**Panel Title**: "P99 Response Time"
**Query**:
```promql
histogram_quantile(0.99, 
  rate(http_request_duration_seconds_bucket[5m])
)
```

**Visualization**: Time series
**Unit**: seconds
**Y-axis**: Start from 0

### 3. P95 Latency Panel

**Panel Title**: "P95 Response Time"
**Query**:
```promql
histogram_quantile(0.95, 
  rate(http_request_duration_seconds_bucket[5m])
)
```

### 4. P90 Latency Panel

**Panel Title**: "P90 Response Time"
**Query**:
```promql
histogram_quantile(0.90, 
  rate(http_request_duration_seconds_bucket[5m])
)
```

### 5. P50 (Median) Latency Panel

**Panel Title**: "P50 Response Time (Median)"
**Query**:
```promql
histogram_quantile(0.50, 
  rate(http_request_duration_seconds_bucket[5m])
)
```

### 6. Multi-Percentile Panel

**Panel Title**: "Response Time Percentiles"
**Queries** (add multiple queries):

Query A (P50):
```promql
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))
```

Query B (P90):
```promql
histogram_quantile(0.90, rate(http_request_duration_seconds_bucket[5m]))
```

Query C (P95):
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

Query D (P99):
```promql
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

**Legend**: Use `{{__name__}} P50`, `{{__name__}} P90`, etc.

### 7. Latency by Route Panel

**Panel Title**: "P95 Latency by Route"
**Query**:
```promql
histogram_quantile(0.95, 
  sum(rate(http_request_duration_seconds_bucket[5m])) by (http_route, le)
)
```

### 8. Latency by Status Code Panel

**Panel Title**: "P95 Latency by Status Code"
**Query**:
```promql
histogram_quantile(0.95, 
  sum(rate(http_request_duration_seconds_bucket[5m])) by (http_status, le)
)
```

## Advanced Queries

### Average Latency
```promql
rate(http_request_duration_seconds_sum[5m]) / 
rate(http_request_duration_seconds_count[5m])
```

### Request Rate
```promql
rate(http_request_duration_seconds_count[5m])
```

### Latency Heatmap
For heatmap visualization:
```promql
sum(rate(http_request_duration_seconds_bucket[5m])) by (le)
```

## Dashboard Configuration Tips

### Time Range
- Use "Last 1 hour" or "Last 6 hours" for monitoring
- Set auto-refresh to 30s or 1m

### Thresholds
Set up visual thresholds:
- **Green**: < 100ms
- **Yellow**: 100ms - 500ms  
- **Red**: > 500ms

### Alerts
Create alerts for SLA violations:
```promql
histogram_quantile(0.95, 
  rate(http_request_duration_seconds_bucket[5m])
) > 0.5
```

## Sample Dashboard JSON

Here's a complete dashboard configuration:

```json
{
  "dashboard": {
    "title": "HTTP Latency Monitoring",
    "panels": [
      {
        "title": "Response Time Percentiles",
        "type": "timeseries",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.90, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P90"
          },
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P99"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "s"
          }
        }
      }
    ]
  }
}
```

## Testing Your Setup

1. **Start the infrastructure**:
   ```bash
   cd setup
   docker-compose up -d
   ```

2. **Run your Go application** with histogram recording

3. **Generate some traffic**:
   ```bash
   # Generate requests to create latency data
   for i in {1..100}; do curl http://localhost:4000/; done
   ```

4. **Check Prometheus** (http://localhost:9090):
   - Query: `http_request_duration_seconds_bucket`
   - You should see histogram buckets

5. **View in Grafana** (http://localhost:3000):
   - Login: admin/admin
   - Create dashboard with the queries above

## Troubleshooting

### No Histogram Data
- Ensure your application is calling `RecordHistogram()`
- Check OTEL collector logs: `docker logs otel-collector`
- Verify Prometheus is scraping: check `/targets` page

### Wrong Percentile Values
- Histogram buckets may not cover your latency range
- Default buckets: 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0, 7.5, 10.0
- For custom buckets, modify the OTEL SDK configuration

### Missing Labels
- Ensure all requests include the same label keys
- Missing labels will create separate histogram series

## Best Practices

1. **Consistent Labels**: Always use the same label keys across requests
2. **Reasonable Buckets**: Default buckets work for most HTTP APIs
3. **Rate Intervals**: Use 5m intervals for percentiles (balance between accuracy and smoothness)
4. **Alert Thresholds**: Set P95 or P99 thresholds based on SLA requirements
5. **Dashboard Organization**: Group related metrics (latency, throughput, errors) together
