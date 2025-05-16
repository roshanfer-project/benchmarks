#!/bin/bash


pid=$(docker inspect --format '{{.State.Pid}}' "$1")
sudo perf record -F 99 -g -e cycles:u -p "$pid" --call-graph dwarf -o "$1.prof"