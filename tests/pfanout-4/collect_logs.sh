#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
declare -a log_pids=()
for svc in backend1 backend2 backend3 backend4 frontend frontend-grpc rajomon-client; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
      if [ "$MODE" = "sidecar" ] || [ "$MODE" = "sidecar-lb" ]; then
        kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
      fi
      if [ "$MODE" = "envoy" ]; then
        kubectl logs "$pod" -c envoy > "$OUTPUT_DIR/${pod}-envoy.log" 2>&1
      fi
    ) &
    log_pids+=($!)
  done
done
for pid in "${log_pids[@]}"; do wait "$pid" || true; done

if [ "$MODE" = "sidecar" ] || [ "$MODE" = "sidecar-lb" ]; then
  declare -a ing_pids=()
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
    ) &
    ing_pids+=($!)
  done
  for pid in "${ing_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "envoy" ]; then
  declare -a ing_pids=()
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" -c envoy > "$OUTPUT_DIR/${pod}-envoy.log" 2>&1
    ) &
    ing_pids+=($!)
  done
  for pid in "${ing_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "envoy" ]; then
  ENVOY_METRICS_DIR=${ENVOY_METRICS_DIR:-./metrics/envoy}
  mkdir -p "$ENVOY_METRICS_DIR"
  declare -a stats_pids=()
  for svc in backend1 backend2 backend3 backend4 frontend frontend-grpc rajomon-client ingress; do
    for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
      [ -z "$pod" ] && continue
      app=$(kubectl get pod "$pod" -o jsonpath='{.metadata.labels.app}' 2>/dev/null)
      [ -z "$app" ] && continue
      ( kubectl cp "$pod:/tmp/envoy_stats.csv" "$ENVOY_METRICS_DIR/${app}.csv" -c envoy-stats 2>/dev/null || true ) &
      stats_pids+=($!)
    done
  done
  for pid in "${stats_pids[@]}"; do wait "$pid" || true; done
fi

if { [ "$MODE" = "sidecar" ] || [ "$MODE" = "sidecar-lb" ]; } && [ "${COLLECT_SIDECAR_NANOLOG:-}" = "1" ]; then
  declare -a cp_pids=()
  for svc in backend1 backend2 backend3 backend4 frontend frontend-grpc rajomon-client; do
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
