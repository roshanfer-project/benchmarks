#!/bin/bash

OUTPUT_DIR=${OUTPUT_DIR:-"./logs"}
mkdir -p "$OUTPUT_DIR"

echo "Collecting logs to $OUTPUT_DIR..."

SERVICES="graph posts home user compose nginx nginx-grpc rajomon-client"

declare -a log_pids=()
for svc in $SERVICES; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-sidecar.log"
      kubectl logs "$pod" -c app > "$OUTPUT_DIR/${pod}-app.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-app.log"
    ) &
    log_pids+=($!)
  done
done

for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
  (
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
    kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-sidecar.log"
  ) &
  log_pids+=($!)
done

for pid in "${log_pids[@]}"; do wait "$pid" || true; done

if [ "${COLLECT_SIDECAR_NANOLOG:-}" = "1" ]; then
  declare -a cp_pids=()
  for svc in $SERVICES; do
    for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
      ( kubectl cp "$pod:/compressedLog" "$OUTPUT_DIR/${pod}-sidecar.clog" -c sidecar 2>/dev/null || true ) &
      cp_pids+=($!)
    done
  done
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
    ( kubectl cp "$pod:/compressedLog" "$OUTPUT_DIR/${pod}-ingress-sidecar.clog" -c sidecar 2>/dev/null || true ) &
    cp_pids+=($!)
  done
  for pid in "${cp_pids[@]}"; do wait "$pid" || true; done
fi

echo "Logs collected."
