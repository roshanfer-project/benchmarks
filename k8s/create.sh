#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CONFIG_FILE="$SCRIPT_DIR/config.env"
export HOSTS_FILE="${HOSTS_FILE:-"$SCRIPT_DIR/hosts.txt"}"

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

if [ ${#HOSTS[@]} -eq 0 ]; then
    log_error "No hosts found in $HOSTS_FILE"
    exit 1
fi

# Helper to parse "user@host" or just "host"
# Sets global variables CURRENT_USER and CURRENT_HOST
parse_host_entry() {
    local entry=$1
    if [[ "$entry" == *"@"* ]]; then
        CURRENT_USER=$(echo "$entry" | cut -d'@' -f1)
        CURRENT_HOST=$(echo "$entry" | cut -d'@' -f2)
    else
        CURRENT_USER="$SSH_USER"
        CURRENT_HOST="$entry"
    fi
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
log_info "  Default SSH User: $SSH_USER"

# Function to run command via SSH
ssh_exec() {
    local user=$1
    local host=$2
    local cmd=$3
    ssh $SSH_OPTS "$user@$host" "$cmd"
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
    log_success "Cluster verified ready. Skipping setup."
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

# Copy kubeconfig to local project directory
log_info "Copying kubeconfig to $SCRIPT_DIR/kubeconfig..."
ssh $SSH_OPTS "$SERVER_User@$SERVER_HOST" "sudo cat /etc/rancher/k3s/k3s.yaml" > "$SCRIPT_DIR/kubeconfig"
# Replace localhost/127.0.0.1 with ssh hostname
sed -i "s/127.0.0.1/$SERVER_HOST/g" "$SCRIPT_DIR/kubeconfig"
chmod 600 "$SCRIPT_DIR/kubeconfig"

# Install to default location for global usage
mkdir -p ~/.kube
if [ -f ~/.kube/config ]; then
    log_info "Backing up existing kubeconfig to ~/.kube/config.bak"
    mv ~/.kube/config ~/.kube/config.bak
fi
cp "$SCRIPT_DIR/kubeconfig" ~/.kube/config
chmod 600 ~/.kube/config

export KUBECONFIG="$SCRIPT_DIR/kubeconfig"
log_success "Control plane initialized. Kubeconfig installed to ~/.kube/config and $SCRIPT_DIR/kubeconfig"

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
echo "Kubeconfig is installed in ~/.kube/config."
echo "You can run 'kubectl get nodes' immediately."
kubectl get nodes



