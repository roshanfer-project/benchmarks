#!/bin/bash

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Destroying Social Benchmark..."
echo "Root Dir: $ROOT_DIR"

cd "$ROOT_DIR"

fail=0
declare -a pids=()

for kn in graph posts home user compose nginx nginx-grpc rajomon-client ingress; do
  (
    kubectl delete pod -l app="$kn" --ignore-not-found --wait=true
    kubectl delete service -l app="$kn" --ignore-not-found
  ) &
  pids+=($!)
done

(
  kubectl delete deployment -l app=redis --ignore-not-found --wait=true
  kubectl delete service -l app=redis --ignore-not-found
) &
pids+=($!)

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap sidecar-configs --ignore-not-found
kubectl delete configmap social-config --ignore-not-found
kubectl delete configmap prometheus-config --ignore-not-found

echo "Destroy complete."
exit "$fail"
