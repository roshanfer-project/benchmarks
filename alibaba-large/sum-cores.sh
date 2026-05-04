#!/usr/bin/env bash
# Sum per-node "cpu" fields in callgraph.json (workload resource budget).
# Excludes the synthetic USER node by default. No Python required.
set -euo pipefail

usage() {
  echo "Usage: $0 [--with-user] [callgraph.json]" >&2
  echo "  Default callgraph: ./callgraph.json next to this script." >&2
  echo "  --with-user    Include USER node cpu in the total." >&2
  exit 1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GRAPH="$SCRIPT_DIR/callgraph.json"
WITH_USER=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-user) WITH_USER=1; shift ;;
    -h | --help) usage ;;
    -*)
      echo "Unknown option: $1" >&2
      usage
      ;;
    *)
      GRAPH="$1"
      shift
      ;;
  esac
done

[[ -f "$GRAPH" ]] || { echo "Not found: $GRAPH" >&2; exit 1; }

if command -v jq >/dev/null 2>&1; then
  if [[ "$WITH_USER" -eq 1 ]]; then
    jq '[.nodes[].cpu] | add' "$GRAPH"
  else
    jq '[.nodes[] | select(.id != "USER") | .cpu] | add' "$GRAPH"
  fi
  exit 0
fi

# No jq: stream lines; assume each node block has "id" before "cpu" (true for gen output).
sum=0
cur_id=""
while IFS= read -r line || [[ -n "$line" ]]; do
  if [[ "$line" =~ \"id\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
    cur_id="${BASH_REMATCH[1]}"
  elif [[ "$line" =~ ^[[:space:]]*\"cpu\"[[:space:]]*:[[:space:]]*([0-9]+) ]]; then
    if [[ "$WITH_USER" -eq 0 && "$cur_id" == "USER" ]]; then
      continue
    fi
    ((sum += BASH_REMATCH[1]))
  fi
done <"$GRAPH"

echo "$sum"
