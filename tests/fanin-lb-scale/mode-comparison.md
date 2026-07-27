# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 10 | — | 1 | 10 | — | — | — |
| plain | backend2 | 10 | — | 1 | 10 | — | — | — |
| plain | frontend | 10 | — | 1 | 10 | — | — | — |
| plain | shared | 10 | — | 1 | 10 | — | — | — |
| p2c | backend1 | 1 | — | 10 | 1 | — | — | — |
| p2c | backend2 | 1 | — | 10 | 1 | — | — | — |
| p2c | frontend | 1 | — | 10 | 1 | — | — | — |
| p2c | shared | 1 | — | 10 | 1 | — | — | — |
| p2c | ingress | — | 4 | 1 | — | — | — | — |
| wrr | backend1 | 1 | — | 10 | 1 | — | — | — |
| wrr | backend2 | 1 | — | 10 | 1 | — | — | — |
| wrr | frontend | 1 | — | 10 | 1 | — | — | — |
| wrr | shared | 1 | — | 10 | 1 | — | — | — |
| wrr | ingress | — | 4 | 1 | — | — | — | — |
| sidecar | backend1 | 10 | 2 | 1 | 10 | 10 | 1.0 | 1 |
| sidecar | backend2 | 10 | 2 | 1 | 10 | 10 | 1.0 | 1 |
| sidecar | frontend | 10 | 2 | 1 | 10 | 10 | 1.0 | 1 |
| sidecar | shared | 10 | 2 | 1 | 10 | 10 | 0.0 | 1 |
| sidecar | ingress | — | 4 | 1 | — | — | — | — |
| approx | backend1 | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx | backend2 | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx | frontend | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx | shared | 1 | 0.5 | 10 | 1 | 1 | 0.0 | 1 |
| approx | ingress | — | 4 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend2 | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | frontend | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | shared | 1 | 0.5 | 10 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | ingress | — | 4 | 1 | — | — | — | — |
| approx-edf | backend1 | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend2 | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx-edf | frontend | 1 | 0.5 | 10 | 1 | 1 | 1.0 | 1 |
| approx-edf | shared | 1 | 0.5 | 10 | 1 | 1 | 0.0 | 1 |
| approx-edf | ingress | — | 4 | 1 | — | — | — | — |
| envoy | backend1 | 10 | 1 | 1 | 10 | — | — | — |
| envoy | backend2 | 10 | 1 | 1 | 10 | — | — | — |
| envoy | frontend | 10 | 1 | 1 | 10 | — | — | — |
| envoy | shared | 10 | 1 | 1 | 10 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 10 | — | 1 | 10 | — | — | — |
| rajomon | backend2 | 10 | — | 1 | 10 | — | — | — |
| rajomon | frontend | 10 | — | 1 | 10 | — | — | — |
| rajomon | shared | 10 | — | 1 | 10 | — | — | — |
| rajomon | rajomon-client | 11 | — | 1 | 11 | — | — | — |
| rajomon-lb | backend1 | 1 | — | 10 | 1 | — | — | — |
| rajomon-lb | backend2 | 1 | — | 10 | 1 | — | — | — |
| rajomon-lb | frontend | 1 | — | 10 | 1 | — | — | — |
| rajomon-lb | shared | 1 | — | 10 | 1 | — | — | — |
| rajomon-lb | rajomon-client | 1 | — | 10 | 1 | — | — | — |
| dagor | backend1 | 10 | — | 1 | 10 | — | — | — |
| dagor | backend2 | 10 | — | 1 | 10 | — | — | — |
| dagor | frontend | 10 | — | 1 | 10 | — | — | — |
| dagor | shared | 10 | — | 1 | 10 | — | — | — |
| dagor | rajomon-client | 11 | — | 1 | 11 | — | — | — |
| dagor-lb | backend1 | 1 | — | 10 | 1 | — | — | — |
| dagor-lb | backend2 | 1 | — | 10 | 1 | — | — | — |
| dagor-lb | frontend | 1 | — | 10 | 1 | — | — | — |
| dagor-lb | shared | 1 | — | 10 | 1 | — | — | — |
| dagor-lb | rajomon-client | 1 | — | 10 | 1 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 40 | 0 | — |
| p2c | 40 | 4 | 10 |
| wrr | 40 | 4 | 10 |
| sidecar | 40 | 12 | 3.33 |
| approx | 40 | 24 | 1.67 |
| approx-fcfs | 40 | 24 | 1.67 |
| approx-edf | 40 | 24 | 1.67 |
| envoy | 40 | 5 | 8 |
| rajomon | 51 | 0 | — |
| rajomon-lb | 50 | 0 | — |
| dagor | 51 | 0 | — |
| dagor-lb | 50 | 0 | — |
