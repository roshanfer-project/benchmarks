# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| plain | shared | 5 | — | 1 | 5 | — | — | — |
| p2c | backend1 | 3 | — | 1 | 3 | — | — | — |
| p2c | backend2 | 3 | — | 1 | 3 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | shared | 5 | — | 1 | 5 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend1 | 3 | — | 1 | 3 | — | — | — |
| wrr | backend2 | 3 | — | 1 | 3 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | shared | 5 | — | 1 | 5 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend1 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | backend2 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | shared | 5 | 2 | 1 | 5 | 5 | 0.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| approx | backend1 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| approx | backend2 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| approx | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| approx | shared | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | backend2 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | shared | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | backend1 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| approx-edf | backend2 | 3 | 0.5 | 1 | 3 | 3 | 1.0 | 1 |
| approx-edf | frontend | 2 | 0.5 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | shared | 5 | 0.5 | 1 | 5 | 5 | 0.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
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
| p2c | 13 | 2 | 6.5 |
| wrr | 13 | 2 | 6.5 |
| sidecar | 13 | 10 | 1.3 |
| approx | 13 | 4 | 3.25 |
| approx-fcfs | 13 | 4 | 3.25 |
| approx-edf | 13 | 4 | 3.25 |
| envoy | 13 | 5 | 2.6 |
| rajomon | 16 | 0 | — |
| rajomon-lb | 15 | 0 | — |
| dagor | 16 | 0 | — |
| dagor-lb | 15 | 0 | — |
