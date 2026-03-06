#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
STATUS_FILE=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
BENCH=alibaba-large

if [ -n "$STATUS_FILE" ]; then
  STATUS_DIR=$(dirname "$STATUS_FILE")
  STATUS_BASE=$(basename "$STATUS_FILE")
  mkdir -p "$STATUS_DIR"
  STATUS_DIR=$(cd "$STATUS_DIR" && pwd)
  STATUS_FILE="${STATUS_DIR}/${STATUS_BASE}"
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

if [ -d "../sidecar" ]; then
  echo "Building sidecar..."
  (cd ../sidecar && ./build.sh Release)
  docker build -f ../sidecar/Dockerfile -t "${REGISTRY}/sidecar-sidecar:${TAG}" ../sidecar
fi

echo "Building MS_12657..."
docker build --build-arg SERVICE=services/MS_12657 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-12657:${TAG}" .
echo "Building MS_14758..."
docker build --build-arg SERVICE=services/MS_14758 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-14758:${TAG}" .
echo "Building MS_18750..."
docker build --build-arg SERVICE=services/MS_18750 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-18750:${TAG}" .
echo "Building MS_19439..."
docker build --build-arg SERVICE=services/MS_19439 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-19439:${TAG}" .
echo "Building MS_21298..."
docker build --build-arg SERVICE=services/MS_21298 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-21298:${TAG}" .
echo "Building MS_25781..."
docker build --build-arg SERVICE=services/MS_25781 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-25781:${TAG}" .
echo "Building MS_25806..."
docker build --build-arg SERVICE=services/MS_25806 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-25806:${TAG}" .
echo "Building MS_2687..."
docker build --build-arg SERVICE=services/MS_2687 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-2687:${TAG}" .
echo "Building MS_33572..."
docker build --build-arg SERVICE=services/MS_33572 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-33572:${TAG}" .
echo "Building MS_38190..."
docker build --build-arg SERVICE=services/MS_38190 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-38190:${TAG}" .
echo "Building MS_40087..."
docker build --build-arg SERVICE=services/MS_40087 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-40087:${TAG}" .
echo "Building MS_41667..."
docker build --build-arg SERVICE=services/MS_41667 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-41667:${TAG}" .
echo "Building MS_43032..."
docker build --build-arg SERVICE=services/MS_43032 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-43032:${TAG}" .
echo "Building MS_43754..."
docker build --build-arg SERVICE=services/MS_43754 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-43754:${TAG}" .
echo "Building MS_44246..."
docker build --build-arg SERVICE=services/MS_44246 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-44246:${TAG}" .
echo "Building MS_45067..."
docker build --build-arg SERVICE=services/MS_45067 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-45067:${TAG}" .
echo "Building MS_51783..."
docker build --build-arg SERVICE=services/MS_51783 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-51783:${TAG}" .
echo "Building MS_51787..."
docker build --build-arg SERVICE=services/MS_51787 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-51787:${TAG}" .
echo "Building MS_53792..."
docker build --build-arg SERVICE=services/MS_53792 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-53792:${TAG}" .
echo "Building MS_56113..."
docker build --build-arg SERVICE=services/MS_56113 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-56113:${TAG}" .
echo "Building MS_5720..."
docker build --build-arg SERVICE=services/MS_5720 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-5720:${TAG}" .
echo "Building MS_58796..."
docker build --build-arg SERVICE=services/MS_58796 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-58796:${TAG}" .
echo "Building MS_62039..."
docker build --build-arg SERVICE=services/MS_62039 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-62039:${TAG}" .
echo "Building MS_64512..."
docker build --build-arg SERVICE=services/MS_64512 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-64512:${TAG}" .
echo "Building MS_66921..."
docker build --build-arg SERVICE=services/MS_66921 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-66921:${TAG}" .
echo "Building MS_67465..."
docker build --build-arg SERVICE=services/MS_67465 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-67465:${TAG}" .
echo "Building MS_70124..."
docker build --build-arg SERVICE=services/MS_70124 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-70124:${TAG}" .
echo "Building MS_7103..."
docker build --build-arg SERVICE=services/MS_7103 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-7103:${TAG}" .
echo "Building MS_9105..."
docker build --build-arg SERVICE=services/MS_9105 -f Dockerfile -t "${REGISTRY}/${BENCH}-ms-9105:${TAG}" .

echo "Pushing images..."
if [ -d "../sidecar" ]; then
  docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
fi
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

if [ -n "$STATUS_FILE" ]; then
  mkdir -p "$(dirname "$STATUS_FILE")"
  touch "$STATUS_FILE"
fi
echo "Build complete."
