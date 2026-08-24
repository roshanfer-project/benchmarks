#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
REPO_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"
HOSTS_FILE="${HOSTS_FILE:-"$REPO_ROOT/hosts.txt"}"
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

# shellcheck source=/dev/null
source "$REPO_ROOT/scripts/pick_github_ssh_key.sh"

SSH_OPTS=${SSH_OPTS:-"-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"}
REPO_URL="${REPO_URL:-git@github.com:farzad1132/roshanfer-experiments.git}"

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

if [ ${#HOSTS[@]} -eq 0 ]; then
    log_error "No hosts found in $HOSTS_FILE"
    exit 1
fi

parse_host_entry() {
    local entry=$1
    if [[ "$entry" != *"@"* ]]; then
        log_error "Host line must be user@host: $entry"
        exit 1
    fi
    CURRENT_USER="${entry%%@*}"
    CURRENT_HOST="${entry#*@}"
}

ssh_exec() {
    local user=$1
    local host=$2
    local cmd=$3
    ssh $SSH_OPTS "$user@$host" "$cmd"
}

sanitize_log_id() {
    local s=$1
    s="${s//@/_at_}"
    s="${s//\//_}"
    echo "$s" | tr -c 'A-Za-z0-9_.-' '_'
}

# One host — invoked inside subshell; all output redirected to per-host log by caller.
provision_single_host() {
    set -e
    local entry=$1
    parse_host_entry "$entry"
    local node_user="$CURRENT_USER"
    local node_host="$CURRENT_HOST"

    log_loc() { echo -e "[${node_host}] ${BLUE}[INFO]${NC} $1"; }
    ok_loc() { echo -e "[${node_host}] ${GREEN}[SUCCESS]${NC} $1"; }

    log_loc "Provisioning ($node_user)..."

    if [ "$FORCE_PROVISION" != "true" ] && ssh_exec "$node_user" "$node_host" "[ -f .roshanfer_provisioned ]"; then
        ok_loc "Already provisioned (.roshanfer_provisioned). Skipping (-f to force)."
        return 0
    fi

    log_loc "[1/4] Setting up SSH keys..."
    ssh_exec "$node_user" "$node_host" "mkdir -p ~/.ssh && chmod 700 ~/.ssh"
    _gh_key="$(pick_github_ssh_key || true)"
    if [ -n "$_gh_key" ]; then
        scp $SSH_OPTS "$_gh_key" "$node_user@$node_host:~/.ssh/id_ed25519"
        scp $SSH_OPTS "${_gh_key}.pub" "$node_user@$node_host:~/.ssh/id_ed25519.pub"
        ssh_exec "$node_user" "$node_host" "chmod 600 ~/.ssh/id_ed25519 && chmod 644 ~/.ssh/id_ed25519.pub"
        unset _gh_key
    else
        _proto="${GIT_PROTOCOL:-ssh}"
        _proto="${_proto,,}"
        if [ "$_proto" = "ssh" ]; then
            echo -e "[${node_host}] ${RED}[ERROR]${NC} No OpenSSH GitHub key on this control node. Re-run ./scripts/cloudlab_enter.sh from your laptop (or copy a key + .pub here)."
            exit 1
        fi
        log_loc "No OpenSSH GitHub key to copy. Skipping key copy."
        unset _gh_key
    fi

    log_loc "[2/4] Installing Go..."
    cat "$SCRIPT_DIR/install_go.sh" | ssh_exec "$node_user" "$node_host" "bash -s"

    log_loc "[3/4] Clone / pull repo..."
    ssh_exec "$node_user" "$node_host" "ssh-keyscan github.com >> ~/.ssh/known_hosts 2>/dev/null"
    DIR_NAME="roshanfer-experiments"

    CLONE_CMD="if [ ! -d '$DIR_NAME' ]; then
                   git clone $REPO_URL $DIR_NAME;
               else
                   echo 'Repo exists, pulling latest...';
                   cd $DIR_NAME && git pull;
               fi"

    ssh_exec "$node_user" "$node_host" "$CLONE_CMD"

    log_loc "Initializing submodules (benchmarks, rwg)..."
    ssh_exec "$node_user" "$node_host" "cd $DIR_NAME && git submodule update --init --recursive benchmarks rwg"

    log_loc "Building rwg..."
    ssh_exec "$node_user" "$node_host" "cd $DIR_NAME/rwg && /usr/local/go/bin/go build ."

    log_loc "[4/4] High performance settings..."
    cat "$SCRIPT_DIR/high_perf.sh" | ssh_exec "$node_user" "$node_host" "sudo bash -s"

    ssh_exec "$node_user" "$node_host" "touch .roshanfer_provisioned"

    ok_loc "Finished provisioning"
}

if [ "${#HOSTS[@]}" -eq 0 ]; then
    log_error "No hosts in $HOSTS_FILE"
    exit 1
fi

_proto="${GIT_PROTOCOL:-ssh}"
_proto="${_proto,,}"
if [ "$_proto" = "ssh" ] && ! pick_github_ssh_key >/dev/null; then
    log_error "No OpenSSH GitHub key on this control node. Re-run ./scripts/cloudlab_enter.sh from your laptop (or copy a key + .pub here)."
    exit 1
fi
unset _proto

RUN_STAMP="$(date -u +%Y%m%d_%H%M%S)"
PROVISION_LOG_DIR="${PROVISION_LOG_DIR:-${SCRIPT_DIR}/provision_logs/run_${RUN_STAMP}_$$}"
mkdir -p "$PROVISION_LOG_DIR"

# Default: fan out all hosts. Optional: MAX_PARALLEL_PROVISION=N throttle.
if [ -n "${MAX_PARALLEL_PROVISION+x}" ] && [ -n "$MAX_PARALLEL_PROVISION" ]; then
    max_jobs="$MAX_PARALLEL_PROVISION"
else
    max_jobs="${#HOSTS[@]}"
fi
if [ "$max_jobs" -lt 1 ]; then
    max_jobs="${#HOSTS[@]}"
fi

log_info "Starting provisioning for ${#HOSTS[@]} hosts (max_parallel=${max_jobs}, logs=${PROVISION_LOG_DIR})..."

declare -a pids=()
declare -a metas=()
declare -a fail_entries=()

for entry in "${HOSTS[@]}"; do
    safe=$(sanitize_log_id "$entry")
    logf="${PROVISION_LOG_DIR}/provision_host_${safe}.log"

    while [ "${#pids[@]}" -ge "$max_jobs" ]; do
        if ! wait "${pids[0]}"; then
            fail_entries+=("${metas[0]}")
        fi
        pids=("${pids[@]:1}")
        metas=("${metas[@]:1}")
    done

    (
        provision_single_host "$entry"
    ) >"$logf" 2>&1 &
    pids+=("$!")
    metas+=("$entry|$logf")
done

while [ "${#pids[@]}" -gt 0 ]; do
    if ! wait "${pids[0]}"; then
        fail_entries+=("${metas[0]}")
    fi
    pids=("${pids[@]:1}")
    metas=("${metas[@]:1}")
done

log_info "=== Provisioning summary (${#HOSTS[@]} hosts) ==="
log_info "Log directory: $PROVISION_LOG_DIR"

fail_count="${#fail_entries[@]}"
if [ "$fail_count" -eq 0 ]; then
    log_success "All hosts provisioned successfully."
    exit 0
fi

log_error "$fail_count host(s) failed:"
for fe in "${fail_entries[@]}"; do
    e="${fe%%|*}"
    lf="${fe#*|}"
    log_error "  - $e (log: $lf)"
done
exit 1
