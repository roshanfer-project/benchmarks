#!/bin/bash

if [ -d ".log" ]; then
    rm -rf ".log"
fi

mkdir ".log"
cd ".log"

names=("user" "geo" "rate" "profile" "reservation" "search" "frontend")

for name in "${names[@]}"; do
    c_name="${name}-sidecar"
    #docker container logs $c_name &> "${c_name}.log"
    docker container cp $c_name:./compressedLog "${c_name}.clog"
    ../../../../sidecar/NanoLog/runtime/decompressor decompress "${c_name}.clog" > "${c_name}.log"
    ../metrics.py --file "${c_name}.log" --no-print
done