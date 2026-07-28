# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | backend | 2 | — | 1 | 2 | — | — | — |
| plain | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | backend | 1 | — | 2 | 1 | — | — | — |
| p2c | frontend | 2 | — | 1 | 2 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | backend | 1 | — | 2 | 1 | — | — | — |
| wrr | frontend | 2 | — | 1 | 2 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | backend | 2 | 2 | 1 | 2 | 2 | 0.0 | 1 |
| sidecar | frontend | 2 | 2 | 1 | 2 | 2 | 1.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| approx | backend | 1 | 1 | 2 | 1 | 1 | 0.0 | 1 |
| approx | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | backend | 1 | 1 | 2 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | backend | 1 | 1 | 2 | 1 | 1 | 0.0 | 1 |
| approx-edf | frontend | 2 | 1 | 1 | 2 | 2 | 1.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | backend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | frontend | 2 | 1 | 1 | 2 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | backend | 2 | — | 1 | 2 | — | — | — |
| rajomon | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| rajomon-lb | backend | 1 | — | 2 | 1 | — | — | — |
| rajomon-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| rajomon-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |
| dagor | backend | 2 | — | 1 | 2 | — | — | — |
| dagor | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor | rajomon-client | 3 | — | 1 | 3 | — | — | — |
| dagor-lb | backend | 1 | — | 2 | 1 | — | — | — |
| dagor-lb | frontend | 2 | — | 1 | 2 | — | — | — |
| dagor-lb | rajomon-client | 2 | — | 1 | 2 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 4 | 0 | — |
| p2c | 4 | 2 | 2 |
| wrr | 4 | 2 | 2 |
| sidecar | 4 | 6 | 0.667 |
| approx | 4 | 5 | 0.8 |
| approx-fcfs | 4 | 5 | 0.8 |
| approx-edf | 4 | 5 | 0.8 |
| envoy | 4 | 3 | 1.33 |
| rajomon | 7 | 0 | — |
| rajomon-lb | 6 | 0 | — |
| dagor | 7 | 0 | — |
| dagor-lb | 6 | 0 | — |
