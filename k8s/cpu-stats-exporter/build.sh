#!/bin/bash
# build.sh - Build and push cpu-stats-exporter Docker image

set -e

# Configuration
REGISTRY="${DOCKER_REGISTRY:-farzad1132}"
IMAGE_NAME="cpu-stats-exporter"
TAG="${VERSION:-latest}"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"

echo "======================================================================"
echo "Building cpu-stats-exporter"
echo "======================================================================"
echo "Image: ${FULL_IMAGE}"
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

# Build Go binary locally first to verify it compiles
echo "Testing Go build..."
go build -o cpu-stats-exporter-test main.go
rm -f cpu-stats-exporter-test
echo "✓ Go build successful"
echo ""

# Build Docker image
echo "Building Docker image..."
docker build -t "${FULL_IMAGE}" .
echo "✓ Docker build successful"
echo ""

# Show image size
echo "Image size:"
docker images "${FULL_IMAGE}" --format "{{.Repository}}:{{.Tag}} - {{.Size}}"
echo ""

# Push to registry
read -p "Push to registry? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Pushing image to registry..."
    docker push "${FULL_IMAGE}"
    echo "✓ Image pushed successfully"
else
    echo "Skipping registry push"
fi

echo ""
echo "======================================================================"
echo "Build complete!"
echo "Image: ${FULL_IMAGE}"
echo "======================================================================"
