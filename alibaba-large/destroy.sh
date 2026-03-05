#!/bin/bash
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"
kubectl delete pod -l app=ms-12657 --ignore-not-found --wait=false
kubectl delete service -l app=ms-12657 --ignore-not-found
kubectl delete pod -l app=ms-14758 --ignore-not-found --wait=false
kubectl delete service -l app=ms-14758 --ignore-not-found
kubectl delete pod -l app=ms-18750 --ignore-not-found --wait=false
kubectl delete service -l app=ms-18750 --ignore-not-found
kubectl delete pod -l app=ms-19439 --ignore-not-found --wait=false
kubectl delete service -l app=ms-19439 --ignore-not-found
kubectl delete pod -l app=ms-21298 --ignore-not-found --wait=false
kubectl delete service -l app=ms-21298 --ignore-not-found
kubectl delete pod -l app=ms-25781 --ignore-not-found --wait=false
kubectl delete service -l app=ms-25781 --ignore-not-found
kubectl delete pod -l app=ms-25806 --ignore-not-found --wait=false
kubectl delete service -l app=ms-25806 --ignore-not-found
kubectl delete pod -l app=ms-2687 --ignore-not-found --wait=false
kubectl delete service -l app=ms-2687 --ignore-not-found
kubectl delete pod -l app=ms-33572 --ignore-not-found --wait=false
kubectl delete service -l app=ms-33572 --ignore-not-found
kubectl delete pod -l app=ms-38190 --ignore-not-found --wait=false
kubectl delete service -l app=ms-38190 --ignore-not-found
kubectl delete pod -l app=ms-40087 --ignore-not-found --wait=false
kubectl delete service -l app=ms-40087 --ignore-not-found
kubectl delete pod -l app=ms-41667 --ignore-not-found --wait=false
kubectl delete service -l app=ms-41667 --ignore-not-found
kubectl delete pod -l app=ms-43032 --ignore-not-found --wait=false
kubectl delete service -l app=ms-43032 --ignore-not-found
kubectl delete pod -l app=ms-43754 --ignore-not-found --wait=false
kubectl delete service -l app=ms-43754 --ignore-not-found
kubectl delete pod -l app=ms-44246 --ignore-not-found --wait=false
kubectl delete service -l app=ms-44246 --ignore-not-found
kubectl delete pod -l app=ms-45067 --ignore-not-found --wait=false
kubectl delete service -l app=ms-45067 --ignore-not-found
kubectl delete pod -l app=ms-51783 --ignore-not-found --wait=false
kubectl delete service -l app=ms-51783 --ignore-not-found
kubectl delete pod -l app=ms-51787 --ignore-not-found --wait=false
kubectl delete service -l app=ms-51787 --ignore-not-found
kubectl delete pod -l app=ms-53792 --ignore-not-found --wait=false
kubectl delete service -l app=ms-53792 --ignore-not-found
kubectl delete pod -l app=ms-56113 --ignore-not-found --wait=false
kubectl delete service -l app=ms-56113 --ignore-not-found
kubectl delete pod -l app=ms-5720 --ignore-not-found --wait=false
kubectl delete service -l app=ms-5720 --ignore-not-found
kubectl delete pod -l app=ms-58796 --ignore-not-found --wait=false
kubectl delete service -l app=ms-58796 --ignore-not-found
kubectl delete pod -l app=ms-62039 --ignore-not-found --wait=false
kubectl delete service -l app=ms-62039 --ignore-not-found
kubectl delete pod -l app=ms-64512 --ignore-not-found --wait=false
kubectl delete service -l app=ms-64512 --ignore-not-found
kubectl delete pod -l app=ms-66921 --ignore-not-found --wait=false
kubectl delete service -l app=ms-66921 --ignore-not-found
kubectl delete pod -l app=ms-67465 --ignore-not-found --wait=false
kubectl delete service -l app=ms-67465 --ignore-not-found
kubectl delete pod -l app=ms-70124 --ignore-not-found --wait=false
kubectl delete service -l app=ms-70124 --ignore-not-found
kubectl delete pod -l app=ms-7103 --ignore-not-found --wait=false
kubectl delete service -l app=ms-7103 --ignore-not-found
kubectl delete pod -l app=ms-9105 --ignore-not-found --wait=false
kubectl delete service -l app=ms-9105 --ignore-not-found
kubectl delete configmap alibaba-large-config --ignore-not-found
echo "Destroy complete."
