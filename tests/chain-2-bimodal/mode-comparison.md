# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend | 3 | — | 1 | 3 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend | 3 | — | 1 | 3 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend | 3 | — | 1 | 3 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| roshanfer | backend | 3 | 2 | 1 | 3 | 3 | 0.0 | 1 |
| roshanfer | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| roshanfer | ingress | — | 2 | 1 | — | — | — | — |
| approx | backend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | backend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | backend | 3 | 1 | 1 | 3 | 3 | 1.0 | 1 |
| approx-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend | 3 | 1 | 1 | 3 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend | 3 | — | 1 | 3 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend | 3 | — | 1 | 3 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 5 | 0 | — |
| p2c | 5 | 2 | 2.5 |
| wrr | 5 | 2 | 2.5 |
| roshanfer | 5 | 6 | 0.833 |
| approx | 5 | 4 | 1.25 |
| approx-fcfs | 5 | 4 | 1.25 |
| approx-edf | 5 | 4 | 1.25 |
| envoy | 5 | 3 | 1.67 |
| rajomon | 8 | 0 | — |
| rajomon-lb | 7 | 0 | — |
| dagor | 8 | 0 | — |
| dagor-lb | 7 | 0 | — |
