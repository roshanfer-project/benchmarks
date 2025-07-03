#!/bin/bash

export K6_OTEL_METRIC_PREFIX="k6_"
export K6_OTEL_FLUSH_INTERVAL="100ms"
export K6_OTEL_EXPORT_INTERVAL="100ms"
export K6_OTEL_GRPC_EXPORTER_ENDPOINT="192.168.1.100:4317"
export K6_OTEL_GRPC_EXPORTER_INSECURE=true

start_time=$(date +%s.%6N)
k6 run script.js -o experimental-opentelemetry
end_time=$(date +%s.%6N)

echo "Write timestamps to a file"
file="../../../experiments/ov-rate-vs-time/timestamps.csv"
echo "$start_time" > "$file"
echo "$end_time" >> "$file"