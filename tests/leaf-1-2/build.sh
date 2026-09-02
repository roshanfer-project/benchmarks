#!/bin/bash
set -euo pipefail
TAG=${1:-${TAG:-latest}}
STATUS_FILE=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=leaf-1-2

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

if [ "${SKIP_SIDECAR_BUILD:-}" != "1" ] && [ -n "$SIDECAR_DIR" ]; then
  echo "Building sidecar..."
  (cd "$SIDECAR_DIR" && ./build.sh Release)
  docker build -f "$SIDECAR_DIR/Dockerfile" -t "${REGISTRY}/sidecar-sidecar:${TAG}" "$SIDECAR_DIR"
fi

ENVOY_STATS_DIR=""
D="$ROOT_DIR"
while [ -n "$D" ] && [ "$D" != "/" ]; do
  if [ -d "$D/callgraph-framework/envoy-stats-exporter" ]; then
    ENVOY_STATS_DIR="$D/callgraph-framework/envoy-stats-exporter"
    break
  fi
  D="$(cd "$D/.." && pwd)"
done
if [ "${SKIP_SIDECAR_BUILD:-}" = "1" ]; then
  if [ -z "$ENVOY_STATS_DIR" ]; then
    echo "build.sh: callgraph-framework/envoy-stats-exporter not found (walk up from $ROOT_DIR)" >&2
    exit 1
  fi
  echo "Building envoy-stats-exporter..."
  REGISTRY="${REGISTRY}" TAG="${TAG}" bash "$ENVOY_STATS_DIR/build.sh"
fi

echo "Building workload images (docker buildx bake)..."
REGISTRY="${REGISTRY}" TAG="${TAG}" BENCH="${BENCH}" docker buildx bake -f docker-bake.hcl

echo "Pushing images..."
PUSH_IMAGES=()
if [ "${SKIP_SIDECAR_BUILD:-}" != "1" ] && [ -n "$SIDECAR_DIR" ]; then
  PUSH_IMAGES+=("${REGISTRY}/sidecar-sidecar:${TAG}")
fi

PUSH_IMAGES+=("${REGISTRY}/${BENCH}-frontend:${TAG}")

PUSH_IMAGES+=("${REGISTRY}/${BENCH}-frontend-grpc:${TAG}")

PUSH_IMAGES+=("${REGISTRY}/${BENCH}-rajomon-client:${TAG}")

printf '%s\0' "${PUSH_IMAGES[@]}" | xargs -0 -P "$(nproc)" -n1 docker push

if [ -n "$STATUS_FILE" ]; then
  mkdir -p "$(dirname "$STATUS_FILE")"
  touch "$STATUS_FILE"
fi
echo "Build complete."
