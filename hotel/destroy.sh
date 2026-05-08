#!/bin/bash

echo "Destroying Hotel Benchmark deployment..."

fail=0
declare -a pids=()

for kn in \
  frontend search profile geo rate reservation user ingress frontend-grpc rajomon-client \
  mongodb-geo mongodb-profile mongodb-rate mongodb-reservation mongodb-user \
  memcached-profile memcached-rate memcached-reserve
do
  (
    kubectl delete pod -l app="$kn" --ignore-not-found --wait=true
    kubectl delete service -l app="$kn" --ignore-not-found
  ) &
  pids+=($!)
done

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap hotel-config sidecar-configs prometheus-config --ignore-not-found

echo "Cleanup complete."
exit "$fail"
