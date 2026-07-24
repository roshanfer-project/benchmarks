# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain | shared | 4 | — | 1 | 4 | — | — | — |
| plain-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| plain-lb | backend2 | 1 | — | 3 | 1 | — | — | — |
| plain-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| plain-lb | shared | 1 | — | 4 | 1 | — | — | — |
| plain-lb | ingress | — | 4 | 1 | — | — | — | — |
| sidecar | backend1 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | backend2 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | shared | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar | ingress | — | 4 | 1 | — | — | — | — |
| approx | backend1 | 1 | 0.5 | 3 | 1 | 1 | 1.0 | 1 |
| approx | backend2 | 1 | 0.5 | 3 | 1 | 1 | 1.0 | 1 |
| approx | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| approx | shared | 1 | 0.5 | 4 | 1 | 1 | 0.0 | 1 |
| approx | ingress | — | 4 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 1 | 0.5 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend2 | 1 | 0.5 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | shared | 1 | 0.5 | 4 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | ingress | — | 4 | 1 | — | — | — | — |
| approx-edf | backend1 | 1 | 0.5 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend2 | 1 | 0.5 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | shared | 1 | 0.5 | 4 | 1 | 1 | 0.0 | 1 |
| approx-edf | ingress | — | 4 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | shared | 4 | 1 | 1 | 4 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | shared | 4 | — | 1 | 4 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | backend2 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | shared | 1 | — | 4 | 1 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | shared | 4 | — | 1 | 4 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | backend2 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | shared | 1 | — | 4 | 1 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 12 | 0 | — |
| plain-lb | 12 | 4 | 3 |
| sidecar | 12 | 12 | 1 |
| approx | 12 | 9.5 | 1.26 |
| approx-fcfs | 12 | 9.5 | 1.26 |
| approx-edf | 12 | 9.5 | 1.26 |
| envoy | 12 | 5 | 2.4 |
| rajomon | 15 | 0 | — |
| rajomon-lb | 14 | 0 | — |
| dagor | 15 | 0 | — |
| dagor-lb | 14 | 0 | — |
