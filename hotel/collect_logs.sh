#!/bin/bash
# Collect logs for hotel benchmark

# Output directory is passed as environment variable or argument?
# Collector.py passes OUTPUT_DIR env var.
OUTPUT_DIR=${OUTPUT_DIR:-"./logs"}
mkdir -p "$OUTPUT_DIR"

echo "Collecting logs to $OUTPUT_DIR..."

# List of services in the hotel benchmark
SERVICES="frontend profile search geo rate reservation user frontend-grpc rajomon-client"

# Collect Service Logs
for svc in $SERVICES; do
    PODS=$(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}')
    for pod in $PODS; do
        echo "  Collecting logs for $pod..."
        # Default log (usually first container or default)
        kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
        
        # Collect sidecar and app specific container logs
        # This covers both plain (only app probably) and sidecar (app + sidecar) deployments
        kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-sidecar.log"
        kubectl logs "$pod" -c app > "$OUTPUT_DIR/${pod}-app.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-app.log"
    done
done

# Collect Ingress Logs
# Ingress label is app=ingress
ING_PODS=$(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}')
for pod in $ING_PODS; do
    echo "  Collecting logs for $pod..."
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
    # Ingress container is named 'sidecar' in ingress.yaml
    kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-sidecar.log"
done

if [ "${COLLECT_SIDECAR_NANOLOG:-}" = "1" ]; then
  for svc in $SERVICES; do
    for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
      kubectl cp "$pod:/compressedLog" "$OUTPUT_DIR/${pod}-sidecar.clog" -c sidecar 2>/dev/null || true
    done
  done
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
    kubectl cp "$pod:/compressedLog" "$OUTPUT_DIR/${pod}-ingress-sidecar.clog" -c sidecar 2>/dev/null || true
  done
fi

echo "Logs collected."
