#!/bin/bash
# Collect logs for tests/test1

# Output directory is passed as environment variable or argument?
# Collector.py passes OUTPUT_DIR env var.
OUTPUT_DIR=${OUTPUT_DIR:-"./logs"}
mkdir -p "$OUTPUT_DIR"

echo "Collecting logs to $OUTPUT_DIR..."

# Collect App Logs
APP_PODS=$(kubectl get pods -l app=app -o jsonpath='{.items[*].metadata.name}')
for pod in $APP_PODS; do
    echo "  Collecting logs for $pod..."
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
    # If there are sidecars (envoy/sidecar), collect those too if accessible
    # (Assuming single container or default container. If sidecar, users might want -c sidecar)
    # Check containers
    # Try to collect sidecar logs, suppress error if container missing
    kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-sidecar.log"
    kubectl logs "$pod" -c app > "$OUTPUT_DIR/${pod}-app.log" 2>&1 || rm -f "$OUTPUT_DIR/${pod}-app.log"
done

# Collect Ingress Logs
ING_PODS=$(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}')
for pod in $ING_PODS; do
    echo "  Collecting logs for $pod..."
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
done

echo "Logs collected."
