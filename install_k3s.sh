#!/bin/bash

# Exit on error
set -e

echo "Installing k3s..."

# Install k3s with specific configuration
# - Disable traefik (we might want to use our own ingress or NodePort)
# - Configure kubelet args for static CPU manager (similar to microk8s setup)
# - Configure service-node-port-range to allow ports starting from 3000 (default is 30000-32767)
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --disable traefik --kubelet-arg=cpu-manager-policy=static --kubelet-arg=kube-reserved=cpu=500m,memory=500Mi --kubelet-arg=system-reserved=cpu=500m,memory=500Mi --kube-apiserver-arg service-node-port-range=3000-32767" sh -

echo "Waiting for k3s to be ready..."
# Wait for the node to be ready
until sudo k3s kubectl get node > /dev/null 2>&1; do 
    echo "Waiting for k3s API..."
    sleep 2
done
sudo k3s kubectl wait --for=condition=Ready node --all --timeout=60s

echo "Configuring permissions for the current user..."
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
chmod 600 ~/.kube/config

# Check if kubectl is installed, if not alias it or rely on k3s kubectl
if ! command -v kubectl &> /dev/null; then
    echo "kubectl not found. Creating alias..."
    # Add alias to bashrc/zshrc if not present
    if [ -f ~/.bashrc ]; then
        if ! grep -q "alias kubectl='sudo k3s kubectl'" ~/.bashrc; then
            echo "alias kubectl='sudo k3s kubectl'" >> ~/.bashrc
        fi
    fi
    if [ -f ~/.zshrc ]; then
        if ! grep -q "alias kubectl='sudo k3s kubectl'" ~/.zshrc; then
            echo "alias kubectl='sudo k3s kubectl'" >> ~/.zshrc
        fi
    fi
    echo "Alias created. You might need to source your shell rc file."
else
    echo "kubectl is installed. Configuration updated at ~/.kube/config"
fi

echo "Deploying local registry (NodePort 32000)..."
# Deploy a simple registry
cat <<EOF | sudo k3s kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
      - name: registry
        image: registry:2
        ports:
        - containerPort: 5000
---
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: kube-system
spec:
  selector:
    app: registry
  ports:
  - port: 5000
    targetPort: 5000
    nodePort: 32000
  type: NodePort
EOF

echo "Waiting for registry to be ready..."
sudo k3s kubectl wait --for=condition=available deployment/registry -n kube-system --timeout=60s

echo "K3s installation and configuration complete!"
