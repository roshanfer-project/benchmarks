# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 1 | — | 1 | 1 | — | — | — |
| plain | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain | backend3 | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend1 | 1 | — | 1 | 1 | — | — | — |
| p2c | backend2 | 2 | — | 1 | 2 | — | — | — |
| p2c | backend3 | 3 | — | 1 | 3 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 6 | 1 | — | — | — | — |
| wrr | backend1 | 1 | — | 1 | 1 | — | — | — |
| wrr | backend2 | 2 | — | 1 | 2 | — | — | — |
| wrr | backend3 | 3 | — | 1 | 3 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 6 | 1 | — | — | — | — |
| sidecar | backend1 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| sidecar | backend2 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | backend3 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | ingress | — | 6 | 1 | — | — | — | — |
| approx | backend1 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| approx | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx | backend3 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx | ingress | — | 6 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | backend3 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | ingress | — | 6 | 1 | — | — | — | — |
| approx-edf | backend1 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | backend3 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | ingress | — | 6 | 1 | — | — | — | — |
| envoy | backend1 | 1 | 1 | 1 | 1 | — | — | — |
| envoy | backend2 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | backend3 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 1 | — | 1 | 1 | — | — | — |
| rajomon | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon | backend3 | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 1 | — | 1 | 1 | — | — | — |
| rajomon-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | backend3 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 1 | — | 1 | 1 | — | — | — |
| dagor | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor | backend3 | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 1 | — | 1 | 1 | — | — | — |
| dagor-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | backend3 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 8 | 0 | — |
| p2c | 8 | 6 | 1.33 |
| wrr | 8 | 6 | 1.33 |
| sidecar | 8 | 14 | 0.571 |
| approx | 8 | 10 | 0.8 |
| approx-fcfs | 8 | 10 | 0.8 |
| approx-edf | 8 | 10 | 0.8 |
| envoy | 8 | 5 | 1.6 |
| rajomon | 11 | 0 | — |
| rajomon-lb | 10 | 0 | — |
| dagor | 11 | 0 | — |
| dagor-lb | 10 | 0 | — |
