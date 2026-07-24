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
| plain-lb | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend1 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | backend2 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| approx | backend1 | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| approx | backend2 | 1 | 0.5 | 1 | 1 | 1 | 0.0 | 1 |
| approx | frontend | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| approx-fcfs | backend2 | 1 | 0.5 | 1 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | frontend | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | backend1 | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| approx-edf | backend2 | 1 | 0.5 | 1 | 1 | 1 | 0.0 | 1 |
| approx-edf | frontend | 2 | 0.5 | 1 | 2 | 2 | 0.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
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
| plain-lb | 5 | 2 | 2.5 |
| sidecar | 5 | 8 | 0.625 |
| approx | 5 | 3.5 | 1.43 |
| approx-fcfs | 5 | 3.5 | 1.43 |
| approx-edf | 5 | 3.5 | 1.43 |
| envoy | 5 | 4 | 1.25 |
| rajomon | 8 | 0 | — |
| rajomon-lb | 7 | 0 | — |
| dagor | 8 | 0 | — |
| dagor-lb | 7 | 0 | — |
