# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 2 | — | 1 | 2 | — | — | — |
| plain | backend2 | 1 | — | 1 | 1 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend1 | 2 | — | 1 | 2 | — | — | — |
| p2c | backend2 | 1 | — | 1 | 1 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend1 | 2 | — | 1 | 2 | — | — | — |
| wrr | backend2 | 1 | — | 1 | 1 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | backend1 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| roshanfer | backend2 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| roshanfer | frontend | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue | backend1 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | backend2 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| amphiqueue | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-fcfs | backend1 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | backend2 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| amphiqueue-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-edf | backend1 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | backend2 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| amphiqueue-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | ingress | — | 2 | 1 | — | — | — | — |
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
| p2c | 5 | 2 | 2.5 |
| wrr | 5 | 2 | 2.5 |
| roshanfer | 5 | 8 | 0.625 |
| amphiqueue | 5 | 5 | 1 |
| amphiqueue-fcfs | 5 | 5 | 1 |
| amphiqueue-edf | 5 | 5 | 1 |
| envoy | 5 | 4 | 1.25 |
| rajomon | 8 | 0 | — |
| rajomon-lb | 7 | 0 | — |
| dagor | 8 | 0 | — |
| dagor-lb | 7 | 0 | — |
