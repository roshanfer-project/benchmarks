# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 2 | — | 1 | 2 | — | — | — |
| plain | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain | frontend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | ingress | — | 6 | 1 | — | — | — | — |
| sidecar | backend1 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | backend2 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | frontend | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | ingress | — | 6 | 1 | — | — | — | — |
| sidecar-lb | backend1 | 2 | 1 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar-lb | backend2 | 2 | 1 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar-lb | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar-lb | ingress | — | 6 | 1 | — | — | — | — |
| envoy | backend1 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | backend2 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | frontend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 2 | — | 1 | 2 | — | — | — |
| rajomon | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor | backend1 | 2 | — | 1 | 2 | — | — | — |
| dagor | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 7 | 0 | — |
| plain-lb | 7 | 6 | 1.17 |
| sidecar | 7 | 12 | 0.583 |
| sidecar-lb | 7 | 9 | 0.778 |
| envoy | 7 | 4 | 1.75 |
| rajomon | 11 | 0 | — |
| rajomon-lb | 10 | 0 | — |
| dagor | 11 | 0 | — |
| dagor-lb | 10 | 0 | — |
