#!/bin/bash

# Directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$( cd "$DIR/../.." && pwd )"
HOSTS_FILE="${HOSTS_FILE:-"$REPO_ROOT/hosts.txt"}"

if [ ! -f "$HOSTS_FILE" ]; then
    echo "Error: hosts.txt not found at $HOSTS_FILE"
    exit 1
fi

echo "Updating repository on all hosts listed in $HOSTS_FILE..."

while IFS= read -r host || [ -n "$host" ]; do
    # Skip empty lines and comments
    [[ $host =~ ^#.*$ ]] && continue
    [[ -z $host ]] && continue

    echo "=== Updating $host ==="
    ssh -n -o StrictHostKeyChecking=no -o ConnectTimeout=5 "$host" "cd ~/roshanfer-experiments && echo 'Pulling latest changes...' && git pull"
    
    if [ $? -eq 0 ]; then
        echo "Successfully updated $host"
    else
        echo "Failed to update $host"
    fi
    echo "-----------------------------------"
done < "$HOSTS_FILE"

echo "Update complete."
