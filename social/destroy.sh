#!/bin/bash

# Usage: ./destroy.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Destroying Social Benchmark..."
echo "Root Dir: $ROOT_DIR"

cd "$ROOT_DIR"
DEPLOY_DIR="social/k8s"

# Delete Ingress
if [ -f "${DEPLOY_DIR}/ingress.yaml" ]; then
    kubectl delete -f "${DEPLOY_DIR}/ingress.yaml" --ignore-not-found
fi

# Delete Apps (Plain & Sidecar)
# We generated tmp apps in deploy script, but we can delete by label or by original manifest if the names match.
# Deleting by label is safer/cleaner.
echo "Deleting pods by label..."
kubectl delete pod -l app=graph --ignore-not-found --wait=false
kubectl delete pod -l app=posts --ignore-not-found --wait=false
kubectl delete pod -l app=home --ignore-not-found --wait=false
kubectl delete pod -l app=user --ignore-not-found --wait=false
kubectl delete pod -l app=compose --ignore-not-found --wait=false
kubectl delete pod -l app=nginx --ignore-not-found --wait=false
kubectl delete pod -l app=ingress --ignore-not-found --wait=false
kubectl delete deployment -l app=redis --ignore-not-found --wait=false

kubectl delete service -l app=graph --ignore-not-found
kubectl delete service -l app=posts --ignore-not-found
kubectl delete service -l app=home --ignore-not-found
kubectl delete service -l app=user --ignore-not-found
kubectl delete service -l app=compose --ignore-not-found
kubectl delete service -l app=nginx --ignore-not-found
kubectl delete service -l app=ingress --ignore-not-found
kubectl delete service -l app=redis --ignore-not-found

# Delete ConfigMaps
kubectl delete configmap sidecar-configs --ignore-not-found
kubectl delete configmap social-config --ignore-not-found

echo "Waiting for pods to terminate..."
kubectl wait --for=delete pod -l app=graph --timeout=60s || true
kubectl wait --for=delete pod -l app=posts --timeout=60s || true
kubectl wait --for=delete pod -l app=home --timeout=60s || true
kubectl wait --for=delete pod -l app=user --timeout=60s || true
kubectl wait --for=delete pod -l app=compose --timeout=60s || true
kubectl wait --for=delete pod -l app=nginx --timeout=60s || true
kubectl wait --for=delete pod -l app=ingress --timeout=60s || true
kubectl wait --for=delete pod -l app=redis --timeout=60s || true

echo "Destroy complete."
