# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 8 | — | 1 | 8 | — | — | — |
| plain | backend2 | 8 | — | 1 | 8 | — | — | — |
| plain | frontend | 12 | — | 1 | 12 | — | — | — |
| plain | shared | 10 | — | 1 | 10 | — | — | — |
| p2c | backend1 | 1 | — | 8 | 1 | — | — | — |
| p2c | backend2 | 1 | — | 8 | 1 | — | — | — |
| p2c | frontend | 3 | — | 4 | 3 | — | — | — |
| p2c | shared | 1 | — | 10 | 1 | — | — | — |
| p2c | ingress | — | 4 | 1 | — | — | — | — |
| wrr | backend1 | 1 | — | 8 | 1 | — | — | — |
| wrr | backend2 | 1 | — | 8 | 1 | — | — | — |
| wrr | frontend | 3 | — | 4 | 3 | — | — | — |
| wrr | shared | 1 | — | 10 | 1 | — | — | — |
| wrr | ingress | — | 4 | 1 | — | — | — | — |
| sidecar | backend1 | 8 | 2 | 1 | 8 | 8 | 1.0 | 1 |
| sidecar | backend2 | 8 | 2 | 1 | 8 | 8 | 1.0 | 1 |
| sidecar | frontend | 12 | 2 | 1 | 12 | 12 | 1.0 | 1 |
| sidecar | shared | 10 | 2 | 1 | 10 | 10 | 0.0 | 1 |
| sidecar | ingress | — | 4 | 1 | — | — | — | — |
| approx | backend1 | 1 | 1 | 8 | 1 | 1 | 1.0 | 1 |
| approx | backend2 | 1 | 1 | 8 | 1 | 1 | 1.0 | 1 |
| approx | frontend | 3 | 1 | 4 | 3 | 3 | 1.0 | 1 |
| approx | shared | 1 | 1 | 10 | 1 | 1 | 0.0 | 1 |
| approx | ingress | — | 4 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 1 | 1 | 8 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend2 | 1 | 1 | 8 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | frontend | 3 | 1 | 4 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | shared | 1 | 1 | 10 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | ingress | — | 4 | 1 | — | — | — | — |
| approx-edf | backend1 | 1 | 1 | 8 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend2 | 1 | 1 | 8 | 1 | 1 | 1.0 | 1 |
| approx-edf | frontend | 3 | 1 | 4 | 3 | 3 | 1.0 | 1 |
| approx-edf | shared | 1 | 1 | 10 | 1 | 1 | 0.0 | 1 |
| approx-edf | ingress | — | 4 | 1 | — | — | — | — |
| envoy | backend1 | 8 | 1 | 1 | 8 | — | — | — |
| envoy | backend2 | 8 | 1 | 1 | 8 | — | — | — |
| envoy | frontend | 12 | 1 | 1 | 12 | — | — | — |
| envoy | shared | 10 | 1 | 1 | 10 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 8 | — | 1 | 8 | — | — | — |
| rajomon | backend2 | 8 | — | 1 | 8 | — | — | — |
| rajomon | frontend | 12 | — | 1 | 12 | — | — | — |
| rajomon | shared | 10 | — | 1 | 10 | — | — | — |
| rajomon | rajomon-client | 13 | — | 1 | 13 | — | — | — |
| rajomon-lb | backend1 | 1 | — | 8 | 1 | — | — | — |
| rajomon-lb | backend2 | 1 | — | 8 | 1 | — | — | — |
| rajomon-lb | frontend | 3 | — | 4 | 3 | — | — | — |
| rajomon-lb | shared | 1 | — | 10 | 1 | — | — | — |
| rajomon-lb | rajomon-client | 3 | — | 4 | 3 | — | — | — |
| dagor | backend1 | 8 | — | 1 | 8 | — | — | — |
| dagor | backend2 | 8 | — | 1 | 8 | — | — | — |
| dagor | frontend | 12 | — | 1 | 12 | — | — | — |
| dagor | shared | 10 | — | 1 | 10 | — | — | — |
| dagor | rajomon-client | 13 | — | 1 | 13 | — | — | — |
| dagor-lb | backend1 | 1 | — | 8 | 1 | — | — | — |
| dagor-lb | backend2 | 1 | — | 8 | 1 | — | — | — |
| dagor-lb | frontend | 3 | — | 4 | 3 | — | — | — |
| dagor-lb | shared | 1 | — | 10 | 1 | — | — | — |
| dagor-lb | rajomon-client | 3 | — | 4 | 3 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 38 | 0 | — |
| p2c | 38 | 4 | 9.5 |
| wrr | 38 | 4 | 9.5 |
| sidecar | 38 | 12 | 3.17 |
| approx | 38 | 34 | 1.12 |
| approx-fcfs | 38 | 34 | 1.12 |
| approx-edf | 38 | 34 | 1.12 |
| envoy | 38 | 5 | 7.6 |
| rajomon | 51 | 0 | — |
| rajomon-lb | 50 | 0 | — |
| dagor | 51 | 0 | — |
| dagor-lb | 50 | 0 | — |
