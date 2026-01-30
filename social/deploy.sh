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
        rajomon) MODE="rajomon";;
        dagor) MODE="dagor";;
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

elif [ "$MODE" == "sidecar" ]; then
    # ConfigMap Generation
    # Merge env file and dynamic vars into a temp file
    cat social/k8s/sidecar.env > "$TMP_DIR/sidecar_merged.env"
    echo "" >> "$TMP_DIR/sidecar_merged.env"
    echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"

    kubectl create configmap social-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"

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

elif [ "$MODE" == "rajomon" ] || [ "$MODE" == "dagor" ]; then
    # ConfigMap Generation
    if [ "$MODE" == "rajomon" ]; then
        # Default values for Rajomon Config
        PRICE_UPDATE_RATE=${priceUpdateRate}
        LATENCY_THRESHOLD=${latencyThreshold}
        TOKEN_UPDATE_RATE=${tokenUpdateRate}
        PRICE_STEP=${priceStep}
        TOKEN_UPDATE_STEP=${tokenUpdateStep}

        echo "Using Rajomon Config:"
        echo "  priceUpdateRate=$PRICE_UPDATE_RATE"
        echo "  latencyThreshold=$LATENCY_THRESHOLD"
        echo "  tokenUpdateRate=$TOKEN_UPDATE_RATE"
        echo "  priceStep=$PRICE_STEP"
        echo "  tokenUpdateStep=$TOKEN_UPDATE_STEP"

        # Merge env file and dynamic vars into a temp file
        cat social/k8s/rajomon.env > "$TMP_DIR/rajomon_merged.env"
        echo "" >> "$TMP_DIR/rajomon_merged.env"
        echo "priceUpdateRate=$PRICE_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
        echo "latencyThreshold=$LATENCY_THRESHOLD" >> "$TMP_DIR/rajomon_merged.env"
        echo "tokenUpdateRate=$TOKEN_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
        echo "priceStep=$PRICE_STEP" >> "$TMP_DIR/rajomon_merged.env"
        echo "tokenUpdateStep=$TOKEN_UPDATE_STEP" >> "$TMP_DIR/rajomon_merged.env"

        kubectl create configmap social-config --from-env-file="$TMP_DIR/rajomon_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    else
        # dagor
        # Merge env file and dynamic vars into a temp file
        cat social/k8s/dagor.env > "$TMP_DIR/dagor_merged.env"
        echo "" >> "$TMP_DIR/dagor_merged.env"
        
        # Inject Alpha and Beta if they exist in shell environment
        if [ ! -z "$Alpha" ]; then
            echo "Alpha=$Alpha" >> "$TMP_DIR/dagor_merged.env"
        fi
        if [ ! -z "$Beta" ]; then
            echo "Beta=$Beta" >> "$TMP_DIR/dagor_merged.env"
        fi

        kubectl create configmap social-config --from-env-file="$TMP_DIR/dagor_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    fi
    
    # App Manifests
    cp "${DEPLOY_DIR}/app-grpc.yaml" "$TMP_DIR/app.yaml"
    cp "${DEPLOY_DIR}/redis.yaml" "$TMP_DIR/"

    # Image Replacement
    # Common services
    for SERVICE in "graph" "posts" "home" "user" "compose"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/social-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
    done
    # New services
    sed -i "s|nginx-grpc:latest|${REGISTRY}/social-nginx-grpc:${TAG}|g" "${TMP_DIR}/app.yaml"
    sed -i "s|rajomon-client:latest|${REGISTRY}/social-rajomon-client:${TAG}|g" "${TMP_DIR}/app.yaml"
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

# Apply Prometheus
echo "Deploying Prometheus and Pushgateway..."
kubectl apply -f "${DEPLOY_DIR}/prometheus.yaml"
kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
# We don't necessarily need to wait for Prometheus server to be ready for the app to start, 
# but it's good practice.
kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

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
    if [ "$MODE" == "rajomon" ] || [ "$MODE" == "dagor" ]; then
         apply_service "nginx-grpc" "$TMP_DIR/app.yaml"
         echo "Waiting for Nginx GRPC..."
         kubectl wait --for=condition=ready pod/nginx-grpc --timeout=60s || true
         
         apply_service "rajomon-client" "$TMP_DIR/app.yaml"
         echo "Waiting for Rajomon Client..."
         kubectl wait --for=condition=ready pod/rajomon-client --timeout=60s || true
    else
         apply_service "nginx" "$TMP_DIR/app.yaml"
         echo "Waiting for Nginx service to be ready..."
         kubectl wait --for=condition=ready pod/nginx --timeout=60s || true
    fi
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
