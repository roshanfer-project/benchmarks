#!/bin/bash

export K6_OTEL_METRIC_PREFIX="k6_"
export K6_OTEL_FLUSH_INTERVAL="100ms"
export K6_OTEL_EXPORT_INTERVAL="100ms"
export K6_OTEL_GRPC_EXPORTER_ENDPOINT="192.168.1.100:4317"
export K6_OTEL_GRPC_EXPORTER_INSECURE=true

# if the first argument is grpc, set the script to ov-grpc.js
if [ "$1" == "grpc" ]; then
  script="ov-grpc.js"
else
  script="script.js"
fi

start_time=$(date +%s.%6N)
k6 run $script -o experimental-opentelemetry
end_time=$(date +%s.%6N)

echo "Write timestamps to a file"
file="../../../experiments/ov-vs-time/timestamps.csv"
echo "$start_time" > "$file"
echo "$end_time" >> "$file"