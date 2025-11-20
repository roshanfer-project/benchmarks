#!/bin/bash

# Exit on error
set -e

echo "Deleting k3s..."

# Check if k3s-uninstall.sh exists
if [ -f /usr/local/bin/k3s-uninstall.sh ]; then
    echo "Running k3s-uninstall.sh..."
    /usr/local/bin/k3s-uninstall.sh
else
    echo "k3s-uninstall.sh not found. Is k3s installed?"
fi

# Clean up aliases
echo "Cleaning up aliases..."
if [ -f ~/.bashrc ]; then
    sed -i "/alias kubectl='sudo k3s kubectl'/d" ~/.bashrc
fi
if [ -f ~/.zshrc ]; then
    sed -i "/alias kubectl='sudo k3s kubectl'/d" ~/.zshrc
fi

# Clean up kube config
echo "Cleaning up kube config..."
if [ -f ~/.kube/config ]; then
    # We blindly remove/modify it? 
    # Better to check if it looks like k3s config (server: https://127.0.0.1:6443)
    if grep -q "https://127.0.0.1:6443" ~/.kube/config; then
        rm -f ~/.kube/config
    fi
fi

echo "K3s deletion complete!"

