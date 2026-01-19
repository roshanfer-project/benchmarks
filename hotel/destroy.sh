#!/bin/bash

echo "Destroying Hotel Benchmark deployment..."

# Delete Pods
kubectl delete pod frontend search profile geo rate reservation user ingress --ignore-not-found

# Delete Services
kubectl delete service hotel-frontend hotel-search hotel-profile hotel-geo hotel-rate hotel-reservation hotel-user ingress --ignore-not-found

# Delete ConfigMaps
kubectl delete configmap hotel-config sidecar-configs --ignore-not-found

echo "Cleanup complete."
