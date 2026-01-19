#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
HOSTS_FILE="${HOSTS_FILE:-"$SCRIPT_DIR/hosts.txt"}"
CONFIG_FILE="$SCRIPT_DIR/../k8s/config.env" # Re-use k8s config if available, or just for SSH_OPTS

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi

# Ensure SSH_OPTS is defined if not sourced
SSH_OPTS=${SSH_OPTS:-"-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"}
SSH_USER=${SSH_USER:-"ubuntu"} # Default user if not specified

if [ ! -f "$HOSTS_FILE" ]; then
    log_error "Hosts file not found: $HOSTS_FILE"
    exit 1
fi

FORCE_PROVISION="false"
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -f|--force) FORCE_PROVISION="true" ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

mapfile -t HOSTS < <(grep -vE '^\s*#|^\s*$' "$HOSTS_FILE")

# Helper to parse "user@host"
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

ssh_exec() {
    local user=$1
    local host=$2
    local cmd=$3
    ssh $SSH_OPTS "$user@$host" "$cmd"
}

log_info "Starting provisioning for ${#HOSTS[@]} hosts..."

for entry in "${HOSTS[@]}"; do
    parse_host_entry "$entry"
    node_user="$CURRENT_USER"
    node_host="$CURRENT_HOST"
    
    log_info "Provisioning $node_host ($node_user)..."

    # Check if already provisioned
    if [ "$FORCE_PROVISION" != "true" ] && ssh_exec "$node_user" "$node_host" "[ -f .roshanfer_provisioned ]"; then
         log_success "  Host $node_host already provisioned (found .roshanfer_provisioned). Skipping (-f to force)."
         continue
    fi

    # 1. SSH Keys Setup
    log_info "  [1/4] Setting up SSH keys..."
    ssh_exec "$node_user" "$node_host" "mkdir -p ~/.ssh && chmod 700 ~/.ssh"
    # Copy local keys to remote (for git clone etc)
    if [ -f ~/.ssh/id_ed25519 ]; then
        scp $SSH_OPTS ~/.ssh/id_ed25519 "$node_user@$node_host:~/.ssh/"
        scp $SSH_OPTS ~/.ssh/id_ed25519.pub "$node_user@$node_host:~/.ssh/"
        ssh_exec "$node_user" "$node_host" "chmod 600 ~/.ssh/id_ed25519 && chmod 644 ~/.ssh/id_ed25519.pub"
    elif [ -f ~/.ssh/id_rsa ]; then
        scp $SSH_OPTS ~/.ssh/id_rsa "$node_user@$node_host:~/.ssh/"
        scp $SSH_OPTS ~/.ssh/id_rsa.pub "$node_user@$node_host:~/.ssh/"
        ssh_exec "$node_user" "$node_host" "chmod 600 ~/.ssh/id_rsa && chmod 644 ~/.ssh/id_rsa.pub"
    else
        log_info "  No default SSH keys found to copy (id_ed25519 or id_rsa). Skipping key copy."
    fi

    # 2. Install Go
    log_info "  [2/4] Installing Go..."
    cat "$SCRIPT_DIR/install_go.sh" | ssh_exec "$node_user" "$node_host" "bash -s"

    # 3. Clone Experiment Repo
    log_info "  [3/4] Cloning experiment repository..."
    ssh_exec "$node_user" "$node_host" "ssh-keyscan github.com >> ~/.ssh/known_hosts 2>/dev/null"
    # Using the repo URL from k8s/create.sh logic
    REPO_URL="git@github.com:farzad1132/roshanfer-experments.git"
    DIR_NAME="roshanfer-experments"
    
    CLONE_CMD="if [ ! -d '$DIR_NAME' ]; then 
                   git clone $REPO_URL $DIR_NAME; 
               else 
                   echo 'Repo exists, pulling latest...';
                   cd $DIR_NAME && git pull;
               fi"
    
    ssh_exec "$node_user" "$node_host" "$CLONE_CMD"
    
    # Initialize submodules
    log_info "        Initializing submodules (rwg only)..."
    ssh_exec "$node_user" "$node_host" "cd $DIR_NAME && git submodule update --init --recursive rwg"

    # Build rwg
    log_info "        Building rwg..."
    ssh_exec "$node_user" "$node_host" "cd $DIR_NAME/rwg && /usr/local/go/bin/go build ."


    # 4. High Performance Setup
    log_info "  [4/4] Configuring high performance settings..."
    cat "$SCRIPT_DIR/high_perf.sh" | ssh_exec "$node_user" "$node_host" "sudo bash -s"

    # Mark as provisioned
    ssh_exec "$node_user" "$node_host" "touch .roshanfer_provisioned"

    log_success "Finished provisioning $node_host"
done

log_success "All hosts provisioned successfully!"
