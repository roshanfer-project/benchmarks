#!/bin/bash

sudo docker-compose -f ./docker-compose.yaml down -v

# List of process names to check and act upon if found
names=("geo.o" "rate.o" "profile.o" "reservation.o" "search.o" "frontend.o")

for name in "${names[@]}"; do
    if pgrep "$name" > /dev/null; then
        echo "Process '$name' exists. killing..."
        pkill -9 "$name"
    else
        echo "Process '$name' not running."
    fi
done