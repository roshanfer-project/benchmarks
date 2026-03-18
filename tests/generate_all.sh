#!/bin/bash
set -e
cd "$(dirname "$0")"
TESTS_DIR="$(pwd)"
FRAMEWORK_DIR="$TESTS_DIR/../callgraph-framework"

for dir in */; do
  dir="${dir%/}"
  if [ -f "$dir/callgraph.json" ]; then
    echo "Generating $dir..."
    (cd "$FRAMEWORK_DIR" && go run ./cmd/gen "$TESTS_DIR/$dir/callgraph.json" -o "$TESTS_DIR/$dir")
    echo "Done: $dir"
  fi
done
echo "All test benchmarks generated."
