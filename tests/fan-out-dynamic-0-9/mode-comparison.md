# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 2 | — | 1 | 2 | — | — | — |
| plain | backend2 | 1 | — | 1 | 1 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend2 | 1 | — | 1 | 1 | — | — | — |
| plain-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| sidecar | backend1 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | backend2 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| sidecar-lb | backend1 | 2 | 1 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar-lb | backend2 | 1 | 1 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar-lb | frontend | 2 | 1 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar-lb | ingress | — | 1 | 1 | — | — | — | — |
| envoy | backend1 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | backend2 | 1 | 1 | 1 | 1 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 2 | — | 1 | 2 | — | — | — |
| rajomon | backend2 | 1 | — | 1 | 1 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | backend2 | 1 | — | 1 | 1 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 2 | — | 1 | 2 | — | — | — |
| dagor | backend2 | 1 | — | 1 | 1 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | backend2 | 1 | — | 1 | 1 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 5 | 0 | — |
| plain-lb | 5 | 0 | — |
| sidecar | 5 | 8 | 0.625 |
| sidecar-lb | 5 | 4 | 1.25 |
| envoy | 5 | 4 | 1.25 |
| rajomon | 8 | 0 | — |
| rajomon-lb | 7 | 0 | — |
| dagor | 8 | 0 | — |
| dagor-lb | 7 | 0 | — |
