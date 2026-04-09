#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
STATUS_FILE=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=dynamic-large

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

echo "Building backend1..."
docker build --build-arg SERVICE=services/backend1 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend1:${TAG}" .
echo "Building backend2..."
docker build --build-arg SERVICE=services/backend2 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend2:${TAG}" .
echo "Building backend3..."
docker build --build-arg SERVICE=services/backend3 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend3:${TAG}" .
echo "Building backend4..."
docker build --build-arg SERVICE=services/backend4 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend4:${TAG}" .
echo "Building backend5..."
docker build --build-arg SERVICE=services/backend5 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend5:${TAG}" .
echo "Building backend6..."
docker build --build-arg SERVICE=services/backend6 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend6:${TAG}" .
echo "Building backend7..."
docker build --build-arg SERVICE=services/backend7 -f Dockerfile -t "${REGISTRY}/${BENCH}-backend7:${TAG}" .
echo "Building frontend..."
docker build --build-arg SERVICE=services/frontend -f Dockerfile -t "${REGISTRY}/${BENCH}-frontend:${TAG}" .
echo "Building frontend-grpc..."
docker build --build-arg SERVICE=services/frontend-grpc -f Dockerfile -t "${REGISTRY}/${BENCH}-frontend-grpc:${TAG}" .
echo "Building rajomon-client..."
docker build --build-arg SERVICE=services/rajomon-client -f Dockerfile -t "${REGISTRY}/${BENCH}-rajomon-client:${TAG}" .

echo "Pushing images..."
if [ -n "$SIDECAR_DIR" ]; then
  docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
fi
docker push "${REGISTRY}/${BENCH}-backend1:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend2:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend3:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend4:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend5:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend6:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend7:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend-grpc:${TAG}"
docker push "${REGISTRY}/${BENCH}-rajomon-client:${TAG}"

if [ -n "$STATUS_FILE" ]; then
  mkdir -p "$(dirname "$STATUS_FILE")"
  touch "$STATUS_FILE"
fi
echo "Build complete."
