#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"
kubectl delete pod -l app=backend1 --ignore-not-found --wait=true
kubectl delete service -l app=backend1 --ignore-not-found
kubectl delete pod -l app=backend2 --ignore-not-found --wait=true
kubectl delete service -l app=backend2 --ignore-not-found
kubectl delete pod -l app=backend3 --ignore-not-found --wait=true
kubectl delete service -l app=backend3 --ignore-not-found
kubectl delete pod -l app=backend4 --ignore-not-found --wait=true
kubectl delete service -l app=backend4 --ignore-not-found
kubectl delete pod -l app=backend5 --ignore-not-found --wait=true
kubectl delete service -l app=backend5 --ignore-not-found
kubectl delete pod -l app=backend6 --ignore-not-found --wait=true
kubectl delete service -l app=backend6 --ignore-not-found
kubectl delete pod -l app=frontend --ignore-not-found --wait=true
kubectl delete service -l app=frontend --ignore-not-found
kubectl delete configmap dynamic-large-config --ignore-not-found
kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap prometheus-config --ignore-not-found
if [ "$MODE" = "sidecar" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap sidecar-configs --ignore-not-found
fi
if [ "$MODE" = "rajomon" ] || [ "$MODE" = "dagor" ]; then
  kubectl delete pod -l app=rajomon-client --ignore-not-found --wait=true
  kubectl delete service -l app=rajomon-client --ignore-not-found
  kubectl delete service dynamic-large-entry --ignore-not-found
  kubectl delete pod -l app=frontend-grpc --ignore-not-found --wait=true
  kubectl delete service -l app=frontend-grpc --ignore-not-found
fi
echo "Destroy complete."
