# K3s cluster scripts

Create / reset / delete a K3s cluster (Flannel host-gw).

## Repository layout

```text
k8s/
├── create.sh                  install server + agents, write kubeconfig
├── reset.sh                   delete then create
├── delete.sh                  uninstall K3s, remove local kubeconfig
├── config.env                 K3s version, CIDRs, kubelet reservations
├── update_repo.sh             git pull on every hosts.txt line
├── install_cpu_stats.sh       apply cpu-stats-exporter DaemonSet
├── cpu-stats-daemonset.yaml   DaemonSet manifest
└── cpu-stats-exporter/        per-node CPU stats image
```

## Requirements

* **SSH Access**: scripts run from the control node and SSH to every target as the user in `hosts.txt`.
* **Sudo**: that user must have passwordless sudo on the target nodes.

## Setup

1. **Configure Environment**: edit `config.env` for versions and resource reservations.
    ```bash
    # config.env
    K3S_VERSION="v1.31.4+k3s1"
    CPU_MANAGER_POLICY="static"
    ```

2. **Hosts**: repo-root `hosts.txt` (`user@host` per line). Skip the first `NUM_GENERATORS` lines; the first remaining host is the control plane, the rest are agents.
    ```text
    user@node1.example.com
    user@node2.example.com
    ```

## Usage

### Create Cluster
Installs K3s (server then agents) and deploys `cpu-stats-exporter`.
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

## Features
* **Flannel host-gw**: K3s CNI with `--flannel-backend=host-gw` (needs L2). Traefik is disabled. NodePort range is `3000-32767`.
* **Static CPU Manager**: kubelet `--cpu-manager-policy=static` for exclusive core pinning.
* **cpu-stats-exporter**: DaemonSet applied at the end of `create.sh`.
