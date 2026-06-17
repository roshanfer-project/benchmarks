#!/bin/bash
set -e
MODE=${1:-${SYSTEM:-plain}}
ARG2=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH=multi-api
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

if [ "$MODE" = "plain" ] && [ "$ARG2" = "debug" ]; then
  echo "deploy.sh: debug only with sidecar or sidecar-lb; use ./deploy.sh sidecar debug or ./deploy.sh sidecar-lb debug" >&2
  exit 1
fi
if [ "$MODE" = "plain-lb" ] && [ -n "$ARG2" ]; then
  echo "deploy.sh: plain-lb mode does not take a second argument" >&2
  exit 1
fi
if [ "$MODE" = "sidecar" ] && [ -n "$ARG2" ] && [ "$ARG2" != "debug" ]; then
  echo "deploy.sh: unknown second argument: $ARG2 (expected: debug)" >&2
  exit 1
fi
if [ "$MODE" = "sidecar-lb" ] && [ -n "$ARG2" ] && [ "$ARG2" != "debug" ]; then
  echo "deploy.sh: unknown second argument: $ARG2 (expected: debug)" >&2
  exit 1
fi
if { [ "$MODE" = "rajomon" ] || [ "$MODE" = "dagor" ] || [ "$MODE" = "dagor-lb" ] || [ "$MODE" = "rajomon-lb" ]; } && [ -n "$ARG2" ]; then
  echo "deploy.sh: rajomon, dagor, dagor-lb, and rajomon-lb modes do not take a second argument" >&2
  exit 1
fi
SIDECAR_DEBUG=0
if { [ "$MODE" = "sidecar" ] || [ "$MODE" = "sidecar-lb" ]; } && [ "$ARG2" = "debug" ]; then
  SIDECAR_DEBUG=1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
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

sidecar_debug_require_yq() {
  command -v yq >/dev/null 2>&1 || {
    echo "deploy.sh sidecar/sidecar-lb debug needs mikefarah yq v4: https://github.com/mikefarah/yq" >&2
    exit 1
  }
}

sidecar_debug_merge_glog_file() {
  [ ! -f k8s/sidecar-debug-glog.env ] && return 0
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in ''|\#*) continue ;; esac
    k="${line%%=*}"
    v="${line#*=}"
    case "$k" in
      SIDECAR_GLOG_V) [ -z "${SIDECAR_GLOG_V+x}" ] && export SIDECAR_GLOG_V="$v" ;;
      SIDECAR_GLOG_VMODULE) [ -z "${SIDECAR_GLOG_VMODULE+x}" ] && export SIDECAR_GLOG_VMODULE="$v" ;;
    esac
  done < k8s/sidecar-debug-glog.env
}

sidecar_debug_patch_workload_yaml() {
  local f=$1
  yq eval-all 'select(.kind == "Pod") |= (.spec.restartPolicy = "Never")' -i "$f"
  export GV_VAL="${SIDECAR_GLOG_V:-}"
  export VM_VAL="${SIDECAR_GLOG_VMODULE:-}"
  if [ -n "$GV_VAL" ] || [ -n "$VM_VAL" ]; then
    yq eval-all '
select(.kind == "Pod") |= (.spec.containers |= map(
  select(.name == "sidecar") |= (
    (.env // []) as $e |
    ($e | map(select(.name != "GLOG_v" and .name != "GLOG_vmodule"))) as $base |
    ([{"name":"GLOG_v","value":strenv(GV_VAL)},{"name":"GLOG_vmodule","value":strenv(VM_VAL)}]
      | map(select(.value != ""))) as $add |
    .env = $base + $add
  )
)) |
select(.kind == "Deployment") |= (.spec.template.spec.containers |= map(
  select(.name == "sidecar") |= (
    (.env // []) as $e |
    ($e | map(select(.name != "GLOG_v" and .name != "GLOG_vmodule"))) as $base |
    ([{"name":"GLOG_v","value":strenv(GV_VAL)},{"name":"GLOG_vmodule","value":strenv(VM_VAL)}]
      | map(select(.value != ""))) as $add |
    .env = $base + $add
  )
))' -i "$f"
  fi
}

if [ "$MODE" = "sidecar" ]; then
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_require_yq
    sidecar_debug_merge_glog_file
  fi
  cat k8s/sidecar.env > "$TMP_DIR/sidecar_merged.env"
  echo "" >> "$TMP_DIR/sidecar_merged.env"
  echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  cp k8s/manifests/sidecar-configs.yaml "$TMP_DIR/sidecar-configs.yaml"
  if [ -n "${SIDECAR_OVER_COMMIT:-}" ]; then
    echo "deploy.sh: SIDECAR_OVER_COMMIT=${SIDECAR_OVER_COMMIT} (patch all over_commitment in sidecar-configs)"
    export SIDECAR_OVER_COMMIT
    perl -i -pe 's/(over_commitment:\s*)[\d.]+/${1}$ENV{SIDECAR_OVER_COMMIT}/g' "$TMP_DIR/sidecar-configs.yaml"
    awk -v want="$SIDECAR_OVER_COMMIT" '
      /over_commitment:/ {
        if ($2 != want) {
          print "deploy.sh: ERROR expected over_commitment " want " got " $2 > "/dev/stderr"
          exit 1
        }
      }
    ' "$TMP_DIR/sidecar-configs.yaml"
  fi
  kubectl apply -f "$TMP_DIR/sidecar-configs.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  for SVC in backend1 backend2 frontend; do
    sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "k8s/manifests/${SVC}-sidecar.yaml" | \
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" > "$TMP_DIR/${SVC}-sidecar.yaml"
    if [ "$SIDECAR_DEBUG" = "1" ]; then
      sidecar_debug_patch_workload_yaml "$TMP_DIR/${SVC}-sidecar.yaml"
    fi
    kubectl apply -f "$TMP_DIR/${SVC}-sidecar.yaml"
    kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
  done

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_patch_workload_yaml "$TMP_DIR/ingress.yaml"
  fi
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl_wait_ready_or_fail ingress 30
elif [ "$MODE" = "sidecar-lb" ]; then
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_require_yq
    sidecar_debug_merge_glog_file
  fi
  cat k8s/sidecar-lb.env > "$TMP_DIR/sidecar_lb_merged.env"
  echo "" >> "$TMP_DIR/sidecar_lb_merged.env"
  echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_lb_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/sidecar_lb_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  cp k8s/manifests/sidecar-lb-configs.yaml "$TMP_DIR/sidecar-lb-configs.yaml"
  if [ -n "${SIDECAR_OVER_COMMIT:-}" ]; then
    echo "deploy.sh: SIDECAR_OVER_COMMIT=${SIDECAR_OVER_COMMIT} (patch all over_commitment in sidecar-lb-configs)"
    export SIDECAR_OVER_COMMIT
    perl -i -pe 's/(over_commitment:\s*)[\d.]+/${1}$ENV{SIDECAR_OVER_COMMIT}/g' "$TMP_DIR/sidecar-lb-configs.yaml"
    awk -v want="$SIDECAR_OVER_COMMIT" '
      /over_commitment:/ {
        if ($2 != want) {
          print "deploy.sh: ERROR expected over_commitment " want " got " $2 > "/dev/stderr"
          exit 1
        }
      }
    ' "$TMP_DIR/sidecar-lb-configs.yaml"
  fi
  kubectl apply -f "$TMP_DIR/sidecar-lb-configs.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  for SVC in backend1 backend2 frontend; do
    sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "k8s/manifests/${SVC}-sidecar-lb.yaml" | \
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" > "$TMP_DIR/${SVC}-sidecar-lb.yaml"
    if [ "$SIDECAR_DEBUG" = "1" ]; then
      sidecar_debug_patch_workload_yaml "$TMP_DIR/${SVC}-sidecar-lb.yaml"
    fi
    kubectl apply -f "$TMP_DIR/${SVC}-sidecar-lb.yaml"
    kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
  done

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress-lb.yaml > "$TMP_DIR/ingress-lb.yaml"
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_patch_workload_yaml "$TMP_DIR/ingress-lb.yaml"
  fi
  kubectl apply -f "$TMP_DIR/ingress-lb.yaml"
  kubectl_wait_ready_or_fail ingress 30
elif [ "$MODE" = "envoy" ]; then
  ENVOY_STATS_IMAGE="${REGISTRY}/envoy-stats-exporter:${TAG}"

  cat k8s/envoy.env > "$TMP_DIR/envoy_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/envoy_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/envoy-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  for SVC in backend1 backend2 frontend; do
    sed -e "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" \
        -e "s|envoy-stats-exporter:latest|${ENVOY_STATS_IMAGE}|g" \
        "k8s/manifests/${SVC}-envoy.yaml" > "$TMP_DIR/${SVC}-envoy.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend; do
    (
      kubectl apply -f "$TMP_DIR/${SVC}-envoy.yaml"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (envoy): one or more workloads failed readiness" >&2
    exit 1
  fi

  sed -e "s|envoy-stats-exporter:latest|${ENVOY_STATS_IMAGE}|g" \
      k8s/manifests/ingress-envoy.yaml > "$TMP_DIR/ingress-envoy.yaml"
  kubectl apply -f "$TMP_DIR/ingress-envoy.yaml"
  kubectl_wait_ready_or_fail ingress 30
elif [ "$MODE" = "rajomon" ]; then
  PRICE_UPDATE_RATE=${priceUpdateRate}
  LATENCY_THRESHOLD=${latencyThreshold}
  TOKEN_UPDATE_RATE=${tokenUpdateRate}
  PRICE_STEP=${priceStep}
  TOKEN_UPDATE_STEP=${tokenUpdateStep}
  echo "Using Rajomon config:"
  echo "  priceUpdateRate=$PRICE_UPDATE_RATE latencyThreshold=$LATENCY_THRESHOLD tokenUpdateRate=$TOKEN_UPDATE_RATE priceStep=$PRICE_STEP tokenUpdateStep=$TOKEN_UPDATE_STEP"
  cat k8s/rajomon.env > "$TMP_DIR/rajomon_merged.env"
  echo "" >> "$TMP_DIR/rajomon_merged.env"
  echo "priceUpdateRate=$PRICE_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
  echo "latencyThreshold=$LATENCY_THRESHOLD" >> "$TMP_DIR/rajomon_merged.env"
  echo "tokenUpdateRate=$TOKEN_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
  echo "priceStep=$PRICE_STEP" >> "$TMP_DIR/rajomon_merged.env"
  echo "tokenUpdateStep=$TOKEN_UPDATE_STEP" >> "$TMP_DIR/rajomon_merged.env"
  K8S_NS=${K8S_NS:-$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)}
  K8S_NS=${K8S_NS:-default}
  sed -i "s|=${BENCH}-\([^=]*\):2000|=${BENCH}-\1.${K8S_NS}.svc.cluster.local:2000|g" "$TMP_DIR/rajomon_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/rajomon_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  cp k8s/manifests/app-grpc.yaml "$TMP_DIR/app-grpc.yaml"
  for IMG in backend1 backend2 frontend-grpc rajomon-client; do
    sed -i "s|${BENCH}-${IMG}:latest|${REGISTRY}/${BENCH}-${IMG}:${TAG}|g" "$TMP_DIR/app-grpc.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend-grpc rajomon-client; do
    (
      kubectl apply -f "$TMP_DIR/app-grpc.yaml" -l app="${SVC}"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (rajomon): one or more workloads failed readiness" >&2
    exit 1
  fi
elif [ "$MODE" = "dagor" ]; then
  cat k8s/dagor.env > "$TMP_DIR/dagor_merged.env"
  echo "" >> "$TMP_DIR/dagor_merged.env"
  if [ -n "$Alpha" ]; then
    echo "Alpha=$Alpha" >> "$TMP_DIR/dagor_merged.env"
  fi
  if [ -n "$Beta" ]; then
    echo "Beta=$Beta" >> "$TMP_DIR/dagor_merged.env"
  fi
  K8S_NS=${K8S_NS:-$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)}
  K8S_NS=${K8S_NS:-default}
  sed -i "s|=${BENCH}-\([^=]*\):2000|=${BENCH}-\1.${K8S_NS}.svc.cluster.local:2000|g" "$TMP_DIR/dagor_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/dagor_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  cp k8s/manifests/app-grpc.yaml "$TMP_DIR/app-grpc.yaml"
  for IMG in backend1 backend2 frontend-grpc rajomon-client; do
    sed -i "s|${BENCH}-${IMG}:latest|${REGISTRY}/${BENCH}-${IMG}:${TAG}|g" "$TMP_DIR/app-grpc.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend-grpc rajomon-client; do
    (
      kubectl apply -f "$TMP_DIR/app-grpc.yaml" -l app="${SVC}"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (dagor): one or more workloads failed readiness" >&2
    exit 1
  fi
elif [ "$MODE" = "dagor-lb" ]; then
  cat k8s/dagor-lb.env > "$TMP_DIR/dagor_lb_merged.env"
  echo "" >> "$TMP_DIR/dagor_lb_merged.env"
  if [ -n "$Alpha" ]; then
    echo "Alpha=$Alpha" >> "$TMP_DIR/dagor_lb_merged.env"
  fi
  if [ -n "$Beta" ]; then
    echo "Beta=$Beta" >> "$TMP_DIR/dagor_lb_merged.env"
  fi
  K8S_NS=${K8S_NS:-$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)}
  K8S_NS=${K8S_NS:-default}
  sed -i "s|=${BENCH}-\([^=]*\):2000|=${BENCH}-\1.${K8S_NS}.svc.cluster.local:2000|g" "$TMP_DIR/dagor_lb_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/dagor_lb_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  cp k8s/manifests/app-grpc-lb.yaml "$TMP_DIR/app-grpc-lb.yaml"
  for IMG in backend1 backend2 frontend-grpc rajomon-client; do
    sed -i "s|${BENCH}-${IMG}:latest|${REGISTRY}/${BENCH}-${IMG}:${TAG}|g" "$TMP_DIR/app-grpc-lb.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend-grpc rajomon-client; do
    (
      kubectl apply -f "$TMP_DIR/app-grpc-lb.yaml" -l app="${SVC}"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (dagor-lb): one or more workloads failed readiness" >&2
    exit 1
  fi
elif [ "$MODE" = "rajomon-lb" ]; then
  PRICE_UPDATE_RATE=${priceUpdateRate}
  LATENCY_THRESHOLD=${latencyThreshold}
  TOKEN_UPDATE_RATE=${tokenUpdateRate}
  PRICE_STEP=${priceStep}
  TOKEN_UPDATE_STEP=${tokenUpdateStep}
  echo "Using Rajomon config:"
  echo "  priceUpdateRate=$PRICE_UPDATE_RATE latencyThreshold=$LATENCY_THRESHOLD tokenUpdateRate=$TOKEN_UPDATE_RATE priceStep=$PRICE_STEP tokenUpdateStep=$TOKEN_UPDATE_STEP"
  cat k8s/rajomon-lb.env > "$TMP_DIR/rajomon_lb_merged.env"
  echo "" >> "$TMP_DIR/rajomon_lb_merged.env"
  echo "priceUpdateRate=$PRICE_UPDATE_RATE" >> "$TMP_DIR/rajomon_lb_merged.env"
  echo "latencyThreshold=$LATENCY_THRESHOLD" >> "$TMP_DIR/rajomon_lb_merged.env"
  echo "tokenUpdateRate=$TOKEN_UPDATE_RATE" >> "$TMP_DIR/rajomon_lb_merged.env"
  echo "priceStep=$PRICE_STEP" >> "$TMP_DIR/rajomon_lb_merged.env"
  echo "tokenUpdateStep=$TOKEN_UPDATE_STEP" >> "$TMP_DIR/rajomon_lb_merged.env"
  K8S_NS=${K8S_NS:-$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)}
  K8S_NS=${K8S_NS:-default}
  sed -i "s|=${BENCH}-\([^=]*\):2000|=${BENCH}-\1.${K8S_NS}.svc.cluster.local:2000|g" "$TMP_DIR/rajomon_lb_merged.env"
  kubectl create configmap multi-api-config --from-env-file="$TMP_DIR/rajomon_lb_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  cp k8s/manifests/app-grpc-lb.yaml "$TMP_DIR/app-grpc-lb.yaml"
  for IMG in backend1 backend2 frontend-grpc rajomon-client; do
    sed -i "s|${BENCH}-${IMG}:latest|${REGISTRY}/${BENCH}-${IMG}:${TAG}|g" "$TMP_DIR/app-grpc-lb.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend-grpc rajomon-client; do
    (
      kubectl apply -f "$TMP_DIR/app-grpc-lb.yaml" -l app="${SVC}"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (rajomon-lb): one or more workloads failed readiness" >&2
    exit 1
  fi
elif [ "$MODE" = "plain-lb" ]; then
  kubectl create configmap multi-api-config --from-env-file=k8s/plain-lb.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend; do
    (
      sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}-lb.yaml" > "$TMP_DIR/${SVC}-lb.yaml"
      kubectl apply -f "$TMP_DIR/${SVC}-lb.yaml"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (plain-lb): one or more workloads failed readiness" >&2
    exit 1
  fi

  kubectl apply -f k8s/manifests/entry-lb.yaml
else
  kubectl create configmap multi-api-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in backend1 backend2 frontend; do
    (
      sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
      kubectl apply -f "$TMP_DIR/${SVC}.yaml"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (plain): one or more workloads failed readiness" >&2
    exit 1
  fi

  kubectl apply -f k8s/manifests/entry.yaml
fi

rm -rf "$TMP_DIR"
echo "Deploy complete."
