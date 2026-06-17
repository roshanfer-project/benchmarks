# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend | 2 | — | 1 | 2 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend | 1 | — | 2 | 1 | — | — | — |
| plain-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| sidecar | backend | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| sidecar-lb | backend | 1 | 1 | 2 | 1 | 1 | 0.0 | 1 |
| sidecar-lb | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar-lb | ingress | — | 1 | 1 | — | — | — | — |
| envoy | backend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend | 2 | — | 1 | 2 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend | 1 | — | 2 | 1 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend | 2 | — | 1 | 2 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend | 1 | — | 2 | 1 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 4 | 0 | — |
| plain-lb | 4 | 0 | — |
| sidecar | 4 | 6 | 0.667 |
| sidecar-lb | 4 | 4 | 1 |
| envoy | 4 | 3 | 1.33 |
| rajomon | 7 | 0 | — |
| rajomon-lb | 6 | 0 | — |
| dagor | 7 | 0 | — |
| dagor-lb | 6 | 0 | — |
