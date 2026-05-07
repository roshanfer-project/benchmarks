#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
declare -a log_pids=()
for svc in backend1 backend2 backend3 backend4 backend5 backend6 backend7 backend8 frontend frontend-grpc rajomon-client; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
      if [ "$MODE" = "sidecar" ]; then
        kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
      fi
    ) &
    log_pids+=($!)
  done
done
for pid in "${log_pids[@]}"; do wait "$pid" || true; done

if [ "$MODE" = "sidecar" ]; then
  declare -a ing_pids=()
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
    ) &
    ing_pids+=($!)
  done
  for pid in "${ing_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "sidecar" ] && [ "${COLLECT_SIDECAR_NANOLOG:-}" = "1" ]; then
  declare -a cp_pids=()
  for svc in backend1 backend2 backend3 backend4 backend5 backend6 backend7 backend8 frontend frontend-grpc rajomon-client; do
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
