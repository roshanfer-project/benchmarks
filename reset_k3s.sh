#!/bin/bash

# Exit on error
set -e

echo "Resetting k3s..."

# We reset by uninstalling and re-installing to ensure a clean slate
# verifying no leftovers.

./delete_k3s.sh
./install_k3s.sh

echo "K3s reset complete!"

