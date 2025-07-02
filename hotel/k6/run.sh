#!/bin/bash

export K6_OTEL_METRIC_PREFIX="k6_"
export K6_OTEL_FLUSH_INTERVAL="500ms"
export K6_OTEL_EXPORT_INTERVAL="500ms"
export K6_OTEL_GRPC_EXPORTER_ENDPOINT="192.168.1.100:4317"
export K6_OTEL_GRPC_EXPORTER_INSECURE=true


k6 run script.js -o experimental-opentelemetry