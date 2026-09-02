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
| roshanfer | backend1 | 1 | 2 | 1 | 1 | 1 | 0.0 | 1 |
| roshanfer | backend2 | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| roshanfer | backend3 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| roshanfer | ingress | — | 6 | 1 | — | — | — | — |
| amphiqueue | backend1 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| amphiqueue | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | backend3 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | ingress | — | 6 | 1 | — | — | — | — |
| amphiqueue-fcfs | backend1 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| amphiqueue-fcfs | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | backend3 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | ingress | — | 6 | 1 | — | — | — | — |
| amphiqueue-edf | backend1 | 1 | 1 | 1 | 1 | 1 | 1.0 | 1 |
| amphiqueue-edf | backend2 | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | backend3 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | ingress | — | 6 | 1 | — | — | — | — |
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
| roshanfer | 8 | 14 | 0.571 |
| amphiqueue | 8 | 10 | 0.8 |
| amphiqueue-fcfs | 8 | 10 | 0.8 |
| amphiqueue-edf | 8 | 10 | 0.8 |
| envoy | 8 | 5 | 1.6 |
| rajomon | 11 | 0 | — |
| rajomon-lb | 10 | 0 | — |
| dagor | 11 | 0 | — |
| dagor-lb | 10 | 0 | — |
