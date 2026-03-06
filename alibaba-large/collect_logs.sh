#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
for svc in ms-12657 ms-14758 ms-18750 ms-19439 ms-21298 ms-25781 ms-25806 ms-2687 ms-33572 ms-38190 ms-40087 ms-41667 ms-43032 ms-43754 ms-44246 ms-45067 ms-51783 ms-51787 ms-53792 ms-56113 ms-5720 ms-58796 ms-62039 ms-64512 ms-66921 ms-67465 ms-70124 ms-7103 ms-9105; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
    if [ "$MODE" = "sidecar" ]; then
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
    fi
  done
done
if [ "$MODE" = "sidecar" ]; then
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
  done
fi
echo "Logs collected."
