# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | frontend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | ingress | — | 6 | 1 | — | — | — | — |
| sidecar | frontend | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | ingress | — | 6 | 1 | — | — | — | — |
| sidecar-lb | frontend | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | ingress | — | 6 | 1 | — | — | — | — |
| envoy | frontend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 3 | 0 | — |
| plain-lb | 3 | 6 | 0.5 |
| sidecar | 3 | 8 | 0.375 |
| sidecar-lb | 3 | 6.5 | 0.462 |
| envoy | 3 | 2 | 1.5 |
| rajomon | 7 | 0 | — |
| rajomon-lb | 6 | 0 | — |
| dagor | 7 | 0 | — |
| dagor-lb | 6 | 0 | — |
