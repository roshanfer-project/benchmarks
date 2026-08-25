#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
declare -a log_pids=()
for svc in ms-12657 ms-14758 ms-18750 ms-19439 ms-21298 ms-25781 ms-25806 ms-2687 ms-33572 ms-38190 ms-40087 ms-41667 ms-43032 ms-43754 ms-44246 ms-45067 ms-51783 ms-51787 ms-53792 ms-56113 ms-5720 ms-58796 ms-62039 ms-64512 ms-66921 ms-67465 ms-70124 ms-7103 ms-9105 ms-64512-grpc rajomon-client; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
      if [ "$MODE" = "roshanfer" ]; then
        kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
      fi
    ) &
    log_pids+=($!)
  done
done
for pid in "${log_pids[@]}"; do wait "$pid" || true; done

if [ "$MODE" = "roshanfer" ]; then
  declare -a ing_pids=()
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
    ) &
    ing_pids+=($!)
  done
  for pid in "${ing_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "roshanfer" ] && [ "${COLLECT_SIDECAR_NANOLOG:-}" = "1" ]; then
  declare -a cp_pids=()
  for svc in ms-12657 ms-14758 ms-18750 ms-19439 ms-21298 ms-25781 ms-25806 ms-2687 ms-33572 ms-38190 ms-40087 ms-41667 ms-43032 ms-43754 ms-44246 ms-45067 ms-51783 ms-51787 ms-53792 ms-56113 ms-5720 ms-58796 ms-62039 ms-64512 ms-66921 ms-67465 ms-70124 ms-7103 ms-9105 ms-64512-grpc rajomon-client; do
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
