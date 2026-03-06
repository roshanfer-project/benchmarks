#!/bin/bash
set -e
MODE=${1:-${SYSTEM:-plain}}
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH=alibaba-large
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"

if [ "$MODE" = "sidecar" ]; then
  cat k8s/sidecar.env > "$TMP_DIR/sidecar_merged.env"
  echo "" >> "$TMP_DIR/sidecar_merged.env"
  echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"
  kubectl create configmap alibaba-large-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/sidecar-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  cp k8s/manifests/app-sidecar.yaml "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-14758:latest|${REGISTRY}/${BENCH}-ms-14758:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-12657:latest|${REGISTRY}/${BENCH}-ms-12657:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-45067:latest|${REGISTRY}/${BENCH}-ms-45067:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-7103:latest|${REGISTRY}/${BENCH}-ms-7103:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-19439:latest|${REGISTRY}/${BENCH}-ms-19439:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-56113:latest|${REGISTRY}/${BENCH}-ms-56113:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-25806:latest|${REGISTRY}/${BENCH}-ms-25806:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-21298:latest|${REGISTRY}/${BENCH}-ms-21298:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-25781:latest|${REGISTRY}/${BENCH}-ms-25781:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-2687:latest|${REGISTRY}/${BENCH}-ms-2687:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-40087:latest|${REGISTRY}/${BENCH}-ms-40087:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-43032:latest|${REGISTRY}/${BENCH}-ms-43032:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-51783:latest|${REGISTRY}/${BENCH}-ms-51783:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-44246:latest|${REGISTRY}/${BENCH}-ms-44246:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-51787:latest|${REGISTRY}/${BENCH}-ms-51787:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-41667:latest|${REGISTRY}/${BENCH}-ms-41667:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-33572:latest|${REGISTRY}/${BENCH}-ms-33572:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-5720:latest|${REGISTRY}/${BENCH}-ms-5720:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-53792:latest|${REGISTRY}/${BENCH}-ms-53792:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-38190:latest|${REGISTRY}/${BENCH}-ms-38190:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-58796:latest|${REGISTRY}/${BENCH}-ms-58796:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-18750:latest|${REGISTRY}/${BENCH}-ms-18750:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-62039:latest|${REGISTRY}/${BENCH}-ms-62039:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-43754:latest|${REGISTRY}/${BENCH}-ms-43754:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-66921:latest|${REGISTRY}/${BENCH}-ms-66921:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-67465:latest|${REGISTRY}/${BENCH}-ms-67465:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-70124:latest|${REGISTRY}/${BENCH}-ms-70124:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-9105:latest|${REGISTRY}/${BENCH}-ms-9105:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|${BENCH}-ms-64512:latest|${REGISTRY}/${BENCH}-ms-64512:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  
  kubectl apply -f "$TMP_DIR/app-sidecar.yaml"
  kubectl wait --for=condition=Ready pod -l app=ms-14758 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-12657 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-45067 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-7103 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-19439 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-56113 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-25806 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-21298 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-25781 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-2687 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-40087 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-43032 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-51783 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-44246 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-51787 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-41667 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-33572 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-5720 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-53792 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-38190 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-58796 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-18750 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-62039 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-43754 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-66921 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-67465 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-70124 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-9105 --timeout=${WAIT_TIMEOUT}s
  kubectl wait --for=condition=Ready pod -l app=ms-64512 --timeout=${WAIT_TIMEOUT}s
  

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl wait --for=condition=Ready pod -l app=ingress --timeout=30s || true
else
  kubectl create configmap alibaba-large-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  for SVC in  ms-14758  ms-12657  ms-45067  ms-7103  ms-19439  ms-56113  ms-25806  ms-21298  ms-25781  ms-2687  ms-40087  ms-43032  ms-51783  ms-44246  ms-51787  ms-41667  ms-33572  ms-5720  ms-53792  ms-38190  ms-58796  ms-18750  ms-62039  ms-43754  ms-66921  ms-67465  ms-70124  ms-9105  ms-64512 ; do
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
    kubectl apply -f "$TMP_DIR/${SVC}.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  kubectl apply -f k8s/manifests/entry.yaml
fi

rm -rf "$TMP_DIR"
echo "Deploy complete."
