# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | frontend | 5 | — | 1 | 5 | — | — | — |
| plain-lb | frontend | 5 | — | 1 | 5 | — | — | — |
| plain-lb | ingress | — | 4 | 1 | — | — | — | — |
| sidecar | frontend | 5 | 2 | 1 | 5 | 5 | 0.0 | 1 |
| sidecar | ingress | — | 4 | 1 | — | — | — | — |
| approx | frontend | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| approx | ingress | — | 4 | 1 | — | — | — | — |
| approx-fcfs | frontend | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| approx-fcfs | ingress | — | 4 | 1 | — | — | — | — |
| approx-edf | frontend | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| approx-edf | ingress | — | 4 | 1 | — | — | — | — |
| envoy | frontend | 5 | 1 | 1 | 5 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | frontend | 5 | — | 1 | 5 | — | — | — |
| rajomon | rajomon-client | 6 | — | 1 | 6 | — | — | — |
| rajomon-lb | frontend | 5 | — | 1 | 5 | — | — | — |
| rajomon-lb | rajomon-client | 5 | — | 1 | 5 | — | — | — |
| dagor | frontend | 5 | — | 1 | 5 | — | — | — |
| dagor | rajomon-client | 6 | — | 1 | 6 | — | — | — |
| dagor-lb | frontend | 5 | — | 1 | 5 | — | — | — |
| dagor-lb | rajomon-client | 5 | — | 1 | 5 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 5 | 0 | — |
| plain-lb | 5 | 4 | 1.25 |
| sidecar | 5 | 6 | 0.833 |
| approx | 5 | 4.5 | 1.11 |
| approx-fcfs | 5 | 4.5 | 1.11 |
| approx-edf | 5 | 4.5 | 1.11 |
| envoy | 5 | 2 | 2.5 |
| rajomon | 11 | 0 | — |
| rajomon-lb | 10 | 0 | — |
| dagor | 11 | 0 | — |
| dagor-lb | 10 | 0 | — |
