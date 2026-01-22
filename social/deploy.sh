#!/bin/bash
set -e

# Usage: ./deploy.sh [sidecar|plain] [--skip-build]
# Default settings
MODE="${SYSTEM:-sidecar}"
SKIP_BUILD=true
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-$(date +%Y-%m-%d)}

# Parse arguments
for arg in "$@"; do
    case $arg in
        sidecar) MODE="sidecar";;
        plain) MODE="plain";;
    esac
done

echo "Deploying Social Benchmark in $MODE mode"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# 3. Prepare Manifests
DEPLOY_DIR="social/k8s"
TMP_DIR="${DEPLOY_DIR}/tmp_apply"
mkdir -p "$TMP_DIR"

if [ "$MODE" == "plain" ]; then
    # ConfigMap Generation
    kubectl create configmap social-config --from-env-file=social/k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    
    # App Manifests
    cp "${DEPLOY_DIR}/app-plain.yaml" "$TMP_DIR/app.yaml"
    cp "${DEPLOY_DIR}/redis.yaml" "$TMP_DIR/"
    
    # Image Replacement
    for SERVICE in "graph" "posts" "home" "user" "compose" "nginx"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/social-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
    done

else # sidecar
    # ConfigMap Generation
    kubectl create configmap social-config --from-env-file=social/k8s/sidecar.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"

    # ConfigMap for sidecars
    cp "${DEPLOY_DIR}/sidecar-configs.yaml" "$TMP_DIR/"
    cp "${DEPLOY_DIR}/redis.yaml" "$TMP_DIR/"

    # App Manifests
    cp "${DEPLOY_DIR}/app-sidecar.yaml" "$TMP_DIR/app.yaml"
    
    # Ingress Manifest
    cp "${DEPLOY_DIR}/ingress.yaml" "$TMP_DIR/"

    # Image Replacement
    for SERVICE in "graph" "posts" "home" "user" "compose" "nginx"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/social-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
    done
    
    sed -i "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "${TMP_DIR}/app.yaml"
    sed -i "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "${TMP_DIR}/ingress.yaml"
fi

# 4. Apply Manifests with Order
echo "Applying manifests..."

# Apply ConfigMaps
kubectl apply -f "$TMP_DIR/configmap.yaml"
if [ "$MODE" == "sidecar" ]; then
    kubectl apply -f "$TMP_DIR/sidecar-configs.yaml"
fi

# Apply Redis
echo "Deploying Redis..."
kubectl apply -f "$TMP_DIR/redis.yaml"
echo "Waiting for Redis to be ready..."
kubectl wait --for=condition=available deployment/redis --timeout=60s || true

# Apply Services (Loop for dependency wait logic, though robust dependency handling is better via init containers or retry loops in apps)
# Assuming apps have retry logic or we wait.

# Function to apply specific resource
apply_service() {
    local service=$1
    local file=$2
    echo "Deploying $service..."
    kubectl apply -f "$file" -l app=$service
}

if [ -f "$TMP_DIR/app.yaml" ]; then
    # Leaves
    for SVC in "graph" "posts" "user"; do
        apply_service $SVC "$TMP_DIR/app.yaml"
    done
    echo "Waiting for Leaf services to be ready..."
    kubectl wait --for=condition=ready pod/graph --timeout=60s || true
    kubectl wait --for=condition=ready pod/posts --timeout=60s || true
    kubectl wait --for=condition=ready pod/user --timeout=60s || true
    
    # Intermediate
    apply_service "home" "$TMP_DIR/app.yaml"
    apply_service "compose" "$TMP_DIR/app.yaml"
    echo "Waiting for Intermediate services to be ready..."
    kubectl wait --for=condition=ready pod/home --timeout=60s || true
    kubectl wait --for=condition=ready pod/compose --timeout=60s || true
    
    # Root
    apply_service "nginx" "$TMP_DIR/app.yaml"
    echo "Waiting for Nginx service to be ready..."
    kubectl wait --for=condition=ready pod/nginx --timeout=60s || true
fi

# Ingress
if [ "$MODE" == "sidecar" ]; then
    echo "Deploying Ingress..."
    kubectl apply -f "$TMP_DIR/ingress.yaml"
    echo "Waiting for Ingress to be ready..."
    kubectl wait --for=condition=ready pod/ingress --timeout=30s || true
fi

# Apply Services (Idempotent re-apply for Services)
kubectl apply -f "$TMP_DIR/app.yaml"

rm -rf "$TMP_DIR"
echo "Deployment complete."
