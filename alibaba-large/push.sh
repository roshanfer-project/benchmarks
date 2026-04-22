#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=alibaba-large

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

echo "Pushing images..."
docker push "${REGISTRY}/${BENCH}-ms-12657:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-14758:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-18750:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-19439:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-21298:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-25781:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-25806:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-2687:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-33572:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-38190:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-40087:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-41667:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-43032:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-43754:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-44246:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-45067:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-51783:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-51787:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-53792:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-56113:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-5720:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-58796:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-62039:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-64512:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-66921:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-67465:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-70124:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-7103:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-9105:${TAG}"
docker push "${REGISTRY}/${BENCH}-ms-64512-grpc:${TAG}"
docker push "${REGISTRY}/${BENCH}-rajomon-client:${TAG}"

echo "Push complete."
