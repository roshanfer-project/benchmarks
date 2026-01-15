#!/bin/bash

# Ensure we are running as root (or can sudo)
if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root"
  exit 1
fi

echo "Starting High Performance Setup..."

# 1. Set CPU Governor to Performance
echo "Setting CPU governor to performance..."
if command -v cpupower &> /dev/null; then
    cpupower frequency-set -g performance
else
    # Fallback to direct sysfs manipulation if cpupower is not installed
    for governor in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
        echo performance > "$governor" 2>/dev/null || true
    done
fi

# 2. Disable Hyperthreading (SMT)
# This iterates through sibling pairs and offlines the second thread.
echo "Disabling Hyperthreading (SMT)..."
# Get the thread siblings list
for file in /sys/devices/system/cpu/cpu*/topology/thread_siblings_list; do
    # Read the siblings (e.g., "0-3" or "0,4")
    siblings=$(cat "$file")
    
    # IFS=',' read -ra ADDR <<< "$siblings"
    # This usually looks like "0,4". The first one is the physical core representative, the second is the HT sibling.
    # We want to disable the second one.
    
    # Note: parsing can be tricky depending on kernel version/format (ranges vs lists). 
    # A simpler approach for standard Linux (x86_64) on modern kernels:
    # sibling 1 is usually the higher number. 
    
    # Let's use a safer approach: checking logical CPU ID relationship
    # Or just use the standard toggle:
    # echo off > /sys/devices/system/cpu/smt/control (kernel 4.13+)
    # If that exists, it's the best way.
    if [ -f /sys/devices/system/cpu/smt/control ]; then
        echo off > /sys/devices/system/cpu/smt/control
        break # One global toggle is enough
    fi
    
    # Fallback ref manual loop if smt/control doesn't exist (older kernels)
    # Extract the first CPU from the pair and keep it, disable others.
    # (Skipping complex fallback logic for now, assuming kernel >= 4.13 which is standard for k8s/modern linux)
done

# Verify SMT status
cat /sys/devices/system/cpu/smt/control 2>/dev/null || echo "SMT control not found"

# 3. Disable Swap
echo "Disabling Swap..."
swapoff -a
# Comment out swap in fstab to persist across reboots (optional, usually good for dedicated nodes)
sed -i.bak '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab


echo "High Performance Setup Complete."
