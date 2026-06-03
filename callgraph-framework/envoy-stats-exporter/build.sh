#!/bin/bash
set -euo pipefail

REGISTRY="${REGISTRY:-farzad1132}"
TAG="${TAG:-latest}"
FULL_IMAGE="${REGISTRY}/envoy-stats-exporter:${TAG}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building envoy-stats-exporter (${FULL_IMAGE})..."
go build -o /tmp/envoy-stats-exporter-test main.go
rm -f /tmp/envoy-stats-exporter-test

docker build -t "${FULL_IMAGE}" .
docker push "${FULL_IMAGE}"
echo "envoy-stats-exporter pushed: ${FULL_IMAGE}"
