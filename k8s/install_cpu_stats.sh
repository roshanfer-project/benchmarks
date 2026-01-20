#!/bin/bash
# install_cpu_stats.sh - Deploy cpu-stats-exporter DaemonSet

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "======================================================================"
echo "Deploying cpu-stats-exporter DaemonSet"
echo "======================================================================"

# Deploy DaemonSet
echo "Applying cpu-stats-exporter DaemonSet..."
kubectl apply -f "${SCRIPT_DIR}/cpu-stats-daemonset.yaml"

echo ""
echo "Waiting for cpu-stats-exporter pods to be ready..."
kubectl rollout status daemonset/cpu-stats-exporter -n kube-system --timeout=120s

echo ""
echo "======================================================================"
echo "cpu-stats-exporter deployment complete!"
echo "======================================================================"

# Show pod status
echo ""
echo "cpu-stats-exporter pods:"
kubectl get pods -n kube-system -l app=cpu-stats-exporter -o wide

echo ""
echo "======================================================================"
echo "cpu-stats-exporter is now running on all nodes!"
echo ""
echo "Metrics available at:"
echo "  - Per node: http://<node-ip>:9100/metrics"
echo ""
echo "Configuration:"
echo "  - Update interval: 2 seconds"
echo "  - Data source: /sys/fs/cgroup (direct kernel stats)"
echo "  - Pod metadata: Kubelet API"
echo "======================================================================"

# Test endpoint
echo ""
echo "Testing metrics endpoint..."
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
echo "Fetching from http://${NODE_IP}:9100/metrics"
curl -s "http://${NODE_IP}:9100/metrics" | head -20 || echo "Note: Endpoint may not be immediately accessible"

echo ""
echo "======================================================================"
echo "Installation complete!"
echo "===================================================================="
