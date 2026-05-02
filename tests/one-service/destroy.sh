#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

fail=0
declare -a pids=()
for kn in "frontend"; do
  (
    kubectl delete pod -l app="$kn" --ignore-not-found --wait=true
    kubectl delete service -l app="$kn" --ignore-not-found
  ) &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

kubectl delete configmap one-service-config --ignore-not-found
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
  kubectl delete service one-service-entry --ignore-not-found
  kubectl delete pod -l app=frontend-grpc --ignore-not-found --wait=true
  kubectl delete service -l app=frontend-grpc --ignore-not-found
fi
echo "Destroy complete."
exit "$fail"
