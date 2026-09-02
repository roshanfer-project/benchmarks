# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend1 | 3 | — | 1 | 3 | — | — | — |
| p2c | backend2 | 3 | — | 1 | 3 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend1 | 3 | — | 1 | 3 | — | — | — |
| wrr | backend2 | 3 | — | 1 | 3 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | backend1 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | backend2 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| approx | backend1 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx | backend2 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | backend1 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | backend2 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | backend1 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-edf | backend2 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 8 | 0 | — |
| p2c | 8 | 2 | 4 |
| wrr | 8 | 2 | 4 |
| roshanfer | 8 | 8 | 1 |
| approx | 8 | 5 | 1.6 |
| approx-fcfs | 8 | 5 | 1.6 |
| approx-edf | 8 | 5 | 1.6 |
| envoy | 8 | 4 | 2 |
| rajomon | 11 | 0 | — |
| rajomon-lb | 10 | 0 | — |
| dagor | 11 | 0 | — |
| dagor-lb | 10 | 0 | — |
