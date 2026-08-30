#!/bin/bash
# Regenerates each test benchmark and alibaba-large via callgraph-framework `gen`,
# then ACM service-level PDF (`callgraph-service.pdf`; requires repo-root `.venv` for Python).
set -e
cd "$(dirname "$0")"
TESTS_DIR="$(pwd)"
FRAMEWORK_DIR="$TESTS_DIR/../callgraph-framework"

generate_one() {
  dest="$1"
  name="$(basename "$dest")"
  echo "Generating $name..."
  (
    cd "$FRAMEWORK_DIR"
    go run ./cmd/gen "$dest/callgraph.json" -o "$dest"
    go run ./cmd/viz -paper "$dest/callgraph.json" -o "$dest/callgraph-service.pdf"
  )
  (cd "$dest" && go mod tidy)
  echo "Done: $name"
}

for dir in */; do
  dir="${dir%/}"
  if [ -f "$dir/callgraph.json" ]; then
    generate_one "$TESTS_DIR/$dir"
  fi
done

ALIBABA_DIR="$TESTS_DIR/../alibaba-large"
if [ -f "$ALIBABA_DIR/callgraph.json" ]; then
  generate_one "$ALIBABA_DIR"
fi

echo "All test benchmarks and alibaba-large generated."
