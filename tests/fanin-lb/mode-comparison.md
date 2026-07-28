# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain | shared | 4 | — | 1 | 4 | — | — | — |
| p2c | backend1 | 1 | — | 3 | 1 | — | — | — |
| p2c | backend2 | 1 | — | 2 | 1 | — | — | — |
| p2c | frontend | 1 | — | 2 | 1 | — | — | — |
| p2c | shared | 1 | — | 4 | 1 | — | — | — |
| p2c | ingress | — | 4 | 1 | — | — | — | — |
| wrr | backend1 | 1 | — | 3 | 1 | — | — | — |
| wrr | backend2 | 1 | — | 2 | 1 | — | — | — |
| wrr | frontend | 1 | — | 2 | 1 | — | — | — |
| wrr | shared | 1 | — | 4 | 1 | — | — | — |
| wrr | ingress | — | 4 | 1 | — | — | — | — |
| sidecar | backend1 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | backend2 | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | shared | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar | ingress | — | 4 | 1 | — | — | — | — |
| approx | backend1 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx | backend2 | 1 | 1 | 2 | 1 | 1 | 1.0 | 1 |
| approx | frontend | 1 | 1 | 2 | 1 | 1 | 1.0 | 1 |
| approx | shared | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx | ingress | — | 4 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend2 | 1 | 1 | 2 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | frontend | 1 | 1 | 2 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | shared | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | ingress | — | 4 | 1 | — | — | — | — |
| approx-edf | backend1 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend2 | 1 | 1 | 2 | 1 | 1 | 1.0 | 1 |
| approx-edf | frontend | 1 | 1 | 2 | 1 | 1 | 1.0 | 1 |
| approx-edf | shared | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx-edf | ingress | — | 4 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | shared | 4 | 1 | 1 | 4 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | shared | 4 | — | 1 | 4 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | backend2 | 1 | — | 2 | 1 | — | — | — |
| rajomon-lb | frontend | 1 | — | 2 | 1 | — | — | — |
| rajomon-lb | shared | 1 | — | 4 | 1 | — | — | — |
| rajomon-lb | rajomon-client | 1 | — | 2 | 1 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | shared | 4 | — | 1 | 4 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | backend2 | 1 | — | 2 | 1 | — | — | — |
| dagor-lb | frontend | 1 | — | 2 | 1 | — | — | — |
| dagor-lb | shared | 1 | — | 4 | 1 | — | — | — |
| dagor-lb | rajomon-client | 1 | — | 2 | 1 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 11 | 0 | — |
| p2c | 11 | 4 | 2.75 |
| wrr | 11 | 4 | 2.75 |
| sidecar | 11 | 12 | 0.917 |
| approx | 11 | 15 | 0.733 |
| approx-fcfs | 11 | 15 | 0.733 |
| approx-edf | 11 | 15 | 0.733 |
| envoy | 11 | 5 | 2.2 |
| rajomon | 14 | 0 | — |
| rajomon-lb | 13 | 0 | — |
| dagor | 14 | 0 | — |
| dagor-lb | 13 | 0 | — |
