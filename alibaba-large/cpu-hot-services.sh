#!/usr/bin/env bash
# Rank workload pods from cpu_utilization_summary.csv by utilization_max (desc). No Python required.
set -euo pipefail

usage() {
  echo "Usage: $0 <cpu_utilization_summary.csv> [head_lines]" >&2
  echo "  Columns: utilization_max, pod, limit (cores)" >&2
  exit 1
}

[[ $# -ge 1 ]] || usage
[[ -f "$1" ]] || { echo "Not found: $1" >&2; exit 1; }
N="${2:-30}"

# Skip header; only ms-* workload pods; sort by utilization_max numeric descending.
awk -F, '
  NR == 1 { next }
  $2 ~ /^ms-/ {
    printf "%f\t%s\t%s\n", $6, $2, $7
  }
' "$1" | sort -t $'\t' -k1,1nr | head -n "$N" | while IFS=$'\t' read -r util pod lim; do
  printf '%s\tutil_max=%s\tlimit=%s\n' "$pod" "$util" "$lim"
done
