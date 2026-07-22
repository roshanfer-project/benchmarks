# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | backend3 | 4 | — | 1 | 4 | — | — | — |
| plain | backend4 | 4 | — | 1 | 4 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend3 | 4 | — | 1 | 4 | — | — | — |
| plain-lb | backend4 | 4 | — | 1 | 4 | — | — | — |
| plain-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend1 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | backend2 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | backend3 | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar | backend4 | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| sidecar-lb | backend1 | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | backend2 | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | backend3 | 4 | 0.5 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar-lb | backend4 | 4 | 0.5 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar-lb | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar-lb | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend3 | 4 | 1 | 1 | 4 | — | — | — |
| envoy | backend4 | 4 | 1 | 1 | 4 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend3 | 4 | — | 1 | 4 | — | — | — |
| rajomon | backend4 | 4 | — | 1 | 4 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend3 | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | backend4 | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend3 | 4 | — | 1 | 4 | — | — | — |
| dagor | backend4 | 4 | — | 1 | 4 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend3 | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | backend4 | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 16 | 0 | — |
| plain-lb | 16 | 2 | 8 |
| sidecar | 16 | 12 | 1.33 |
| sidecar-lb | 16 | 4.5 | 3.56 |
| envoy | 16 | 6 | 2.67 |
| rajomon | 19 | 0 | — |
| rajomon-lb | 18 | 0 | — |
| dagor | 19 | 0 | — |
| dagor-lb | 18 | 0 | — |
