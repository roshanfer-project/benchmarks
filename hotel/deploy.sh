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

echo "Deploying Hotel Benchmark in $MODE mode"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"


# 3. Prepare Manifests
DEPLOY_DIR="hotel/k8s"
TMP_DIR="${DEPLOY_DIR}/tmp_apply"
mkdir -p "$TMP_DIR"

if [ "$MODE" == "plain" ]; then
    # ConfigMap
    kubectl create configmap hotel-config --from-env-file=hotel/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    
    # App Manifests
    cp "${DEPLOY_DIR}/app-plain.yaml" "$TMP_DIR/app.yaml"
    
    # Image Replacement
    for SERVICE in "frontend" "geo" "profile" "rate" "reservation" "search" "user"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/hotel-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
    done

else # sidecar
    # ConfigMap
    kubectl create configmap hotel-config --from-env-file=hotel/sidecar.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    cp "${DEPLOY_DIR}/sidecar-configs.yaml" "$TMP_DIR/"

    # App Manifests
    cp "${DEPLOY_DIR}/app-sidecar.yaml" "$TMP_DIR/app.yaml"
    
    # Ingress Manifest
    cp "${DEPLOY_DIR}/ingress.yaml" "$TMP_DIR/"

    # Image Replacement
    for SERVICE in "frontend" "geo" "profile" "rate" "reservation" "search" "user"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/hotel-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
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

# Function to apply specific resource from a multi-doc yaml
apply_service() {
    local service=$1
    local file=$2
    # This assumes consistent naming or we apply the whole file but that breaks order.
    # To strictly enforce order, we should have split files or use label selection.
    # With one big file, we can't easily select only "geo" pods without splitting.
    # But since they are independent pods in the yaml, applying the whole file applies them "at once".
    # K8s applies in order of appearance? Not guaranteed.
    # To truly enforce order, we must select resources. 
    # Let's rely on kubectl partial application via -l app=<service> if labels are present.
    # The manifests use labels: app: <service>
    
    echo "Deploying $service..."
    kubectl apply -f "$file" -l app=$service
    
    # Wait for ready?
    # User didn't strictly ask for wait, but dependency based deployment usually implies waiting.
    # Let's wait for pod readiness.
    # kubectl wait --for=condition=ready pod -l app=$service --timeout=60s
}

if [ -f "$TMP_DIR/app.yaml" ]; then
    # Leaves
    for SVC in "geo" "rate" "profile" "reservation" "user"; do
        apply_service $SVC "$TMP_DIR/app.yaml"
    done
    echo "Waiting for Leaf services to be ready..."
    kubectl wait --for=condition=ready pod/geo --timeout=30s
    kubectl wait --for=condition=ready pod/rate --timeout=30s
    kubectl wait --for=condition=ready pod/profile --timeout=30s
    kubectl wait --for=condition=ready pod/reservation --timeout=30s
    kubectl wait --for=condition=ready pod/user --timeout=30s
    
    # Intermediate
    apply_service "search" "$TMP_DIR/app.yaml"
    echo "Waiting for Search service to be ready..."
    kubectl wait --for=condition=ready pod/search --timeout=30s
    
    # Root
    apply_service "frontend" "$TMP_DIR/app.yaml"
    echo "Waiting for Frontend service to be ready..."
    kubectl wait --for=condition=ready pod/frontend --timeout=30s
fi

# Ingress
if [ "$MODE" == "sidecar" ]; then
    echo "Deploying Ingress..."
    kubectl apply -f "$TMP_DIR/ingress.yaml"
    echo "Waiting for Ingress to be ready..."
    kubectl wait --for=condition=ready pod/ingress --timeout=30s
fi

# Apply Services (Idempotent re-apply to ensure Service objects are created)
kubectl apply -f "$TMP_DIR/app.yaml"

rm -rf "$TMP_DIR"
echo "Deployment complete."
