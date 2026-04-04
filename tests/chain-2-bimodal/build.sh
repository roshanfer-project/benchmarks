#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
STATUS_FILE=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=chain-2-bimodal

if [ -n "$STATUS_FILE" ]; then
  STATUS_DIR=$(dirname "$STATUS_FILE")
  STATUS_BASE=$(basename "$STATUS_FILE")
  mkdir -p "$STATUS_DIR"
  STATUS_DIR=$(cd "$STATUS_DIR" && pwd)
  STATUS_FILE="${STATUS_DIR}/${STATUS_BASE}"
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

# Find sidecar dir (benchmarks/sidecar) - works for nested benchmarks like tests/one-service
SIDECAR_DIR=""
D="$ROOT_DIR"
while [ -n "$D" ] && [ "$D" != "/" ]; do
  if [ -d "$D/sidecar" ]; then
    SIDECAR_DIR="$D/sidecar"
    break
  fi
  D="$(cd "$D/.." && pwd)"
done

if [ -n "$SIDECAR_DIR" ]; then
  echo "Building sidecar..."
  (cd "$SIDECAR_DIR" && ./build.sh Release)
  docker build -f "$SIDECAR_DIR/Dockerfile" -t "${REGISTRY}/sidecar-sidecar:${TAG}" "$SIDECAR_DIR"
fi

echo "Building backend..."
docker build --build-arg SERVICE=services/backend -f Dockerfile -t "${REGISTRY}/${BENCH}-backend:${TAG}" .
echo "Building frontend..."
docker build --build-arg SERVICE=services/frontend -f Dockerfile -t "${REGISTRY}/${BENCH}-frontend:${TAG}" .

echo "Pushing images..."
if [ -n "$SIDECAR_DIR" ]; then
  docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
fi
docker push "${REGISTRY}/${BENCH}-backend:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend:${TAG}"

if [ -n "$STATUS_FILE" ]; then
  mkdir -p "$(dirname "$STATUS_FILE")"
  touch "$STATUS_FILE"
fi
echo "Build complete."
