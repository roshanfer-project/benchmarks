#!/bin/bash
# Removes generate_all.sh output so each test dir can be regenerated cleanly.
# Never deletes callgraph.json.
set -e
cd "$(dirname "$0")"

for dir in */; do
  dir="${dir%/}"
  if [ -f "$dir/callgraph.json" ]; then
    echo "Cleaning $dir..."
    debug_env=""
    if [ -f "$dir/k8s/sidecar-debug-glog.env" ]; then
      debug_env=$(mktemp)
      cp "$dir/k8s/sidecar-debug-glog.env" "$debug_env"
    fi
    rm -rf \
      "$dir/services" \
      "$dir/utils" \
      "$dir/pkg" \
      "$dir/proto" \
      "$dir/protobuf" \
      "$dir/dagor" \
      "$dir/dagor_init" \
      "$dir/rajomon_init" \
      "$dir/k8s"
    rm -f \
      "$dir/go.mod" \
      "$dir/go.sum" \
      "$dir/Dockerfile" \
      "$dir/docker-bake.hcl" \
      "$dir/.dockerignore" \
      "$dir/entry_path.txt" \
      "$dir/build.sh" \
      "$dir/deploy.sh" \
      "$dir/destroy.sh" \
      "$dir/collect_logs.sh" \
      "$dir/run.sh" \
      "$dir/run-plain.sh" \
      "$dir/callgraph.pdf" \
      "$dir/callgraph-service.pdf"
    if [ -n "$debug_env" ]; then
      mkdir -p "$dir/k8s"
      mv "$debug_env" "$dir/k8s/sidecar-debug-glog.env"
    fi
  fi
done
echo "All test benchmarks cleaned."
