# Call Graph Benchmark Framework

Generates runnable Go microservice benchmarks from call graph JSON.

## Usage

```bash
go run ./cmd/gen <callgraph.json> [-o output-dir]
```

Example:
```bash
go run ./cmd/gen ../alibaba-large/callgraph.json -o ../alibaba-large
```

Generates the benchmark, `callgraph.pdf`, and mode comparison tables (`mode-comparison.md`, `mode-comparison.csv`) in the output directory.

### Mode comparison tables

At codegen time, `cmd/gen` emits deploy-resource tables comparing all supported modes (`plain`, `p2c`, `wrr`, `sidecar`, `approx`, `approx-fcfs`, `approx-edf`, `envoy`, `rajomon`, `dagor`, and LB variants). Values use the same formulas as K8s manifest generation: per-service `app_cpu_limit`, `sidecar_cpu_limit`, `replicas`, `GOMAXPROCS`, and sidecar `cpu_count` / `over_commitment` / `num_threads`. A cluster-totals section sums app and sidecar cores per mode.

### Visualizer

Generate a PDF visualization of the callgraph:

```bash
go run ./cmd/viz <callgraph.json> [-o callgraph.pdf]
```

Requires [graphviz](https://graphviz.org/) (`dot` on PATH): `apt install graphviz` / `brew install graphviz`

**Service-level paper figure** (unlabeled circles, directed edges, ACM quarter-column PDF; microservices with weighted (dynamic) fan-out get distinct fill colors):

```bash
# From repo root (e.g. roshanfer-experiments): use .venv for matplotlib
go run ./cmd/viz -paper path/to/callgraph.json [-o callgraph-service.pdf]
```

With `-paper`, `viz` runs `render_service_pdf.py` using **the repository root `.venv` only** (`$REPO_ROOT/.venv/bin/python3`). Missing venv is an error with install hints. Create at repo root: `python3 -m venv .venv` then `.venv/bin/pip install -r requirements.txt` (see repo root [`requirements.txt`](../../requirements.txt)).

## Requirements

- Go 1.25+
- protoc with protoc-gen-go and protoc-gen-go-grpc plugins
- graphviz (for viz tool and `gen -v`; also used by `viz -paper` for layout)
- For `viz -paper`: Python **must** be repo root **`.venv`** with matplotlib (`pip install -r requirements.txt` from repo root)
- Docker with [Buildx](https://docs.docker.com/build/) enabled (`docker buildx` / `docker buildx bake`) for generated benchmark images

## Scripts

- `build.sh [tag]` — build the sidecar (if present), build all workload images with `docker buildx bake` (shared compile + parallel final stages), then push every image in parallel (`REGISTRY`, `TAG`, `BENCH` env vars behave as before).
- `deploy.sh [plain|p2c|wrr|sidecar|approx|approx-fcfs|approx-edf|rajomon|rajomon-lb|dagor|dagor-lb|envoy]` — deploy to K8s (uses `TAG`, `REGISTRY`). `./deploy.sh sidecar debug` or `./deploy.sh approx debug` (also `approx-fcfs` / `approx-edf`) enables workload debug (see below). `debug` is only valid with `sidecar` or approx modes, not with plain, p2c, wrr, rajomon, rajomon-lb, dagor, dagor-lb, or envoy. **envoy** uses `envoyproxy/envoy:v1.32-latest` sidecars (no SLO queuing, no `SIDECAR_OVER_COMMIT`). For **sidecar** and **approx\***: if **`SIDECAR_OVER_COMMIT`** is set (non-empty), every `over_commitment:` field in `sidecar-configs.yaml` / `${MODE}-configs.yaml` is rewritten to that value before apply (needs `perl` on PATH); omit it to keep values from `callgraph.json` codegen.
- `destroy.sh` — tear down
- `collect_logs.sh` — collect pod logs. If the environment variable `COLLECT_SIDECAR_NANOLOG=1` is set (done by `exec` when `--nanolog-debug` is enabled) and mode is `sidecar` or an approx mode, the script also `kubectl cp`s `/compressedLog` from each sidecar container into `$OUTPUT_DIR` as `*-sidecar.clog` (plus ingress as `*-ingress-sidecar.clog`). Decompression uses `benchmarks/sidecar/external/NanoLog/runtime/decompressor` from the repo checkout that runs the executor.

### Sidecar deploy debug

Use `./deploy.sh sidecar debug` or `./deploy.sh approx debug` after `./build.sh` as usual. Requires [mikefarah yq](https://github.com/mikefarah/yq) v4 on `PATH` (the script exits with an install hint if it is missing).

Debug mode only changes **workload** manifests under a temp copy before `kubectl apply`. For **sidecar**: `*-sidecar.yaml` and `ingress.yaml` (Pods) get `spec.restartPolicy: Never`. For **approx\***: `*-approx.yaml` Deployments get GLOG env on the sidecar container only (Deployment pod templates cannot use `restartPolicy: Never`); `ingress-lb.yaml` (Pod) gets the same restart + GLOG behavior as sidecar ingress. If you set `SIDECAR_GLOG_V` and/or `SIDECAR_GLOG_VMODULE` (or define them in `k8s/sidecar-debug-glog.env`), the `sidecar` container also gets `GLOG_v` / `GLOG_vmodule` (existing entries with those names are replaced). You can use debug mode for **restart behavior only** (sidecar / approx ingress) without setting any verbosity. **Prometheus** is still applied from `k8s/manifests/prometheus.yaml` exactly like non-debug deploy.

Set verbosity via environment (or optional file — see precedence below):

- `SIDECAR_GLOG_V` — maps to glog `GLOG_v` (e.g. `2` enables `VLOG(0)`–`VLOG(2)` globally where other filters allow).
- `SIDECAR_GLOG_VMODULE` — maps to `GLOG_vmodule`; patterns use **source file basenames** (no `.cc`), e.g. `state=2`, `connection=1`, `event_loop=3`.

Examples:

```bash
export SIDECAR_GLOG_V=2
./deploy.sh sidecar debug
```

```bash
export SIDECAR_GLOG_VMODULE=state=2,connection=1
./deploy.sh sidecar debug
```

```bash
export SIDECAR_GLOG_V=1
export SIDECAR_GLOG_VMODULE=event_loop=3
./deploy.sh sidecar debug
```

Optional `k8s/sidecar-debug-glog.env` under **this** benchmark’s `k8s/` (next to that benchmark’s `deploy.sh`). A file in another directory (e.g. `tests/one-service/k8s/...`) is not read when you deploy from `tests/fan-out-dynamic-0-9` — copy, symlink, or recreate it there. Same variable names as above, `#` comments allowed; script reads it only in the debug branch (both `sidecar debug` and `approx* debug`). **Precedence:** values already set in the environment win; the file fills `SIDECAR_GLOG_V` / `SIDECAR_GLOG_VMODULE` only when those variables are unset, so `SIDECAR_GLOG_V=3 ./deploy.sh sidecar debug` overrides a value from the file.

### Over-commitment

`over_commitment` is a sidecar admission parameter written into per-service snippets in `sidecar-configs.yaml` and `approx*-configs.yaml`. It must be in `[0, 1]` when set.

**Modes that use it:** **sidecar**, **approx**, **approx-fcfs**, **approx-edf**. Other modes (plain, p2c, wrr, envoy, rajomon, dagor, and LB variants) do not emit or apply `over_commitment`.

**Defaults when omitted from `callgraph.json`:**

| Mode | Default |
|------|---------|
| `sidecar` | `0` |
| `approx` / `approx-fcfs` / `approx-edf` | `1` |

An explicit `over_commitment` on a node applies to **both** sidecar and approx* ConfigMaps. Ingress config snippets typically do not include the field.

**Recommendation:** Prefer `over_commitment: 1` for microservices with service-time variability — exponential service times and/or multiple endpoints on the same service.

**Deploy override:** For latency-vs-throughput sweeps, set **`SIDECAR_OVER_COMMIT`** when deploying **sidecar** or **approx\*** (e.g. `0`, `0.2`, `1`). `deploy.sh` stages `sidecar-configs.yaml` or `${MODE}-configs.yaml`, replaces each embedded `over_commitment:` value with `SIDECAR_OVER_COMMIT`, then applies the patched manifest (**requires `perl` on PATH**). After patching, the script checks every `over_commitment:` line matches **`SIDECAR_OVER_COMMIT`** and exits non‑zero otherwise. Only snippets that already contain `over_commitment` are affected. The experiment executor forwards **`deploy_env`** into the deploy environment.

### Extra limit

`extra_limit` is an optional sidecar admission parameter (integer ≥ 0) written into per-service snippets in `sidecar-configs.yaml` and `approx*-configs.yaml`. It adds a fixed bump to per-endpoint / dynamic admission limits in the sidecar. Default when omitted: `0`. Applies to **sidecar**, **approx**, **approx-fcfs**, and **approx-edf** (same ConfigMap path as `over_commitment`). Ingress snippets typically do not include the field.

### approx modes

Sidecar admission control with **replication and sidecar-internal load balancing**. Shared proxy topology (`*-approx.yaml` Deployments, dedicated ingress pod, SLO/priority queuing). Env template: **`k8s/approx.env`** (`sidecar=true`). Three modes differ only in mesh late-binding YAML:

| Mode | Mesh config |
|------|-------------|
| `approx` | `mesh_late_binding: False` |
| `approx-fcfs` | `mesh_late_binding: True`, `late_binding_type: fcfs` |
| `approx-edf` | `mesh_late_binding: True`, `late_binding_type: edf` |

ConfigMaps: `approx-configs.yaml`, `approx-fcfs-configs.yaml`, `approx-edf-configs.yaml`. The `replicas` field in `callgraph.json` applies (Deployments, per-replica app CPU, headless internal Services). Workloads deploy **sequentially** in callgraph dependency order (leaves first, entry last), then ingress — same as **sidecar** — so downstream headless Services have ready endpoints before upstream sidecars resolve them at startup.

Load balancing across replicas uses the sidecar proxy’s PPM-queue waiting-count selection (DNS discovery via headless Services). **`load_balancing_policy`** in `callgraph.json` does not apply.

CPU vs **sidecar**: K8s sidecar CPU uses a **1×** multiplier (`1 × sidecar_cpu` per pod) instead of **2×**; ingress is the exception and still uses `UserEntryCount() × 2` (same as **sidecar**). `num_threads` stays `sidecar_cpu` per replica; `cpu_count` (admission) is `cpu / replicas`.

```bash
./build.sh
./deploy.sh approx
./deploy.sh approx-fcfs
./deploy.sh approx-edf
./deploy.sh approx debug   # optional: GLOG verbosity via env or k8s/sidecar-debug-glog.env
```

Use **`destroy.sh approx`** (or `approx-fcfs` / `approx-edf`) to tear down. Load tests use the same URLs as sidecar (`run.sh`).

### Rajomon mode

Matches `benchmarks/hotel` rajomon layout: all workloads talk gRPC with [Rajomon](https://github.com/pennsail/rajomon) interceptors; the HTTP entry is a separate **`rajomon-client`** pod (NodePort **3000** → `ClientPort` 2007) that forwards to **`<entry>-grpc`** (e.g. `frontend-grpc`). Manifests live in `k8s/manifests/app-grpc.yaml`; env template in `k8s/rajomon.env`; graph for Rajomon in `rajomon_init/msgraph.yaml`.

After `go run ./cmd/gen ...`, run `go mod tidy` in the generated benchmark root if you open it as a module.

```bash
./build.sh
# Optional tunables (shell env, same idea as benchmarks/hotel):
# priceUpdateRate, latencyThreshold, tokenUpdateRate, priceStep, tokenUpdateStep
./deploy.sh rajomon
```

Use **`destroy.sh rajomon`** or **`destroy.sh dagor`** when tearing down that mode so `rajomon-client` and the `*-grpc` entry pod are removed. Load tests can use the same URLs as plain mode: `http://<node>:3000/<interface>` (see `entry_path.txt`).

### Dagor mode

Same gRPC topology as Rajomon: `k8s/manifests/app-grpc.yaml`, HTTP entry **`rajomon-client`** (NodePort **3000**), gRPC entry **`<entry>-grpc`**. Env template: **`k8s/dagor.env`** (`dagor=true`). Codegen adds **`dagor/`** and **`dagor_init/`**; `dagor_init` builds a **`BusinessMap`** from entry HTTP paths (interface names) to business class ids `1..N`.

**Tuning (short):**

- **Per-benchmark defaults** — optional `dagor` section in `callgraph.json` sets compile-time defaults in `dagor_init/dagor-config.go` (`queuing_thresh_ms`, `alpha`, `beta`; framework defaults: `2`, `0.45`, `0.01`). Survives `go run ./cmd/gen` regeneration.
- **`Alpha` / `Beta`** — global admission-control knobs (see `benchmarks/hotel/dagor`). Defaults come from `callgraph.json` `dagor` (or framework defaults); override at deploy time with env vars **`Alpha`** and **`Beta`**, or add them to `k8s/dagor.env`, or export `Alpha` / `Beta` in the shell before deploy (they are appended to the merged env like `benchmarks/hotel/deploy.sh` dagor branch).
- **`B` / `U`** — per-request integers: **B** from gRPC metadata **`method`** (the entry interface string, same as the HTTP path) via `BusinessMap`; **U** from **`user-id`** (injected by the end-user Dagor client interceptor). Not set via Alpha/Beta.

```bash
./build.sh
# Optional: export Alpha=0.5 Beta=0.02
./deploy.sh dagor
```

Use **`destroy.sh dagor`** to remove `rajomon-client`, the `*-grpc` entry pod, and related services.

### dagor-lb mode

DAGOR admission control with **p2c/wrr**-style replication and client-side gRPC load balancing. Same gRPC topology as **dagor** (`k8s/manifests/app-grpc-lb.yaml`, HTTP entry **`rajomon-client`**, gRPC entry **`<entry>-grpc`**). Env template: **`k8s/dagor-lb.env`** (`dagor=true`, `plain_lb=true`). The `replicas` and `load_balancing_policy` fields in `callgraph.json` apply (Deployments, per-replica CPU, headless internal Services — see **p2c** / **wrr**).

**Tuning:** same per-benchmark `dagor` defaults and runtime **`Alpha` / `Beta`** overrides as dagor (shell env or `deploy_env` in experiment JSON).

```bash
./build.sh
# Optional: export Alpha=0.5 Beta=0.02
./deploy.sh dagor-lb
```

Use **`destroy.sh dagor-lb`** to remove `rajomon-client`, the `*-grpc` entry workloads, and related services. Load tests use the same URLs as plain/dagor: `http://<node>:3000/<interface>` (`run-plain.sh`).

### rajomon-lb mode

Rajomon admission control with **p2c/wrr**-style replication and client-side gRPC load balancing. Same gRPC topology as **rajomon** (`k8s/manifests/app-grpc-lb.yaml`, HTTP entry **`rajomon-client`** (single replica), gRPC entry **`<entry>-grpc`**). Env template: **`k8s/rajomon-lb.env`** (`rajomon=true`, `plain_lb=true`). The `replicas` and `load_balancing_policy` fields in `callgraph.json` apply (Deployments, per-replica CPU, headless internal Services — see **p2c** / **wrr**).

Each replicated gRPC workload gets a unique Rajomon identity **`serviceName-<podSuffix>`** (from `POD_NAME`) so `priceAggregation: maximal` tracks the max price across all downstream replicas. The end-user **`rajomon-client`** stays a single replica with name **`client`**.

**Tuning:** same Rajomon price env vars as **rajomon** (`priceUpdateRate`, `latencyThreshold`, `tokenUpdateRate`, `priceStep`, `tokenUpdateStep`).

```bash
./build.sh
./deploy.sh rajomon-lb
```

Use **`destroy.sh rajomon-lb`** to tear down. Load tests use the same URLs as plain/rajomon: `http://<node>:3000/<interface>` (`run-plain.sh`).

### p2c and wrr modes

Like **plain**, but supports multiple replicas per microservice via the `replicas` field in `callgraph.json`. Both modes share LB Deployments, headless Services, and the Envoy ingress; they differ only in the client gRPC load-balancing policy pinned by the env file. Deploy with `./deploy.sh p2c` or `./deploy.sh wrr`; tear down with `./destroy.sh p2c` / `./destroy.sh wrr`. Plot/display labels are **P2C** and **WRR**. Load tests use per-API NodePorts like sidecar (`run.sh`): `http://<node>:3000/<interface>`, `http://<node>:3001/<interface>`, …

- Workloads are **Deployments** (not Pods). Per-replica CPU = `cpu / replicas` (fractional allowed) for requests and limits. `GOMAXPROCS = ceil(cpu / replicas)`. Requires `cpu / replicas > 0`.
- Internal gRPC services use **headless** Services (`clusterIP: None`) so clients can discover all pod IPs.
- gRPC clients use **dns:///** load balancing when `plain_lb=true` (set in `k8s/p2c.env`, `k8s/wrr.env`, `k8s/dagor-lb.env`, or `k8s/rajomon-lb.env`). Policy is read at runtime from **`load_balancing_policy`**:
  - **p2c** (`k8s/p2c.env`): always **`least_request`** (power-of-two-choices over active RPC counts; gRPC `choiceCount` defaults to 2).
  - **wrr** (`k8s/wrr.env`): always **`round_robin`** (simple RR, not weighted).
  - **dagor-lb / rajomon-lb**: policy from **`load_balancing_policy`** in `callgraph.json` (default **`least_request`**). Set **`round_robin`** or **`weighted_round_robin`** (ORCA call metrics with `blackoutPeriod: 1s`; servers report QPS / EPS / CPU utilization). Weighted WRR uses EPS via the default `error_utilization_penalty` to deprioritize backends with high error rates.
- Prometheus pushes keep `job=<serviceName>` for all modes. In replicated `plain_lb=true` modes, each pod also pushes with Pushgateway grouping label `instance=<podSuffix>` so replicas do not overwrite each other; `collector.py` writes these as per-replica keys like `backend1-<podSuffix>` in `prometheus.json`. Non-LB modes keep bare keys like `backend1`.
- HTTP entry is a dedicated **Envoy** ingress Pod (`ingress-envoy-lb.yaml`) with NodePorts `3000+i` per user API. Upstream is the frontend headless Service on port **2000** over **HTTP/1** (`LEAST_REQUEST`, `STRICT_DNS`). Ingress CPU and `--concurrency` are `UserEntryCount() × 2` (same core budget as approx ingress). ConfigMap: `p2c-envoy-configs` (shared by p2c and wrr).

## Call Graph Format

- `features` (optional, default `[]`): compile-time toggles baked into generated Go at `go run ./cmd/gen` time. Omitted or empty means all features off. After changing features, regenerate the benchmark and rebuild images (`./build.sh`). Unknown feature names are rejected at codegen time.

  | Feature | Default | Effect |
  |---|---|---|
  | `queueing_delay_export` | off | Enables gRPC tap + `queuing_delay_microseconds` Prometheus histogram (tap-to-interceptor delay). Adds per-request overhead. |

  Queue-size metrics (`max_queue`, `avg_queue`, RPC counters) still work without this feature; only the delay histogram is omitted. This is unrelated to runtime deploy env **`queuing_export`**, which gates whether **sidecar** mode installs the counter interceptor at all.

  Example:

  ```json
  {
    "features": ["queueing_delay_export"],
    "nodes": [ ... ]
  }
  ```

- `load_balancing_policy` (optional, default `least_request`, **dagor-lb and rajomon-lb**): gRPC client load balancer — `least_request` (P2C over active RPC counts), `round_robin`, or `weighted_round_robin` (enables ORCA server metrics). Modes **p2c** and **wrr** pin policy in their env files and ignore this field.
- `dagor` (optional, **dagor and dagor-lb**): per-benchmark DAGOR defaults written to `dagor_init/dagor-config.go` — `queuing_thresh_ms` (default `2`), `alpha` (default `0.45`), `beta` (default `0.01`). All fields optional; omitted fields use framework defaults. Runtime env `Alpha` / `Beta` still override at deploy time.
- `nodes`: one per microservice; `id`, `interfaces` (see **service time** below), `cpu` (optional, default 1, used for cpu_count in sidecar config and app resources), `replicas` (optional, default 1, **p2c, wrr, dagor-lb, rajomon-lb, and approx\***), `sidecar_cpu` (optional, default 1, used for num_threads in sidecar config; k8s sidecar gets 2× this in **sidecar**, 1× in **approx\***), `over_commitment` (optional, must be in [0,1]; written to **sidecar** / **approx\*** configs — see **Over-commitment** for per-mode defaults when omitted), `extra_limit` (optional integer ≥ 0, default 0; written to **sidecar** / **approx\*** configs — see **Extra limit**), `connection_pool_size` (optional, default 200; **entry service only**) — sets both `frontend_pool_connections` and `ingress_pool_connections` in **sidecar** / **approx\*** ConfigMaps

### Service time per interface

Each workload interface (not the synthetic `USER` node) must specify **exactly one** of:

- **`avg_rt`** (number, ≥ 0): fixed busy-loop service time (same units as before; there is no default—omit nothing).
- **`bimodal`**: `{ "rt": [t0, t1], "prob": [p0, p1] }` with two strictly positive `rt` values and probabilities strictly in `(0,1)` summing to 1. Per request, one mode is chosen at random (`ROUTING_SEED` applies, same as weighted routing).
- **`exponential`**: `{ "mean": m }` with strictly positive `mean` (same time units as `avg_rt`). Per request, service time is sampled from an exponential distribution with that mean (`ROUTING_SEED` applies).

Do not set more than one of `avg_rt`, `bimodal`, and `exponential` on the same interface.

Older graphs that omitted `avg_rt` (previously treated as `1.0`) must add an explicit `avg_rt` on each workload interface.

Example bimodal backend: [`../tests/chain-2-bimodal/callgraph.json`](../tests/chain-2-bimodal/callgraph.json). Example exponential backend: [`../tests/chain-2-exponential/callgraph.json`](../tests/chain-2-exponential/callgraph.json).
- `edges`: `source`, `target` as interface IDs (`microservice:interface`); `USER` → entry node. Optional **`api`**: entry interface name (`f1`, not `frontend:f1`). Omitted on `USER` edges means “derive from target”. Omitted on internal edges means the edge applies to **all** APIs (legacy). Multi-API benchmarks should set **`api`** on every internal edge so each API’s virtual graph is explicit.

  **Fan-out groups** (two or more outgoing edges from the same `(source, entry-api)`): exactly **one** of:
  1. **Parallel** — every edge has `"parallel": true`, none have `weight`. Generated code busy-loops once then issues all downstream gRPC calls concurrently (`sync.WaitGroup`). Sidecar mapping sets `pfanout: true` for that row.
  2. **Weighted** — every edge has `weight` (positive, sum 1), none use `parallel: true`. Exactly one downstream per request (cumulative draw). `weight` is not allowed on `USER` edges. Reproducible draws: env **`ROUTING_SEED`** (decimal integer for `strconv.ParseInt`); if unset, wall clock.
  3. **Sequential** — neither of the above: no `weight` on any edge in the group, and not every edge has `parallel: true` (omit `parallel` or `false` on all for legacy sequential order).

  Invalid: mixing weighted and unweighted in one group; `parallel: true` on only some edges in an unweighted multi-edge group; `parallel` with `weight` on the same edge or in the same group.

  Example parallel fan-out (two backends, entry interface `api`):

```json
{
  "nodes": [
    {
      "id": "frontend",
      "interfaces": [
        { "name": "api", "avg_rt": 1.0, "slo": 100, "priority": 1 }
      ],
      "cpu": 1,
      "sidecar_cpu": 1
    },
    {
      "id": "backendA",
      "interfaces": [{ "name": "svc", "avg_rt": 1.0 }],
      "cpu": 1,
      "sidecar_cpu": 1
    },
    {
      "id": "backendB",
      "interfaces": [{ "name": "svc", "avg_rt": 1.0 }],
      "cpu": 1,
      "sidecar_cpu": 1
    }
  ],
  "edges": [
    { "source": "USER", "target": "frontend:api" },
    {
      "source": "frontend:api",
      "target": "backendA:svc",
      "api": "api",
      "parallel": true
    },
    {
      "source": "frontend:api",
      "target": "backendB:svc",
      "api": "api",
      "parallel": true
    }
  ]
}
```
- Entry service sets gRPC metadata **`api`** on outbound calls (same string as on edges). Non-entry handlers `switch` on `api` for downstream calls.
- With **`sidecar=true`**, the entry HTTP server requires inbound headers **`rpc-id`**, **`rpc-local-id`**, and **`deadline`** (injected by the sidecar on the forwarded HTTP/1 request) and forwards them as gRPC metadata so downstream sidecars can correlate with the parent INGRESS RPC (PPM) and use the absolute deadline for egress validation / EDF scheduling. Mid-tier services copy all incoming metadata via `ContextPropagationInterceptor`.
- Entry: HTTP/1 server; others: gRPC services
- Entry path: `/{interface}` (e.g. `/Z8trRkp4mp`). See `entry_path.txt` in output dir.
- Busy loop: 320 repeats = 1ms

### Sidecar mapping note

Non-entry sidecar `mapping` lists downstreams per gRPC method as a **union** over APIs. If methods differ by `api`, the config may over-approximate dependencies until the sidecar schema supports per-`api` rows.
