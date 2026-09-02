#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
REPO_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"
# shellcheck source=/dev/null
source "$REPO_ROOT/scripts/elapsed.sh"
CONFIG_FILE="$SCRIPT_DIR/config.env"
HOSTS_FILE="${HOSTS_FILE:-"$REPO_ROOT/hosts.txt"}"

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

ng="${NUM_GENERATORS:-0}"
if [[ "$ng" =~ ^[0-9]+$ ]] && [ "$ng" -gt 0 ]; then
    HOSTS=("${HOSTS[@]:ng}")
fi

if [ ${#HOSTS[@]} -eq 0 ]; then
    echo "No hosts found in $HOSTS_FILE (after skipping NUM_GENERATORS=$ng)"
    exit 1
fi

parse_host_entry() {
    local entry=$1
    if [[ "$entry" != *"@"* ]]; then
        echo "Host line must be user@host: $entry"
        exit 1
    fi
    CURRENT_USER="${entry%%@*}"
    CURRENT_HOST="${entry#*@}"
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

    # Use standard K3s uninstall scripts
    # We try both in case roles have been swapped on the same hardware
    ssh $SSH_OPTS "$user@$host" "
        if [ -f /usr/local/bin/k3s-uninstall.sh ]; then 
            echo 'Running k3s-uninstall.sh...'
            sudo /usr/local/bin/k3s-uninstall.sh
        fi
        
        if [ -f /usr/local/bin/k3s-agent-uninstall.sh ]; then 
            echo 'Running k3s-agent-uninstall.sh...'
            sudo /usr/local/bin/k3s-agent-uninstall.sh
        fi
    "
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

echo "Cleaning up local kubeconfig..."
rm -f "$SCRIPT_DIR/kubeconfig"

echo "Cluster reset complete."
