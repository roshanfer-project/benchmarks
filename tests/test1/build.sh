#!/bin/bash
set -e

# Usage: ./build.sh <TAG> [STATUS_FILE]

if [ -z "$1" ]; then
    echo "Error: TAG argument required"
    exit 1
fi

TAG=$1
STATUS_FILE=$2

# Default registry
REGISTRY=${REGISTRY:-farzad1132}

# Helper to find root directory
# This script is in benchmarks/tests/test1/build.sh
# We want ROOT_DIR to be benchmarks
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export ROOT_DIR

# Ensure STATUS_FILE is absolute path if provided
if [ -n "$STATUS_FILE" ]; then
    STATUS_DIR=$(dirname "$STATUS_FILE")
    STATUS_BASE=$(basename "$STATUS_FILE")
    mkdir -p "$STATUS_DIR"
    STATUS_DIR=$(cd "$STATUS_DIR" && pwd)
    STATUS_FILE="${STATUS_DIR}/${STATUS_BASE}"
fi

echo "Building Test1 with Tag: $TAG"
echo "Root Dir: $ROOT_DIR"

cd "$ROOT_DIR"

# 1. Build Sidecar Binary and Docker Image
if [ -d "sidecar" ]; then
    echo "Entering sidecar directory to build..."
    cd sidecar
    
    if [ -f "build.sh" ]; then
        echo "Building sidecar binary..."
        ./build.sh Release
    else
        echo "Warning: build.sh not found in sidecar directory."
    fi

    # 2. Build Sidecar Docker Image (Context is sidecar directory)
    echo "Building sidecar docker image..."
    docker build -f Dockerfile -t "${REGISTRY}/sidecar-sidecar:${TAG}" .
    
    cd "$ROOT_DIR"
else
    echo "Error: sidecar directory not found in $ROOT_DIR"
    exit 1
fi

# 3. Build App Docker Image
echo "Building app docker image..."
docker build -f tests/test1/app/Dockerfile -t "${REGISTRY}/test1-app:${TAG}" tests/test1

# 4. Push Images
echo "Pushing images..."
docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
docker push "${REGISTRY}/test1-app:${TAG}"

# 5. Update Status File
if [ -n "$STATUS_FILE" ]; then
    echo "Build successful. Marking status in $STATUS_FILE"
    # Ensure dir exists
    mkdir -p "$(dirname "$STATUS_FILE")"
    touch "$STATUS_FILE"
fi

echo "Build completed successfully for tag ${TAG}"
