#!/bin/bash

docker container logs ingress-sidecar &> ingress.log

docker container logs frontend-sidecar &> frontend.log

docker container logs search-sidecar &> search.log

docker container logs rate-sidecar &> rate.log

docker container logs geo-sidecar &> geo.log

docker container logs profile-sidecar &> profile.log

docker container logs reservation-sidecar &> reservation.log

# Build dslogs command with optional ID filter
DSLOGS_ARGS=(
    "./dslogs.py"
    "./ingress.log" "./frontend.log" "./search.log" "./rate.log" "./geo.log" "./reservation.log" "./profile.log" 
    "--filter" "QM"
    "--filter" "PPMClient"
    "--filter" "RPCForward"
    "--filter" "INGRESS"
    "--color"
    "-w" "30"
    "-W" "210"
)

# Add ID filter if provided
if [ $# -gt 0 ]; then
    for id in "$@"; do
        DSLOGS_ARGS+=("--filter-id")
        DSLOGS_ARGS+=("$id")
    done
fi

# Execute the command and redirect to trace.log
"${DSLOGS_ARGS[@]}" > trace.log