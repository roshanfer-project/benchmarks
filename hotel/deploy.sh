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
    cp "${DEPLOY_DIR}/db.yaml" "$TMP_DIR/"
    
    # Image Replacement
    for SERVICE in "frontend" "geo" "profile" "rate" "reservation" "search" "user"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/hotel-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
    done



elif [ "$MODE" == "sidecar" ]; then
    # Sidecar
    # ConfigMap
    # Merge env file and dynamic vars into a temp file
    cat hotel/sidecar.env > "$TMP_DIR/sidecar_merged.env"
    echo "" >> "$TMP_DIR/sidecar_merged.env"
    echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"

    kubectl create configmap hotel-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    cp "${DEPLOY_DIR}/sidecar-configs.yaml" "$TMP_DIR/"
    cp "${DEPLOY_DIR}/db.yaml" "$TMP_DIR/"

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
else
    # rajomon or dagor
    # ConfigMap
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

        # Merge env file and dynamic vars into a temp file to avoid kubectl error
        # "from-env-file cannot be combined with from-file or from-literal"
        cat hotel/rajomon.env > "$TMP_DIR/rajomon_merged.env"
        echo "" >> "$TMP_DIR/rajomon_merged.env"
        echo "priceUpdateRate=$PRICE_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
        echo "latencyThreshold=$LATENCY_THRESHOLD" >> "$TMP_DIR/rajomon_merged.env"
        echo "tokenUpdateRate=$TOKEN_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
        echo "priceStep=$PRICE_STEP" >> "$TMP_DIR/rajomon_merged.env"
        echo "tokenUpdateStep=$TOKEN_UPDATE_STEP" >> "$TMP_DIR/rajomon_merged.env"

        kubectl create configmap hotel-config \
            --from-env-file="$TMP_DIR/rajomon_merged.env" \
            --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    else
        # dagor
        # Merge env file and dynamic vars into a temp file
        cat hotel/dagor.env > "$TMP_DIR/dagor_merged.env"
        echo "" >> "$TMP_DIR/dagor_merged.env"
        
        # Inject Alpha and Beta if they exist in shell environment
        if [ ! -z "$Alpha" ]; then
            echo "Alpha=$Alpha" >> "$TMP_DIR/dagor_merged.env"
        fi
        if [ ! -z "$Beta" ]; then
            echo "Beta=$Beta" >> "$TMP_DIR/dagor_merged.env"
        fi

        kubectl create configmap hotel-config --from-env-file="$TMP_DIR/dagor_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    fi
    
    # App Manifests
    cp "${DEPLOY_DIR}/app-grpc.yaml" "$TMP_DIR/app.yaml"
    cp "${DEPLOY_DIR}/db.yaml" "$TMP_DIR/"

    # Image Replacement
    # Common services
    for SERVICE in "geo" "profile" "rate" "reservation" "search" "user"; do
        sed -i "s|${SERVICE}:latest|${REGISTRY}/hotel-${SERVICE}:${TAG}|g" "${TMP_DIR}/app.yaml"
    done
    # New services
    sed -i "s|frontend-grpc:latest|${REGISTRY}/hotel-frontend-grpc:${TAG}|g" "${TMP_DIR}/app.yaml"
    sed -i "s|rajomon-client:latest|${REGISTRY}/hotel-rajomon-client:${TAG}|g" "${TMP_DIR}/app.yaml"
fi

# 4. Apply Manifests with Order
echo "Applying manifests..."

# Apply ConfigMaps
kubectl apply -f "$TMP_DIR/configmap.yaml"
if [ "$MODE" == "sidecar" ]; then
    kubectl apply -f "$TMP_DIR/sidecar-configs.yaml"
fi

# Apply DBs
echo "Deploying Databases..."
kubectl apply -f "$TMP_DIR/db.yaml"
echo "Waiting for Databases to be ready..."
# Wait for some key DBs
kubectl wait --for=condition=ready pod/mongodb-geo --timeout=60s
kubectl wait --for=condition=ready pod/mongodb-profile --timeout=60s
kubectl wait --for=condition=ready pod/memcached-profile --timeout=60s

# Apply Prometheus
echo "Deploying Prometheus and Pushgateway..."
kubectl apply -f "${DEPLOY_DIR}/prometheus.yaml"
kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s
# We don't necessarily need to wait for Prometheus server to be ready for the app to start, 
# but it's good practice.
kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s


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
    if [ "$MODE" == "rajomon" ] || [ "$MODE" == "dagor" ]; then
         apply_service "frontend-grpc" "$TMP_DIR/app.yaml"
         echo "Waiting for Frontend GRPC..."
         kubectl wait --for=condition=ready pod/frontend-grpc --timeout=30s
         
         apply_service "rajomon-client" "$TMP_DIR/app.yaml"
         echo "Waiting for Rajomon Client..."
         kubectl wait --for=condition=ready pod/rajomon-client --timeout=30s
    else
         apply_service "frontend" "$TMP_DIR/app.yaml"
         echo "Waiting for Frontend service to be ready..."
         kubectl wait --for=condition=ready pod/frontend --timeout=30s
    fi
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
