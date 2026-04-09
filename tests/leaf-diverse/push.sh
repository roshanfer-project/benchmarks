#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=leaf-diverse

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

echo "Pushing images..."
docker push "${REGISTRY}/${BENCH}-frontend:${TAG}"
docker push "${REGISTRY}/${BENCH}-frontend-grpc:${TAG}"
docker push "${REGISTRY}/${BENCH}-rajomon-client:${TAG}"

echo "Push complete."
