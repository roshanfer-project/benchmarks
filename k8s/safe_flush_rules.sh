#!/bin/bash
# safe_flush_rules.sh
# Flushes K3s/CNI iptables rules while explicitly preserving SSH.

set -e

# 1. Ensure SSH is allowed in INPUT/OUTPUT before flushing anything
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT
iptables -A OUTPUT -p tcp --sport 22 -m conntrack --ctstate ESTABLISHED -j ACCEPT

# 2. Flush standard chains (but set policy to ACCEPT first to avoid lockout)
iptables -P INPUT ACCEPT
iptables -P FORWARD ACCEPT
iptables -P OUTPUT ACCEPT

iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X
iptables -t raw -F
iptables -t raw -X

# 3. Clean up CNI/Flannel links if they exist
ip link delete cni0 2>/dev/null || true
ip link delete flannel.1 2>/dev/null || true
ip link delete flannel-v6.1 2>/dev/null || true
# ip link delete cilium_vxlan 2>/dev/null || true

# 4. Flush ipsets (prevent stale service lookups)
if command -v ipset &> /dev/null; then
    ipset flush || true
    ipset destroy || true
fi

# 5. Flush conntrack (prevent stale connection caching causing timeouts)
if command -v conntrack &> /dev/null; then
    conntrack -F || true
fi


echo "Iptables flushed and network interfaces cleaned. SSH should remain active."
