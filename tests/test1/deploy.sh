#!/bin/bash
set -e

# Usage: ./deploy.sh [sidecar|plain] [--skip-build]
# Default settings
MODE="sidecar"
SKIP_BUILD=false
# Default TAG generation, can be overridden by env var
TAG=${TAG:-$(date +%Y-%m-%d)}

# Parse arguments
for arg in "$@"; do
    case $arg in
        sidecar)
            MODE="sidecar"
            ;;
        plain)
            MODE="plain"
            ;;
        --skip-build)
            SKIP_BUILD=true
            ;;
    esac
done

# Default registry
REGISTRY=${REGISTRY:-farzad1132}

echo "Deploying in $MODE mode"
if [ "$SKIP_BUILD" = true ]; then
    echo "Skipping build and push (assuming images exist)"
fi

# Helper to find root directory
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export ROOT_DIR

echo "Using Root Dir: $ROOT_DIR"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"

cd "$ROOT_DIR"

if [ "$SKIP_BUILD" = false ]; then
    # 1. Build Sidecar Binary
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
        echo "Error: sidecar directory not found."
        exit 1
    fi

    # 3. Build App Docker Image
    echo "Building app docker image..."
    docker build -f tests/test1/app/Dockerfile -t "${REGISTRY}/test1-app:${TAG}" tests/test1

    # 4. Push Images
    echo "Pushing images..."
    docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
    docker push "${REGISTRY}/test1-app:${TAG}"
fi

# 5. Prepare Manifests
DEPLOY_DIR="tests/test1/k8s"
TMP_DIR="${DEPLOY_DIR}/tmp_apply"

mkdir -p "$TMP_DIR"

if [ "$MODE" == "plain" ]; then
    # Generate ConfigMap from plain.env
    kubectl create configmap test1-config --from-env-file=tests/test1/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    
    # Copy plain manifests
    cp "${DEPLOY_DIR}/app-plain.yaml" "$TMP_DIR/app.yaml"
    
    # Replace images
    sed -i "s|test1-app:latest|${REGISTRY}/test1-app:${TAG}|g" "${TMP_DIR}/app.yaml"

else
    # Generate ConfigMap from sidecar.env
    kubectl create configmap test1-config --from-env-file=tests/test1/sidecar.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"

    # Copy sidecar manifests
    cp "${DEPLOY_DIR}/sidecar-configs.yaml" "$TMP_DIR/"
    cp "${DEPLOY_DIR}/ingress.yaml" "$TMP_DIR/"
    cp "${DEPLOY_DIR}/app-sidecar.yaml" "$TMP_DIR/app.yaml"

    # Replace images
    sed -i "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "${TMP_DIR}/ingress.yaml"
    sed -i "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "${TMP_DIR}/app.yaml"
    sed -i "s|test1-app:latest|${REGISTRY}/test1-app:${TAG}|g" "${TMP_DIR}/app.yaml"
fi

# 6. Apply to K8s
echo "Applying manifests..."
kubectl apply -f "$TMP_DIR"

# Cleanup
rm -rf "$TMP_DIR"

echo "Deployment completed successfully with tag ${TAG}"
