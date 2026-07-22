# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| sidecar-lb | backend | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar-lb | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 5 | 0 | — |
| plain-lb | 5 | 2 | 2.5 |
| sidecar | 5 | 6 | 0.833 |
| sidecar-lb | 5 | 3 | 1.67 |
| envoy | 5 | 3 | 1.67 |
| rajomon | 8 | 0 | — |
| rajomon-lb | 7 | 0 | — |
| dagor | 8 | 0 | — |
| dagor-lb | 7 | 0 | — |
