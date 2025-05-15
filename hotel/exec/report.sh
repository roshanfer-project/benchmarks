#!/bin/bash

if [ -d ".log" ]; then
    rm -rf ".log"
fi

mkdir ".log"
cd ".log"

names=("geo" "rate" "profile" "reservation" "search" "frontend")

for name in "${names[@]}"; do
    c_name="${name}-sidecar"
    docker container logs $c_name &> "${c_name}.log"
done