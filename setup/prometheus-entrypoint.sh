#!/bin/sh

# Prometheus startup script with conditional parameters
ARGS="--config.file=/etc/prometheus/prometheus.yml \
      --storage.tsdb.path=/prometheus \
      --web.console.libraries=/etc/prometheus/console_libraries \
      --web.console.templates=/etc/prometheus/consoles \
      --web.enable-lifecycle \
      --web.enable-admin-api \
      --enable-feature=remote-write-receiver \
      --web.listen-address=0.0.0.0:9090 \
      --storage.tsdb.retention.time=${PROMETHEUS_RETENTION:-15d}"

# Add retention size only if specified
if [ -n "${PROMETHEUS_RETENTION_SIZE}" ]; then
    ARGS="$ARGS --storage.tsdb.retention.size=${PROMETHEUS_RETENTION_SIZE}"
fi

# Add query timeout if specified
if [ -n "${PROMETHEUS_QUERY_TIMEOUT}" ]; then
    ARGS="$ARGS --query.timeout=${PROMETHEUS_QUERY_TIMEOUT}"
fi

# Add query max concurrency if specified
if [ -n "${PROMETHEUS_QUERY_MAX_CONCURRENCY}" ]; then
    ARGS="$ARGS --query.max-concurrency=${PROMETHEUS_QUERY_MAX_CONCURRENCY}"
fi

# Start Prometheus with constructed arguments
exec /bin/prometheus $ARGS
