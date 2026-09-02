# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | backend3 | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend1 | 1 | — | 3 | 1 | — | — | — |
| p2c | backend2 | 1 | — | 3 | 1 | — | — | — |
| p2c | backend3 | 1 | — | 3 | 1 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend1 | 1 | — | 3 | 1 | — | — | — |
| wrr | backend2 | 1 | — | 3 | 1 | — | — | — |
| wrr | backend3 | 1 | — | 3 | 1 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | backend1 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | backend2 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | backend3 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| approx | backend1 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx | backend2 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx | backend3 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend2 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | backend3 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | backend1 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend2 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | backend3 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend3 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend3 | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | backend2 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | backend3 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend3 | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | backend2 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | backend3 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 11 | 0 | — |
| p2c | 11 | 2 | 5.5 |
| wrr | 11 | 2 | 5.5 |
| roshanfer | 11 | 10 | 1.1 |
| approx | 11 | 12 | 0.917 |
| approx-fcfs | 11 | 12 | 0.917 |
| approx-edf | 11 | 12 | 0.917 |
| envoy | 11 | 5 | 2.2 |
| rajomon | 14 | 0 | — |
| rajomon-lb | 13 | 0 | — |
| dagor | 14 | 0 | — |
| dagor-lb | 13 | 0 | — |
