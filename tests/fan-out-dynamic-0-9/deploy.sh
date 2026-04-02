#!/bin/bash
set -e
MODE=${1:-${SYSTEM:-plain}}
ARG2=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH=fan-out-dynamic-0-9
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

if [ "$MODE" = "plain" ] && [ "$ARG2" = "debug" ]; then
  echo "deploy.sh: debug only with sidecar; use ./deploy.sh sidecar debug" >&2
  exit 1
fi
if [ "$MODE" = "sidecar" ] && [ -n "$ARG2" ] && [ "$ARG2" != "debug" ]; then
  echo "deploy.sh: unknown second argument: $ARG2 (expected: debug)" >&2
  exit 1
fi
SIDECAR_DEBUG=0
if [ "$MODE" = "sidecar" ] && [ "$ARG2" = "debug" ]; then
  SIDECAR_DEBUG=1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"

sidecar_debug_require_yq() {
  command -v yq >/dev/null 2>&1 || {
    echo "deploy.sh sidecar debug needs mikefarah yq v4: https://github.com/mikefarah/yq" >&2
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
  if .name == "sidecar" then
    (.env // []) as $e |
    ($e | map(select(.name != "GLOG_v" and .name != "GLOG_vmodule"))) as $base |
    .env = $base
      + (if (strenv(GV_VAL) | length) > 0 then [{"name":"GLOG_v","value":strenv(GV_VAL)}] else [] end)
      + (if (strenv(VM_VAL) | length) > 0 then [{"name":"GLOG_vmodule","value":strenv(VM_VAL)}] else [] end)
  else .
  end
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
  kubectl create configmap fan-out-dynamic-0-9-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/sidecar-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  for SVC in backend1 backend2 frontend; do
    sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "k8s/manifests/${SVC}-sidecar.yaml" | \
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" > "$TMP_DIR/${SVC}-sidecar.yaml"
    if [ "$SIDECAR_DEBUG" = "1" ]; then
      sidecar_debug_patch_workload_yaml "$TMP_DIR/${SVC}-sidecar.yaml"
    fi
    kubectl apply -f "$TMP_DIR/${SVC}-sidecar.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_patch_workload_yaml "$TMP_DIR/ingress.yaml"
  fi
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl wait --for=condition=Ready pod -l app=ingress --timeout=30s || true
else
  kubectl create configmap fan-out-dynamic-0-9-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  for SVC in backend1 backend2 frontend; do
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
    kubectl apply -f "$TMP_DIR/${SVC}.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  kubectl apply -f k8s/manifests/entry.yaml
fi

rm -rf "$TMP_DIR"
echo "Deploy complete."
