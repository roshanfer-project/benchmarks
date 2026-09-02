# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 2 | — | 1 | 2 | — | — | — |
| plain | backend2 | 2 | — | 1 | 2 | — | — | — |
| plain | frontend | 3 | — | 1 | 3 | — | — | — |
| p2c | backend1 | 2 | — | 1 | 2 | — | — | — |
| p2c | backend2 | 2 | — | 1 | 2 | — | — | — |
| p2c | frontend | 3 | — | 1 | 3 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend1 | 2 | — | 1 | 2 | — | — | — |
| wrr | backend2 | 2 | — | 1 | 2 | — | — | — |
| wrr | frontend | 3 | — | 1 | 3 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | backend1 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| roshanfer | backend2 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| roshanfer | frontend | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue | backend1 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-fcfs | backend1 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-edf | backend1 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | backend2 | 2 | 1 | 1 | 2 | — | — | — |
| envoy | frontend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 2 | — | 1 | 2 | — | — | — |
| rajomon | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor | backend1 | 2 | — | 1 | 2 | — | — | — |
| dagor | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | backend1 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | backend2 | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 7 | 0 | — |
| p2c | 7 | 2 | 3.5 |
| wrr | 7 | 2 | 3.5 |
| roshanfer | 7 | 8 | 0.875 |
| amphiqueue | 7 | 5 | 1.4 |
| amphiqueue-fcfs | 7 | 5 | 1.4 |
| amphiqueue-edf | 7 | 5 | 1.4 |
| envoy | 7 | 4 | 1.75 |
| rajomon | 11 | 0 | — |
| rajomon-lb | 10 | 0 | — |
| dagor | 11 | 0 | — |
| dagor-lb | 10 | 0 | — |
