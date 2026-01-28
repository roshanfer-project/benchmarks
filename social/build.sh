#!/bin/bash
set -e

# Usage: ./build.sh <TAG> [STATUS_FILE]

if [ -z "$1" ]; then
    echo "Error: TAG argument required"
    exit 1
fi

TAG=$1
STATUS_FILE=$2

# Default Registry
REGISTRY=${REGISTRY:-farzad1132}

# Helper to find root directory
# This script is in benchmarks/social/build.sh
# We want ROOT_DIR to be benchmarks directory
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export ROOT_DIR

# Ensure STATUS_FILE is absolute if provided, before changing directory
if [ -n "$STATUS_FILE" ]; then
    # Handle case where file doesn't exist yet
    STATUS_DIR=$(dirname "$STATUS_FILE")
    STATUS_BASE=$(basename "$STATUS_FILE")
    mkdir -p "$STATUS_DIR"
    STATUS_DIR=$(cd "$STATUS_DIR" && pwd)
    STATUS_FILE="${STATUS_DIR}/${STATUS_BASE}"
fi

echo "Building Social Benchmark with Tag: $TAG"
echo "Root Dir: $ROOT_DIR"

cd "$ROOT_DIR"

# 1. Build Sidecar Binary and Docker Image
echo "Building Sidecar..."
if [ -d "sidecar" ]; then
    cd sidecar
    if [ -f "build.sh" ]; then
        ./build.sh Release
    fi
    docker build -f Dockerfile -t "${REGISTRY}/sidecar-sidecar:${TAG}" .
    cd "$ROOT_DIR"
else
    echo "Error: sidecar directory not found in $ROOT_DIR"
    exit 1
fi

# 2. Build Social Services
echo "Building Social Services..."
# Services based on folders in social/
SERVICES=("graph" "posts" "home" "user" "compose" "nginx" "nginx-grpc" "rajomon-client")

for SERVICE in "${SERVICES[@]}"; do
    echo "Building $SERVICE..."
    # Build using the unified Dockerfile in social directory
    docker build --build-arg SERVICE=$SERVICE -f social/Dockerfile -t "${REGISTRY}/social-${SERVICE}:${TAG}" social
done

# 3. Push Images
echo "Pushing images..."
docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
for SERVICE in "${SERVICES[@]}"; do
    docker push "${REGISTRY}/social-${SERVICE}:${TAG}"
done

# 4. Update Status File
if [ -n "$STATUS_FILE" ]; then
    echo "Build successful. Marking status in $STATUS_FILE"
    mkdir -p "$(dirname "$STATUS_FILE")"
    touch "$STATUS_FILE"
fi

echo "Build completed successfully for tag ${TAG}"
