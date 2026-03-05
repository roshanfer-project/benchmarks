#!/bin/bash
set -e
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH=alibaba-large
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"
kubectl create configmap alibaba-large-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
kubectl apply -f "$TMP_DIR/configmap.yaml"

for SVC in  ms-14758  ms-12657  ms-45067  ms-7103  ms-19439  ms-56113  ms-25806  ms-21298  ms-25781  ms-2687  ms-40087  ms-43032  ms-51783  ms-44246  ms-51787  ms-41667  ms-33572  ms-5720  ms-53792  ms-38190  ms-58796  ms-18750  ms-62039  ms-43754  ms-66921  ms-67465  ms-70124  ms-9105  ms-64512 ; do
  sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
  kubectl apply -f "$TMP_DIR/${SVC}.yaml"
  kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
done

kubectl apply -f k8s/manifests/entry.yaml
rm -rf "$TMP_DIR"
echo "Deploy complete."
