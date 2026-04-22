#!/bin/bash
# Usage: $0 PROTOCOL API OUTPUT_DIR [--ignore-errors]
# Requires RWG_RATES and RWG_DURATIONS (comma-separated), set by the executor.
if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]; then
  echo "Error: Missing required arguments"
  echo "Usage: $0 PROTOCOL API OUTPUT_DIR [--ignore-errors]"
  exit 1
fi
: "${RWG_RATES:?RWG_RATES must be set}"
: "${RWG_DURATIONS:?RWG_DURATIONS must be set}"

if [ "$4" == "--ignore-errors" ]; then
    ignore_errors=true
else
    ignore_errors=false
fi

protocol=$1
API=$2
output_dir="$3/out-$API.csv"
address="${TARGET_ADDR:-192.168.1.100}"

if [ "$protocol" == "grpc" ]; then
    echo "GRPC is not supported"
    exit 1
else
    if [ "$API" = "search-hotel" ]; then
        url="http://$address:3000/hotels?lat=37.7867&lon=-122.4112&inDate=2024-08-15&outDate=2024-08-17"
    elif [ "$API" = "reserve-hotel" ]; then
        url="http://$address:3000/reservation?inDate=2025-05-20&outDate=2025-05-22&hotelId=4&customerName=Alice&username=Cornell_1&password=1111111111&number=1"
    else
        echo "Unknown hotel API: $API"
        exit 1
    fi
    if [ "$ignore_errors" = true ]; then
        "$RWG_BINARY" run --url $url -d exp -D "$RWG_DURATIONS" -r "$RWG_RATES" -w 10000 -o $output_dir -t 30 --ignore-errors
        exit 0
    else
        "$RWG_BINARY" run --url $url -d exp -D "$RWG_DURATIONS" -r "$RWG_RATES" -w 10000 -o $output_dir -t 30
        exit "$?"
    fi
fi
