#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

fail=0
declare -a pids=()
for kn in "backend1" "backend2" "frontend"; do
  (
    kubectl delete deployment -l app="$kn" --ignore-not-found --wait=true
    kubectl delete pod -l app="$kn" --ignore-not-found --wait=true
    kubectl delete service -l app="$kn" --ignore-not-found
  ) &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

kubectl delete configmap fan-out-dynamic-0-9-config --ignore-not-found
kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap prometheus-config --ignore-not-found
if [ "$MODE" = "roshanfer" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap sidecar-configs --ignore-not-found
fi
if [ "$MODE" = "amphiqueue" ] || [ "$MODE" = "amphiqueue-fcfs" ] || [ "$MODE" = "amphiqueue-edf" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap amphiqueue-configs amphiqueue-fcfs-configs amphiqueue-edf-configs --ignore-not-found
fi
if [ "$MODE" = "envoy" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap envoy-configs --ignore-not-found
fi
if [ "$MODE" = "p2c" ] || [ "$MODE" = "wrr" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap p2c-envoy-configs --ignore-not-found
fi
if [ "$MODE" = "rajomon" ] || [ "$MODE" = "dagor" ] || [ "$MODE" = "dagor-lb" ] || [ "$MODE" = "rajomon-lb" ]; then
  kubectl delete deployment -l app=rajomon-client --ignore-not-found --wait=true
  kubectl delete pod -l app=rajomon-client --ignore-not-found --wait=true
  kubectl delete service -l app=rajomon-client --ignore-not-found
  kubectl delete service fan-out-dynamic-0-9-entry --ignore-not-found
  kubectl delete deployment -l app=frontend-grpc --ignore-not-found --wait=true
  kubectl delete pod -l app=frontend-grpc --ignore-not-found --wait=true
  kubectl delete service -l app=frontend-grpc --ignore-not-found
fi
echo "Destroy complete."
exit "$fail"
