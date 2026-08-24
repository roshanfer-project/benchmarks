# Benchmarks

Service graphs and cluster scripts. Nested submodule `sidecar/` is the Roshanfer C++ sidecar.

Generated benches (`tests/*`, `alibaba-large/`) share `callgraph.json`, `services/`, `k8s/`, `build.sh`, and `deploy.sh`. `hotel/` and `social/` are hand-written DeathStarBench apps.

## Repository layout

```text
benchmarks/
├── hotel/                     DeathStarBench Hotel Reservation
├── social/                    DeathStarBench Social Network
├── alibaba-large/             Alibaba / DGG 30-MS graph
├── tests/                     synthetic graphs (incl. tutorial one-service)
├── sidecar/                   nested submodule: Roshanfer C++ sidecar
├── callgraph-framework/       callgraph.json → Go services + K8s manifests
├── k8s/                       K3s + Cilium create / reset / delete
└── provisioning/              SSH host bootstrap
```
