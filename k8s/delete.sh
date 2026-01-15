#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CONFIG_FILE="$SCRIPT_DIR/config.env"
HOSTS_FILE="$SCRIPT_DIR/hosts.txt"

# Requirements
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Config file not found: $CONFIG_FILE"
    exit 1
fi
source "$CONFIG_FILE"

if [ ! -f "$HOSTS_FILE" ]; then
    echo "Hosts file not found: $HOSTS_FILE"
    exit 1
fi

# Filter out comments and empty lines
mapfile -t HOSTS < <(grep -vE '^\s*#|^\s*$' "$HOSTS_FILE")

if [ ${#HOSTS[@]} -eq 0 ]; then
    echo "No hosts found in $HOSTS_FILE"
    exit 1
fi

# Helper to parse "user@host" or just "host"
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
SERVER_USER="$CURRENT_USER"
SERVER_HOST="$CURRENT_HOST"

AGENT_ENTRIES=("${HOSTS[@]:1}")

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

echo "Uninstalling Agents..."
for entry in "${AGENT_ENTRIES[@]}"; do
    parse_host_entry "$entry"
    agent_user="$CURRENT_USER"
    agent_host="$CURRENT_HOST"

    echo "  Connecting to $agent_host (User: $agent_user)..."
    
    # Patch the uninstall script to prevent it from killing the network (skip iptables cleanup)
    ssh $SSH_OPTS "$agent_user@$agent_host" "if [ -f /usr/local/bin/k3s-agent-uninstall.sh ]; then sudo sed -i 's/iptables-save/# iptables-save/g' /usr/local/bin/k3s-agent-uninstall.sh && sudo sed -i 's/iptables-restore/# iptables-restore/g' /usr/local/bin/k3s-agent-uninstall.sh; fi"
    ssh $SSH_OPTS "$agent_user@$agent_host" "/usr/local/bin/k3s-agent-uninstall.sh" || echo "  Warning: Uninstall failed on $agent_host or assumed already clean."
    
    # Run safe flush
    echo "  Cleaning up iptables on $agent_host..."
    scp $SSH_OPTS "$SCRIPT_DIR/safe_flush_rules.sh" "$agent_user@$agent_host:/tmp/safe_flush_rules.sh"
    ssh $SSH_OPTS "$agent_user@$agent_host" "chmod +x /tmp/safe_flush_rules.sh && sudo /tmp/safe_flush_rules.sh && rm /tmp/safe_flush_rules.sh"
    
    # Force kill any stuck processes
    ssh $SSH_OPTS "$agent_user@$agent_host" "sudo killall -9 k3s 2>/dev/null || true"
    # Stop legacy kubelet if present (kubespray leftovers)
    ssh $SSH_OPTS "$agent_user@$agent_host" "sudo systemctl stop kubelet 2>/dev/null && sudo systemctl disable kubelet 2>/dev/null || true"
done

echo "Uninstalling Server..."
echo "  Connecting to $SERVER_HOST (User: $SERVER_USER)..."
# Patch the uninstall script to prevent it from killing the network (skip iptables cleanup)
ssh $SSH_OPTS "$SERVER_USER@$SERVER_HOST" "if [ -f /usr/local/bin/k3s-uninstall.sh ]; then sudo sed -i 's/iptables-save/# iptables-save/g' /usr/local/bin/k3s-uninstall.sh && sudo sed -i 's/iptables-restore/# iptables-restore/g' /usr/local/bin/k3s-uninstall.sh; fi"
ssh $SSH_OPTS "$SERVER_USER@$SERVER_HOST" "/usr/local/bin/k3s-uninstall.sh" || echo "  Warning: Uninstall failed on $SERVER_HOST or assumed already clean."

# Run safe flush on server
echo "  Cleaning up iptables on $SERVER_HOST..."
scp $SSH_OPTS "$SCRIPT_DIR/safe_flush_rules.sh" "$SERVER_USER@$SERVER_HOST:/tmp/safe_flush_rules.sh"
ssh $SSH_OPTS "$SERVER_USER@$SERVER_HOST" "chmod +x /tmp/safe_flush_rules.sh && sudo /tmp/safe_flush_rules.sh && rm /tmp/safe_flush_rules.sh"

# Force kill any stuck processes
ssh $SSH_OPTS "$SERVER_USER@$SERVER_HOST" "sudo killall -9 k3s 2>/dev/null || true"
ssh $SSH_OPTS "$SERVER_USER@$SERVER_HOST" "sudo systemctl stop kubelet 2>/dev/null && sudo systemctl disable kubelet 2>/dev/null || true"


# --- PROVISIONING CLEANUP ---
# Skipped: Provisioning (Go, Git, SSH keys) is preserved as requested.
# If you want to full wipe, manually delete the repo and /usr/local/go


echo "Cleaning up local kubeconfig..."

rm -f "$SCRIPT_DIR/kubeconfig"

echo "Cluster reset complete."
