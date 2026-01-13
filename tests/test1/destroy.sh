#!/bin/bash

echo "Destroying test1 deployment..."

# Delete Deployments
kubectl delete deployment app ingress --ignore-not-found

# Delete Services
kubectl delete service app ingress --ignore-not-found

# Delete ConfigMaps
kubectl delete configmap test1-config sidecar-configs --ignore-not-found

echo "Cleanup complete."
