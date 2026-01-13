# K3s Cluster Scripts

This folder contains scripts to manage a lightweight, high-performance Kubernetes cluster using **K3s** and **Cilium**.

## Requirements
*   **SSH Access**: These scripts run from your local machine (or a management node) and require SSH access to all target nodes using the user defined in `config.env` (default is your current user).
*   **Sudo**: The SSH user must have passwordless sudo access on the target nodes.
*   **Cilium CLI**: The scripts will attempt to install the Cilium CLI if not present on the machine running the script.

## Setup

1.  **Configure Environment**: Edit `config.env` to set versions and resource reservations.
    ```bash
    # config.env
    K3S_VERSION="v1.31.4+k3s1"
    CILIUM_VERSION="1.16.5"
    CPU_MANAGER_POLICY="static"
    ```

2.  **Define Hosts**: Edit `hosts.txt` to list your nodes.
    *   **Line 1**: Control Plane (Server)
    *   **Lines 2+**: Workers (Agents)
    ```text
    node1.example.com
    node2.example.com
    ```

## Usage

### Create Cluster
Creates the cluster, joins agents, and installs Cilium.
```bash
./create.sh
```

### Reset Cluster
**WARNING**: Destructive.
Uninstalls K3s from all nodes (via SSH) and immediately recreates the cluster.
```bash
./reset.sh
```

### Delete Cluster
**WARNING**: Destructive.
Uninstalls K3s from all nodes and removes the local kubeconfig.
```bash
./delete.sh
```

## Features enabled
*   **Cilium Networking**: Flannel is disabled in favor of Cilium.
*   **Static CPU Manager**: Kubelet is configured with `--cpu-manager-policy=static` for high-performance workloads demanding exclusive core pinning.
