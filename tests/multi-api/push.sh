#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=multi-api

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

echo "Pushing images..."
docker push "${REGISTRY}/${BENCH}-backend1:${TAG}"
docker push "${REGISTRY}/${BENCH}-backend2:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend-grpc:${TAG}"
docker push "${REGISTRY}/${BENCH}-rajomon-client:${TAG}"

echo "Push complete."
