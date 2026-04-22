#!/bin/bash
# Usage: $0 PROTOCOL API OUTPUT_DIR
# Requires RWG_RATES and RWG_DURATIONS (comma-separated), set by the executor.
if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]; then
  echo "Error: Missing required arguments"
  echo "Usage: $0 PROTOCOL API OUTPUT_DIR"
  exit 1
fi
: "${RWG_RATES:?RWG_RATES must be set}"
: "${RWG_DURATIONS:?RWG_DURATIONS must be set}"

protocol=$1
API=$2
output_dir="$3/out-$API.csv"
address="${TARGET_ADDR:-192.168.1.100}"

if [ "$protocol" == "grpc" ]; then
    echo "GRPC is not supported"
    exit 1
else
    if [ "$API" = "compose-post" ]; then
        char="t"
        num="100"
        repeated_char=$(printf "%0.s$char" $(seq 1 $num))
        url="http://$address:3000/compose?text=$repeated_char"
        echo "url: $url"
    elif [ "$API" = "read-home-timeline" ]; then
        url="http://$address:3008/home"
    elif [ "$API" = "read-user-timeline" ]; then
        url="http://$address:3009/user"
    else
        echo "Unknown social API: $API"
        exit 1
    fi
    "$RWG_BINARY" run --url $url -d exp -D "$RWG_DURATIONS" -r "$RWG_RATES" -w 5000 -o $output_dir -t 15
    exit "$?"
fi
