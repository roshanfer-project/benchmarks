# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend1 | 3 | — | 1 | 3 | — | — | — |
| plain | backend2 | 3 | — | 1 | 3 | — | — | — |
| plain | backend3 | 4 | — | 1 | 4 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend1 | 3 | — | 1 | 3 | — | — | — |
| p2c | backend2 | 3 | — | 1 | 3 | — | — | — |
| p2c | backend3 | 4 | — | 1 | 4 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend1 | 3 | — | 1 | 3 | — | — | — |
| wrr | backend2 | 3 | — | 1 | 3 | — | — | — |
| wrr | backend3 | 4 | — | 1 | 4 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | backend1 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | backend2 | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | backend3 | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| roshanfer | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue | backend1 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue | backend2 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue | backend3 | 4 | 1 | 1 | 4 | 4 | 1.0 | 1 |
| amphiqueue | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-fcfs | backend1 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-fcfs | backend2 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-fcfs | backend3 | 4 | 1 | 1 | 4 | 4 | 1.0 | 1 |
| amphiqueue-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-edf | backend1 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-edf | backend2 | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-edf | backend3 | 4 | 1 | 1 | 4 | 4 | 1.0 | 1 |
| amphiqueue-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| amphiqueue-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend1 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend2 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | backend3 | 4 | 1 | 1 | 4 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon | backend3 | 4 | — | 1 | 4 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend3 | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor | backend3 | 4 | — | 1 | 4 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend1 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend2 | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend3 | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 12 | 0 | — |
| p2c | 12 | 2 | 6 |
| wrr | 12 | 2 | 6 |
| roshanfer | 12 | 10 | 1.2 |
| amphiqueue | 12 | 6 | 2 |
| amphiqueue-fcfs | 12 | 6 | 2 |
| amphiqueue-edf | 12 | 6 | 2 |
| envoy | 12 | 5 | 2.4 |
| rajomon | 15 | 0 | — |
| rajomon-lb | 14 | 0 | — |
| dagor | 15 | 0 | — |
| dagor-lb | 14 | 0 | — |
