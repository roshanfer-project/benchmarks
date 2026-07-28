# Mode comparison

Deploy resource comparison across generated modes (from callgraph.json).

## Workloads

| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |
|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|
| plain | MS_25806 | 8 | — | 1 | 8 | — | — | — |
| plain | MS_2687 | 4 | — | 1 | 4 | — | — | — |
| plain | MS_40087 | 6 | — | 1 | 6 | — | — | — |
| plain | MS_44246 | 4 | — | 1 | 4 | — | — | — |
| plain | MS_51787 | 4 | — | 1 | 4 | — | — | — |
| plain | MS_56113 | 8 | — | 1 | 8 | — | — | — |
| plain | MS_64512 | 5 | — | 1 | 5 | — | — | — |
| plain | MS_70124 | 3 | — | 1 | 3 | — | — | — |
| p2c | MS_25806 | 1 | — | 8 | 1 | — | — | — |
| p2c | MS_2687 | 1 | — | 4 | 1 | — | — | — |
| p2c | MS_40087 | 1 | — | 6 | 1 | — | — | — |
| p2c | MS_44246 | 1 | — | 4 | 1 | — | — | — |
| p2c | MS_51787 | 1 | — | 4 | 1 | — | — | — |
| p2c | MS_56113 | 1 | — | 8 | 1 | — | — | — |
| p2c | MS_64512 | 1 | — | 5 | 1 | — | — | — |
| p2c | MS_70124 | 1 | — | 3 | 1 | — | — | — |
| p2c | ingress | — | 2 | 1 | — | — | — | — |
| wrr | MS_25806 | 1 | — | 8 | 1 | — | — | — |
| wrr | MS_2687 | 1 | — | 4 | 1 | — | — | — |
| wrr | MS_40087 | 1 | — | 6 | 1 | — | — | — |
| wrr | MS_44246 | 1 | — | 4 | 1 | — | — | — |
| wrr | MS_51787 | 1 | — | 4 | 1 | — | — | — |
| wrr | MS_56113 | 1 | — | 8 | 1 | — | — | — |
| wrr | MS_64512 | 1 | — | 5 | 1 | — | — | — |
| wrr | MS_70124 | 1 | — | 3 | 1 | — | — | — |
| wrr | ingress | — | 2 | 1 | — | — | — | — |
| sidecar | MS_25806 | 8 | 2 | 1 | 8 | 8 | 0.0 | 1 |
| sidecar | MS_2687 | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar | MS_40087 | 6 | 2 | 1 | 6 | 6 | 0.0 | 1 |
| sidecar | MS_44246 | 4 | 2 | 1 | 4 | 4 | 0.0 | 1 |
| sidecar | MS_51787 | 4 | 2 | 1 | 4 | 4 | 1.0 | 1 |
| sidecar | MS_56113 | 8 | 2 | 1 | 8 | 8 | 0.0 | 1 |
| sidecar | MS_64512 | 5 | 2 | 1 | 5 | 5 | 1.0 | 1 |
| sidecar | MS_70124 | 3 | 2 | 1 | 3 | 3 | 1.0 | 1 |
| sidecar | ingress | — | 2 | 1 | — | — | — | — |
| approx | MS_25806 | 1 | 1 | 8 | 1 | 1 | 0.0 | 1 |
| approx | MS_2687 | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx | MS_40087 | 1 | 1 | 6 | 1 | 1 | 0.0 | 1 |
| approx | MS_44246 | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx | MS_51787 | 1 | 1 | 4 | 1 | 1 | 1.0 | 1 |
| approx | MS_56113 | 1 | 1 | 8 | 1 | 1 | 0.0 | 1 |
| approx | MS_64512 | 1 | 1 | 5 | 1 | 1 | 1.0 | 1 |
| approx | MS_70124 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx | ingress | — | 2 | 1 | — | — | — | — |
| approx-fcfs | MS_25806 | 1 | 1 | 8 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | MS_2687 | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | MS_40087 | 1 | 1 | 6 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | MS_44246 | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | MS_51787 | 1 | 1 | 4 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | MS_56113 | 1 | 1 | 8 | 1 | 1 | 0.0 | 1 |
| approx-fcfs | MS_64512 | 1 | 1 | 5 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | MS_70124 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-fcfs | ingress | — | 2 | 1 | — | — | — | — |
| approx-edf | MS_25806 | 1 | 1 | 8 | 1 | 1 | 0.0 | 1 |
| approx-edf | MS_2687 | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx-edf | MS_40087 | 1 | 1 | 6 | 1 | 1 | 0.0 | 1 |
| approx-edf | MS_44246 | 1 | 1 | 4 | 1 | 1 | 0.0 | 1 |
| approx-edf | MS_51787 | 1 | 1 | 4 | 1 | 1 | 1.0 | 1 |
| approx-edf | MS_56113 | 1 | 1 | 8 | 1 | 1 | 0.0 | 1 |
| approx-edf | MS_64512 | 1 | 1 | 5 | 1 | 1 | 1.0 | 1 |
| approx-edf | MS_70124 | 1 | 1 | 3 | 1 | 1 | 1.0 | 1 |
| approx-edf | ingress | — | 2 | 1 | — | — | — | — |
| envoy | MS_25806 | 8 | 1 | 1 | 8 | — | — | — |
| envoy | MS_2687 | 4 | 1 | 1 | 4 | — | — | — |
| envoy | MS_40087 | 6 | 1 | 1 | 6 | — | — | — |
| envoy | MS_44246 | 4 | 1 | 1 | 4 | — | — | — |
| envoy | MS_51787 | 4 | 1 | 1 | 4 | — | — | — |
| envoy | MS_56113 | 8 | 1 | 1 | 8 | — | — | — |
| envoy | MS_64512 | 5 | 1 | 1 | 5 | — | — | — |
| envoy | MS_70124 | 3 | 1 | 1 | 3 | — | — | — |
| envoy | ingress | — | 1 | 1 | — | — | — | — |
| rajomon | MS_25806 | 8 | — | 1 | 8 | — | — | — |
| rajomon | MS_2687 | 4 | — | 1 | 4 | — | — | — |
| rajomon | MS_40087 | 6 | — | 1 | 6 | — | — | — |
| rajomon | MS_44246 | 4 | — | 1 | 4 | — | — | — |
| rajomon | MS_51787 | 4 | — | 1 | 4 | — | — | — |
| rajomon | MS_56113 | 8 | — | 1 | 8 | — | — | — |
| rajomon | MS_64512 | 5 | — | 1 | 5 | — | — | — |
| rajomon | MS_70124 | 3 | — | 1 | 3 | — | — | — |
| rajomon | rajomon-client | 6 | — | 1 | 6 | — | — | — |
| rajomon-lb | MS_25806 | 1 | — | 8 | 1 | — | — | — |
| rajomon-lb | MS_2687 | 1 | — | 4 | 1 | — | — | — |
| rajomon-lb | MS_40087 | 1 | — | 6 | 1 | — | — | — |
| rajomon-lb | MS_44246 | 1 | — | 4 | 1 | — | — | — |
| rajomon-lb | MS_51787 | 1 | — | 4 | 1 | — | — | — |
| rajomon-lb | MS_56113 | 1 | — | 8 | 1 | — | — | — |
| rajomon-lb | MS_64512 | 1 | — | 5 | 1 | — | — | — |
| rajomon-lb | MS_70124 | 1 | — | 3 | 1 | — | — | — |
| rajomon-lb | rajomon-client | 1 | — | 5 | 1 | — | — | — |
| dagor | MS_25806 | 8 | — | 1 | 8 | — | — | — |
| dagor | MS_2687 | 4 | — | 1 | 4 | — | — | — |
| dagor | MS_40087 | 6 | — | 1 | 6 | — | — | — |
| dagor | MS_44246 | 4 | — | 1 | 4 | — | — | — |
| dagor | MS_51787 | 4 | — | 1 | 4 | — | — | — |
| dagor | MS_56113 | 8 | — | 1 | 8 | — | — | — |
| dagor | MS_64512 | 5 | — | 1 | 5 | — | — | — |
| dagor | MS_70124 | 3 | — | 1 | 3 | — | — | — |
| dagor | rajomon-client | 6 | — | 1 | 6 | — | — | — |
| dagor-lb | MS_25806 | 1 | — | 8 | 1 | — | — | — |
| dagor-lb | MS_2687 | 1 | — | 4 | 1 | — | — | — |
| dagor-lb | MS_40087 | 1 | — | 6 | 1 | — | — | — |
| dagor-lb | MS_44246 | 1 | — | 4 | 1 | — | — | — |
| dagor-lb | MS_51787 | 1 | — | 4 | 1 | — | — | — |
| dagor-lb | MS_56113 | 1 | — | 8 | 1 | — | — | — |
| dagor-lb | MS_64512 | 1 | — | 5 | 1 | — | — | — |
| dagor-lb | MS_70124 | 1 | — | 3 | 1 | — | — | — |
| dagor-lb | rajomon-client | 1 | — | 5 | 1 | — | — | — |

## Cluster totals

| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |
|------|-----------------|---------------------|-------------------|
| plain | 42 | 0 | — |
| p2c | 42 | 2 | 21 |
| wrr | 42 | 2 | 21 |
| sidecar | 42 | 18 | 2.33 |
| approx | 42 | 44 | 0.955 |
| approx-fcfs | 42 | 44 | 0.955 |
| approx-edf | 42 | 44 | 0.955 |
| envoy | 42 | 9 | 4.67 |
| rajomon | 48 | 0 | — |
| rajomon-lb | 47 | 0 | — |
| dagor | 48 | 0 | — |
| dagor-lb | 47 | 0 | — |
