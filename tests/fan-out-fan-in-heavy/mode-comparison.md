# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain | shared | 5 | — | 1 | 5 | — | — | — |
| plain-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | shared | 5 | — | 1 | 5 | — | — | — |
| plain-lb | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend1 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | backend2 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | shared | 5 | 2 | 1 | 5 | 5 | 0.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| sidecar-lb | backend1 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar-lb | backend2 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar-lb | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar-lb | shared | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| sidecar-lb | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | shared | 5 | 1 | 1 | 5 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | shared | 5 | — | 1 | 5 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | shared | 5 | — | 1 | 5 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | shared | 5 | — | 1 | 5 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | shared | 5 | — | 1 | 5 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 13 | 0 | — |
| plain-lb | 13 | 2 | 6.5 |
| sidecar | 13 | 10 | 1.3 |
| sidecar-lb | 13 | 4 | 3.25 |
| envoy | 13 | 5 | 2.6 |
| rajomon | 16 | 0 | — |
| rajomon-lb | 15 | 0 | — |
| dagor | 16 | 0 | — |
| dagor-lb | 15 | 0 | — |
