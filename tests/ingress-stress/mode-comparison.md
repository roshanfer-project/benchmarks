# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | frontend | 3 | — | 1 | 3 | — | — | — |
| p2c | frontend | 3 | — | 1 | 3 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | frontend | 3 | — | 1 | 3 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | frontend | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-fcfs | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| amphiqueue-edf | frontend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| amphiqueue-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | frontend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| rajomon-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor | rajomon-client | 4 | — | 1 | 4 | — | — | — |
| dagor-lb | frontend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | rajomon-client | 3 | — | 1 | 3 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 3 | 0 | — |
| p2c | 3 | 2 | 1.5 |
| wrr | 3 | 2 | 1.5 |
| roshanfer | 3 | 4 | 0.75 |
| amphiqueue | 3 | 3 | 1 |
| amphiqueue-fcfs | 3 | 3 | 1 |
| amphiqueue-edf | 3 | 3 | 1 |
| envoy | 3 | 2 | 1.5 |
| rajomon | 7 | 0 | — |
| rajomon-lb | 6 | 0 | — |
| dagor | 7 | 0 | — |
| dagor-lb | 6 | 0 | — |
