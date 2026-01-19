#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CONFIG_FILE="$SCRIPT_DIR/config.env"
HOSTS_FILE="${HOSTS_FILE:-"$SCRIPT_DIR/hosts.txt"}"

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

# Universal cleanup function
cleanup_node() {
    local user=$1
    local host=$2
    local role=$3 # Just for logging

    echo "Cleaning up $role node: $host (User: $user)..."

    # 1. Try BOTH uninstall scripts (handle role switches)
    ssh $SSH_OPTS "$user@$host" "
        if [ -f /usr/local/bin/k3s-uninstall.sh ]; then /usr/local/bin/k3s-uninstall.sh; fi
        if [ -f /usr/local/bin/k3s-agent-uninstall.sh ]; then /usr/local/bin/k3s-agent-uninstall.sh; fi
    " || echo "  Warning: specific uninstall scripts failed or missing."

    # 2. Kill Processes & Services (BOTH server and agent types)
    echo "  Killing processes and stopping services..."
    ssh $SSH_OPTS "$user@$host" "
        sudo systemctl stop k3s k3s-agent 2>/dev/null || true
        sudo systemctl disable k3s k3s-agent 2>/dev/null || true
        sudo systemctl reset-failed k3s k3s-agent 2>/dev/null || true
        sudo killall -9 k3s 2>/dev/null || true
    "

    # 3. Clean Directories & Routes (Fixes 'i/o timeout' due to stale host-gw routes)
    # Added forceful unmount loops
    echo "  Cleaning up directories and stale routes..."
    ssh $SSH_OPTS "$user@$host" "
        # Unmount stuborn dirs
        for mount in \$(mount | grep -E '/var/lib/kubelet|/run/k3s' | awk '{print \$3}'); do sudo umount -l \$mount; done
        
        # Remove dirs
        sudo rm -rf /var/lib/rancher /var/lib/kubelet /etc/rancher /etc/cni/net.d /var/lib/cni /run/k3s /run/flannel /etc/systemd/system/k3s.service /etc/systemd/system/k3s-agent.service
        
        # Flush routes
        sudo ip route flush $CLUSTER_CIDR
    "

    # 4. Run safe flush
    echo "  Flushing iptables/conntrack/ipset..."
    scp $SSH_OPTS "$SCRIPT_DIR/safe_flush_rules.sh" "$user@$host:/tmp/safe_flush_rules.sh"
    ssh $SSH_OPTS "$user@$host" "chmod +x /tmp/safe_flush_rules.sh && sudo /tmp/safe_flush_rules.sh && rm /tmp/safe_flush_rules.sh"
}

# --- UNINSTALL AGENTS ---
for entry in "${AGENT_ENTRIES[@]}"; do
    parse_host_entry "$entry"
    cleanup_node "$CURRENT_USER" "$CURRENT_HOST" "Agent"
done

# --- UNINSTALL SERVER ---
cleanup_node "$SERVER_USER" "$SERVER_HOST" "Server"


# --- PROVISIONING CLEANUP ---
# Skipped: Provisioning (Go, Git, SSH keys) is preserved as requested.
# If you want to full wipe, manually delete the repo and /usr/local/go


echo "Cleaning up local kubeconfig..."

rm -f "$SCRIPT_DIR/kubeconfig"

echo "Cluster reset complete."
