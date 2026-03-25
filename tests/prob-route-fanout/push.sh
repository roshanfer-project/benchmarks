#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=prob-route-fanout

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

echo "Pushing images..."
docker push "${REGISTRY}/${BENCH}-backend1:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend2:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend3:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend4:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend5:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend6:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend7:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend:${TAG}"

echo "Push complete."
