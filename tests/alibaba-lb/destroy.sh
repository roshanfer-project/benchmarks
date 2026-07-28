#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

fail=0
declare -a pids=()
for kn in "ms-25806" "ms-2687" "ms-40087" "ms-44246" "ms-51787" "ms-56113" "ms-64512" "ms-70124"; do
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

kubectl delete configmap alibaba-lb-config --ignore-not-found
kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap prometheus-config --ignore-not-found
if [ "$MODE" = "sidecar" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap sidecar-configs --ignore-not-found
fi
if [ "$MODE" = "approx" ] || [ "$MODE" = "approx-fcfs" ] || [ "$MODE" = "approx-edf" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap approx-configs approx-fcfs-configs approx-edf-configs --ignore-not-found
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
  kubectl delete service alibaba-lb-entry --ignore-not-found
  kubectl delete deployment -l app=ms-64512-grpc --ignore-not-found --wait=true
  kubectl delete pod -l app=ms-64512-grpc --ignore-not-found --wait=true
  kubectl delete service -l app=ms-64512-grpc --ignore-not-found
fi
echo "Destroy complete."
exit "$fail"
