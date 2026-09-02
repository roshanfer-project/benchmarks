#!/bin/bash
# Regenerates each test benchmark via callgraph-framework `gen`, then full DOT PDF
# and ACM service-level PDF (`callgraph-service.pdf`; requires repo-root `.venv` for Python).
set -e
cd "$(dirname "$0")"
TESTS_DIR="$(pwd)"
FRAMEWORK_DIR="$TESTS_DIR/../callgraph-framework"

for dir in */; do
  dir="${dir%/}"
  if [ -f "$dir/callgraph.json" ]; then
    echo "Generating $dir..."
    (
      cd "$FRAMEWORK_DIR"
      go run ./cmd/gen "$TESTS_DIR/$dir/callgraph.json" -o "$TESTS_DIR/$dir"
      go run ./cmd/viz -paper "$TESTS_DIR/$dir/callgraph.json" -o "$TESTS_DIR/$dir/callgraph-service.pdf"
    )
    (cd "$TESTS_DIR/$dir" && go mod tidy)
    echo "Done: $dir"
  fi
done
echo "All test benchmarks generated."
