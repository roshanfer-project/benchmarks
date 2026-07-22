# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 5 | — | 1 | 5 | — | — | — |
| plain | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain | backend3 | 3 | — | 1 | 3 | — | — | — |
| plain | backend4 | 3 | — | 1 | 3 | — | — | — |
| plain | backend5 | 1 | — | 1 | 1 | — | — | — |
| plain | backend6 | 1 | — | 1 | 1 | — | — | — |
| plain | backend7 | 1 | — | 1 | 1 | — | — | — |
| plain | backend8 | 2 | — | 1 | 2 | — | — | — |
| plain | frontend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend1 | 5 | — | 1 | 5 | — | — | — |
| plain-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain-lb | backend3 | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend4 | 3 | — | 1 | 3 | — | — | — |
| plain-lb | backend5 | 1 | — | 1 | 1 | — | — | — |
| plain-lb | backend6 | 1 | — | 1 | 1 | — | — | — |
| plain-lb | backend7 | 1 | — | 1 | 1 | — | — | — |
| plain-lb | backend8 | 2 | — | 1 | 2 | — | — | — |
| plain-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| plain-lb | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend1 | 5 | 2 | 1 | 5 | 5 | 0.0 | 1 |
| sidecar | backend2 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | backend3 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | backend4 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | backend5 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar | backend6 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar | backend7 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar | backend8 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | frontend | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| sidecar-lb | backend1 | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| sidecar-lb | backend2 | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar-lb | backend3 | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | backend4 | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | backend5 | 1 | 0.5 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar-lb | backend6 | 1 | 0.5 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar-lb | backend7 | 1 | 0.5 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar-lb | backend8 | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar-lb | frontend | 3 | 0.5 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar-lb | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 5 | 1 | 1 | 5 | — | — | — |
| envoy | backend2 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | backend3 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend4 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend5 | 1 | 1 | 1 | 1 | — | — | — |
| envoy | backend6 | 1 | 1 | 1 | 1 | — | — | — |
| envoy | backend7 | 1 | 1 | 1 | 1 | — | — | — |
| envoy | backend8 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | frontend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 5 | — | 1 | 5 | — | — | — |
| rajomon | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon | backend3 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend4 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend5 | 1 | — | 1 | 1 | — | — | — |
| rajomon | backend6 | 1 | — | 1 | 1 | — | — | — |
| rajomon | backend7 | 1 | — | 1 | 1 | — | — | — |
| rajomon | backend8 | 2 | — | 1 | 2 | — | — | — |
| rajomon | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | backend1 | 5 | — | 1 | 5 | — | — | — |
| rajomon-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | backend3 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend4 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend5 | 1 | — | 1 | 1 | — | — | — |
| rajomon-lb | backend6 | 1 | — | 1 | 1 | — | — | — |
| rajomon-lb | backend7 | 1 | — | 1 | 1 | — | — | — |
| rajomon-lb | backend8 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor | backend1 | 5 | — | 1 | 5 | — | — | — |
| dagor | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor | backend3 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend4 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend5 | 1 | — | 1 | 1 | — | — | — |
| dagor | backend6 | 1 | — | 1 | 1 | — | — | — |
| dagor | backend7 | 1 | — | 1 | 1 | — | — | — |
| dagor | backend8 | 2 | — | 1 | 2 | — | — | — |
| dagor | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | backend1 | 5 | — | 1 | 5 | — | — | — |
| dagor-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | backend3 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend4 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend5 | 1 | — | 1 | 1 | — | — | — |
| dagor-lb | backend6 | 1 | — | 1 | 1 | — | — | — |
| dagor-lb | backend7 | 1 | — | 1 | 1 | — | — | — |
| dagor-lb | backend8 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 21 | 0 | — |
| plain-lb | 21 | 2 | 10.5 |
| sidecar | 21 | 20 | 1.05 |
| sidecar-lb | 21 | 6.5 | 3.23 |
| envoy | 21 | 10 | 2.1 |
| rajomon | 25 | 0 | — |
| rajomon-lb | 24 | 0 | — |
| dagor | 25 | 0 | — |
| dagor-lb | 24 | 0 | — |
