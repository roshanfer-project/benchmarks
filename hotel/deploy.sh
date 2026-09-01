#!/bin/bash
set -e

# Usage: ./deploy.sh [roshanfer|plain] [--skip-build]
# Default settings
MODE="${SYSTEM:-roshanfer}"
SKIP_BUILD=true
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-$(date +%Y-%m-%d)}

# Parse arguments
for arg in "$@"; do
    case $arg in
        roshanfer) MODE="roshanfer";;
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

kubectl_wait_ready_or_fail() {
  local app=$1
  local to=$2
  if kubectl wait --for=condition=Ready pod -l "app=${app}" --timeout="${to}s"; then
    return 0
  fi
  echo "=== deploy.sh: kubectl wait failed for app=${app} (timeout=${to}s) ===" >&2
  kubectl get pods -l "app=${app}" -o wide >&2 || true
  kubectl describe pod -l "app=${app}" >&2 || true
  local p
  while IFS= read -r p; do
    [ -z "$p" ] && continue
    echo "=== logs ${p} (current) ===" >&2
    kubectl logs "$p" --all-containers=true --tail=200 >&2 || true
    echo "=== logs ${p} (previous) ===" >&2
    kubectl logs "$p" --all-containers=true --previous --tail=200 >&2 || true
  done < <(kubectl get pods -l "app=${app}" -o name 2>/dev/null)
  exit 1
}

patch_ingress_aimd_key() {
  local key=$1
  local val=$2
  local f=$3
  [ -z "$val" ] && return 0
  echo "deploy.sh: ${key}=${val} (patch ingress.yaml in sidecar-configs)"
  export AIMD_KEY="$key" AIMD_VAL="$val"
  if grep -qE "^[[:space:]]*${key}:" "$f"; then
    perl -i -pe 's/^(\s*)\Q$ENV{AIMD_KEY}\E:\s*\S+/${1}$ENV{AIMD_KEY}: $ENV{AIMD_VAL}/' "$f"
  else
    perl -i -pe 's/^(\s*)name:\s*ingress\s*$/${1}name: ingress\n${1}$ENV{AIMD_KEY}: $ENV{AIMD_VAL}/' "$f"
  fi
}

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



elif [ "$MODE" == "roshanfer" ]; then
    # Roshanfer
    # ConfigMap
    # Merge env file and dynamic vars into a temp file
    cat hotel/sidecar.env > "$TMP_DIR/sidecar_merged.env"
    echo "" >> "$TMP_DIR/sidecar_merged.env"
    echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"

    kubectl create configmap hotel-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
    cp "${DEPLOY_DIR}/sidecar-configs.yaml" "$TMP_DIR/"

    if [ ! -z "$FRONTEND_SEARCH_OC" ]; then
        echo "Overriding over_commitment for frontend and search to $FRONTEND_SEARCH_OC"
        perl -i -0777 -pe "s/(over_commitment:\s*)[0-9.]+(\s*\n\s*name:\s*(?:frontend|search))/\${1}${FRONTEND_SEARCH_OC}\${2}/g" "$TMP_DIR/sidecar-configs.yaml"
    fi

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
if [ "$MODE" == "roshanfer" ]; then
    patch_ingress_aimd_key aimd_err_d "${SIDECAR_AIMD_ERR_D:-}" "$TMP_DIR/sidecar-configs.yaml"
    patch_ingress_aimd_key aimd_err_i "${SIDECAR_AIMD_ERR_I:-}" "$TMP_DIR/sidecar-configs.yaml"
    patch_ingress_aimd_key aimd_adj_d "${SIDECAR_AIMD_ADJ_D:-}" "$TMP_DIR/sidecar-configs.yaml"
    patch_ingress_aimd_key aimd_adj_i "${SIDECAR_AIMD_ADJ_I:-}" "$TMP_DIR/sidecar-configs.yaml"
    patch_ingress_aimd_key safe_multiply "${SIDECAR_SAFE_MULTIPLY:-}" "$TMP_DIR/sidecar-configs.yaml"
    kubectl apply -f "$TMP_DIR/sidecar-configs.yaml"
fi

# Apply DBs
echo "Deploying Databases..."
kubectl apply -f "$TMP_DIR/db.yaml"
echo "Waiting for Databases to be ready..."
# Wait for some key DBs
kubectl_wait_ready_or_fail mongodb-geo 60
kubectl_wait_ready_or_fail mongodb-profile 60
kubectl_wait_ready_or_fail memcached-profile 60

# Apply Prometheus
echo "Deploying Prometheus and Pushgateway..."
kubectl apply -f "${DEPLOY_DIR}/prometheus.yaml"
kubectl_wait_ready_or_fail prometheus-pushgateway 60
# We don't necessarily need to wait for Prometheus server to be ready for the app to start, 
# but it's good practice.
kubectl_wait_ready_or_fail prometheus 60


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

WAIT_TIMEOUT=${WAIT_TIMEOUT:-60}

if [ -f "$TMP_DIR/app.yaml" ]; then
    if [ "$MODE" == "roshanfer" ]; then
        for SVC in "geo" "rate" "profile" "reservation" "user"; do
            apply_service $SVC "$TMP_DIR/app.yaml"
        done
        echo "Waiting for Leaf services to be ready..."
        kubectl_wait_ready_or_fail geo 30
        kubectl_wait_ready_or_fail rate 30
        kubectl_wait_ready_or_fail profile 30
        kubectl_wait_ready_or_fail reservation 30
        kubectl_wait_ready_or_fail user 30

        apply_service "search" "$TMP_DIR/app.yaml"
        echo "Waiting for Search service to be ready..."
        kubectl_wait_ready_or_fail search 30

        apply_service "frontend" "$TMP_DIR/app.yaml"
        echo "Waiting for Frontend service to be ready..."
        kubectl_wait_ready_or_fail frontend 30
    else
        deploy_fail=0
        declare -a deploy_pids=()
        if [ "$MODE" == "rajomon" ] || [ "$MODE" == "dagor" ]; then
            PARALLEL_SVCS=(geo rate profile reservation user search frontend-grpc rajomon-client)
        else
            PARALLEL_SVCS=(geo rate profile reservation user search frontend)
        fi
        for SVC in "${PARALLEL_SVCS[@]}"; do
            (
                kubectl apply -f "$TMP_DIR/app.yaml" -l app="$SVC"
                kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
            ) &
            deploy_pids+=($!)
        done
        for pid in "${deploy_pids[@]}"; do
            wait "$pid" || deploy_fail=1
        done
        if [ "$deploy_fail" -ne 0 ]; then
            echo "deploy.sh (${MODE}): one or more workloads failed readiness" >&2
            exit 1
        fi
    fi
fi

# Ingress
if [ "$MODE" == "roshanfer" ]; then
    echo "Deploying Ingress..."
    kubectl apply -f "$TMP_DIR/ingress.yaml"
    echo "Waiting for Ingress to be ready..."
    kubectl_wait_ready_or_fail ingress 30
fi

# Apply Services (Idempotent re-apply to ensure Service objects are created)
kubectl apply -f "$TMP_DIR/app.yaml"

rm -rf "$TMP_DIR"
echo "Deployment complete."
