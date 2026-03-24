#!/bin/bash
set -e
MODE=${1:-${SYSTEM:-plain}}
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH=fan-out-fan-in
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"

if [ "$MODE" = "sidecar" ]; then
  cat k8s/sidecar.env > "$TMP_DIR/sidecar_merged.env"
  echo "" >> "$TMP_DIR/sidecar_merged.env"
  echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"
  kubectl create configmap fan-out-fan-in-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/sidecar-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  for SVC in shared backend1 backend2 frontend; do
    sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "k8s/manifests/${SVC}-sidecar.yaml" | \
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" > "$TMP_DIR/${SVC}-sidecar.yaml"
    kubectl apply -f "$TMP_DIR/${SVC}-sidecar.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl wait --for=condition=Ready pod -l app=ingress --timeout=30s || true
else
  kubectl create configmap fan-out-fan-in-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  for SVC in shared backend1 backend2 frontend; do
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
    kubectl apply -f "$TMP_DIR/${SVC}.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  kubectl apply -f k8s/manifests/entry.yaml
fi

rm -rf "$TMP_DIR"
echo "Deploy complete."
