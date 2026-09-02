#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
REPO_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"
# shellcheck source=/dev/null
source "$REPO_ROOT/scripts/elapsed.sh"
CONFIG_FILE="$SCRIPT_DIR/config.env"
export HOSTS_FILE="${HOSTS_FILE:-"$REPO_ROOT/hosts.txt"}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check requirements
if [ ! -f "$CONFIG_FILE" ]; then
    log_error "Config file not found: $CONFIG_FILE"
    exit 1
fi
source "$CONFIG_FILE"

if [ ! -f "$HOSTS_FILE" ]; then
    log_error "Hosts file not found: $HOSTS_FILE"
    exit 1
fi

# Filter out comments and empty lines
mapfile -t HOSTS < <(grep -vE '^\s*#|^\s*$' "$HOSTS_FILE")

ng="${NUM_GENERATORS:-0}"
if [[ "$ng" =~ ^[0-9]+$ ]] && [ "$ng" -gt 0 ]; then
    HOSTS=("${HOSTS[@]:ng}")
fi

if [ ${#HOSTS[@]} -eq 0 ]; then
    log_error "No hosts found in $HOSTS_FILE (after skipping NUM_GENERATORS=$ng)"
    exit 1
fi

# Sets CURRENT_USER and CURRENT_HOST. Lines must be user@host.
parse_host_entry() {
    local entry=$1
    if [[ "$entry" != *"@"* ]]; then
        log_error "Host line must be user@host: $entry"
        exit 1
    fi
    CURRENT_USER="${entry%%@*}"
    CURRENT_HOST="${entry#*@}"
}

SERVER_ENTRY="${HOSTS[0]}"
parse_host_entry "$SERVER_ENTRY"
SERVER_User="$CURRENT_USER"
SERVER_HOST="$CURRENT_HOST"

AGENT_ENTRIES=("${HOSTS[@]:1}")

log_info "Cluster Configuration:"
log_info "  K3s Version: $K3S_VERSION"
log_info "  Server Node: $SERVER_HOST (User: $SERVER_User)"
log_info "  Agent Nodes: ${AGENT_ENTRIES[*]}"

# Function to run command via SSH
ssh_exec() {
    local user=$1
    local host=$2
    local cmd=$3
    ssh $SSH_OPTS "$user@$host" "$cmd"
}

# Pull admin kubeconfig from control plane and install locally (always use current SERVER_HOST).
install_local_kubeconfig_from_server() {
    log_info "Copying kubeconfig to $SCRIPT_DIR/kubeconfig..."
    ssh $SSH_OPTS "$SERVER_User@$SERVER_HOST" "sudo cat /etc/rancher/k3s/k3s.yaml" > "$SCRIPT_DIR/kubeconfig"
    sed -i "s/127.0.0.1/$SERVER_HOST/g" "$SCRIPT_DIR/kubeconfig"
    sed -i "s|https://localhost:6443|https://${SERVER_HOST}:6443|g" "$SCRIPT_DIR/kubeconfig" || true
    chmod 600 "$SCRIPT_DIR/kubeconfig"
    log_success "Kubeconfig saved to $SCRIPT_DIR/kubeconfig (use direnv to set KUBECONFIG automatically)"
}

# --- CHECK EXISTING CLUSTER ---
check_cluster_ready() {
    export KUBECONFIG="$SCRIPT_DIR/kubeconfig"
    if [ ! -f "$KUBECONFIG" ]; then return 1; fi
    
    if ! kubectl get nodes &> /dev/null; then return 1; fi
    
    CURRENT_NODES=$(kubectl get nodes --no-headers | wc -l)
    EXPECTED_NODES=${#HOSTS[@]}
    
    if [ "$CURRENT_NODES" -ge "$EXPECTED_NODES" ]; then
        # Optional: Check if nodes are truly Ready
        READY_NODES=$(kubectl get nodes --no-headers | grep "Ready" | wc -l)
        if [ "$READY_NODES" -ge "$EXPECTED_NODES" ]; then
            return 0
        fi
    fi
    return 1
}

if check_cluster_ready; then
    log_success "Cluster verified ready. Skipping K3s reinstall."
    install_local_kubeconfig_from_server
    export KUBECONFIG="$SCRIPT_DIR/kubeconfig"
    log_success "Local kubeconfig refreshed for $SERVER_HOST ($SCRIPT_DIR/kubeconfig)."
    exit 0
else
    log_info "Cluster not ready or missing. Cleaning up before installation..."
    # Run delete.sh to ensure clean slate
    if [ -f "$SCRIPT_DIR/delete.sh" ]; then
        "$SCRIPT_DIR/delete.sh"
    fi
fi

# --- COMMON SETUP (All Nodes) ---
# Note: Common provisioning (SSH keys, Go, Git Clone) is now handled by ../provisioning/provision.sh
# We assume the nodes are already provisioned.


# --- SERVER INSTALLATION ---
log_info "Installing K3s Server on $SERVER_HOST..."

# K3s installation flags
# Flannel with host-gw backend for high performance (requires direct L2 connectivity)
# Traefik disabled as requested (only NodePort/L4 needed)
# NodePort range extended to include 3000-3009
SERVER_FLAGS="--flannel-backend=host-gw --disable-network-policy --disable=traefik --cluster-cidr=$CLUSTER_CIDR --service-cidr=$SERVICE_CIDR --tls-san=$SERVER_HOST --kube-apiserver-arg=service-node-port-range=3000-32767"
KUBELET_FLAGS="--kubelet-arg=cpu-manager-policy='$CPU_MANAGER_POLICY' --kubelet-arg=kube-reserved='$KUBE_RESERVED' --kubelet-arg=system-reserved='$SYSTEM_RESERVED' --kubelet-arg=eviction-hard='$EVICTION_HARD'"

INSTALL_CMD="curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=$K3S_VERSION sh -s - server $SERVER_FLAGS $KUBELET_FLAGS"

ssh_exec "$SERVER_User" "$SERVER_HOST" "$INSTALL_CMD"

# Configure remote user access to kubectl
log_info "Configuring remote access for user $SERVER_User on $SERVER_HOST..."
SETUP_KUBECONFIG_CMD="mkdir -p ~/.kube && sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config && sudo chown \$(id -u):\$(id -g) ~/.kube/config && chmod 600 ~/.kube/config && echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc"
ssh_exec "$SERVER_User" "$SERVER_HOST" "$SETUP_KUBECONFIG_CMD"

install_local_kubeconfig_from_server
export KUBECONFIG="$SCRIPT_DIR/kubeconfig"
log_success "Control plane initialized. Kubeconfig at $SCRIPT_DIR/kubeconfig"

# Check for kubectl and install if missing
if ! command -v kubectl &> /dev/null; then
    log_info "kubectl not found locally. Installing..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
fi

# Wait for node to be ready (k3s starts quickly but we wait for API)
log_info "Waiting for API server..."
if ! timeout 120 bash -c 'until kubectl get nodes &> /dev/null; do sleep 2; done'; then
    log_error "Timed out waiting for API server. Check network connectivity to $SERVER_HOST:6443."
    exit 1
fi


# Get Node Token
NODE_TOKEN=$(ssh_exec "$SERVER_User" "$SERVER_HOST" "sudo cat /var/lib/rancher/k3s/server/node-token")
log_info "Node Token: ${NODE_TOKEN:0:20}..."

# --- AGENT INSTALLATION ---
for entry in "${AGENT_ENTRIES[@]}"; do
    parse_host_entry "$entry"
    agent_user="$CURRENT_USER"
    agent_host="$CURRENT_HOST"
    
    log_info "Installing K3s Agent on $agent_host (User: $agent_user)..."

    AGENT_CMD="curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=$K3S_VERSION K3S_URL=https://$SERVER_HOST:6443 K3S_TOKEN=$NODE_TOKEN sh -s - agent $KUBELET_FLAGS"
    
    # Run sequentially and check for failure
    if ! ssh_exec "$agent_user" "$agent_host" "$AGENT_CMD"; then
        log_error "Failed to install K3s Agent on $agent_host"
        exit 1
    fi
done
log_success "All agents installed."

log_success "Cluster setup complete!"
echo ""
echo "Kubeconfig at $SCRIPT_DIR/kubeconfig (direnv sets KUBECONFIG automatically)."
kubectl get nodes

log_info "Waiting 30s for the API server to schedule system pods..."
sleep 30

log_info "Waiting for pods to appear (up to 120s, poll every 3s)..."
pods_seen=0
for _ in {1..40}; do
    n=$(kubectl get pods -A --no-headers 2>/dev/null | wc -l)
    n="${n// /}"
    if [ "${n:-0}" -gt 0 ]; then
        pods_seen=1
        break
    fi
    sleep 3
done

if [ "$pods_seen" -eq 0 ]; then
    log_error "No pods found after waiting. Current state:"
    kubectl get pods -A 2>/dev/null || true
    exit 1
fi

log_info "Waiting for all pods to be Ready (timeout 180s)..."
if ! kubectl wait --for=condition=Ready pod --all --all-namespaces --timeout=180s; then
    log_error "Timeout: Not all pods are ready."
    echo "Current Pod Status:"
    kubectl get pods --all-namespaces
    exit 1
fi
log_success "All initial pods are ready."

# --- CPU-STATS-EXPORTER ---
log_info "Building and installing cpu-stats-exporter..."
REGISTRY="${REGISTRY:-farzad1132}"
CPU_STATS_IMAGE="${REGISTRY}/cpu-stats-exporter:latest"
if command -v docker &> /dev/null; then
    (cd "$SCRIPT_DIR/cpu-stats-exporter" && docker build -t "$CPU_STATS_IMAGE" .)
    docker push "$CPU_STATS_IMAGE"
else
    log_info "Docker not found locally; skipping cpu-stats-exporter build. Ensure ${CPU_STATS_IMAGE} exists in registry."
fi
sed "s|image: farzad1132/cpu-stats-exporter:latest|image: ${CPU_STATS_IMAGE}|g" "$SCRIPT_DIR/cpu-stats-daemonset.yaml" | kubectl apply -f -
kubectl rollout status daemonset/cpu-stats-exporter -n kube-system --timeout=120s
log_success "cpu-stats-exporter deployed."


