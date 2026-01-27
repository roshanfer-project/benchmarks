#!/bin/bash

echo "Destroying Hotel Benchmark deployment..."

# Delete Pods
kubectl delete pod frontend search profile geo rate reservation user ingress \
    frontend-grpc rajomon-client \
    mongodb-geo mongodb-profile mongodb-rate mongodb-reservation mongodb-user \
    memcached-rate memcached-profile memcached-reserve --ignore-not-found

kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found

# Delete Services
kubectl delete service hotel-frontend hotel-search hotel-profile hotel-geo hotel-rate hotel-reservation hotel-user ingress \
    frontend-grpc \
    mongodb-geo mongodb-profile mongodb-rate mongodb-reservation mongodb-user \
    memcached-rate memcached-profile memcached-reserve prometheus prometheus-pushgateway prometheus-external --ignore-not-found

# Delete ConfigMaps
kubectl delete configmap hotel-config sidecar-configs prometheus-config --ignore-not-found

echo "Cleanup complete."
