# Call Graph Benchmark Framework

Generates runnable Go microservice benchmarks from call graph JSON.

## Repository layout

```text
callgraph-framework/
├── cmd/gen/                   callgraph.json → Go services + K8s manifests
├── cmd/viz/                   callgraph PDF (graphviz / paper figure)
├── gen/                       codegen (services, k8s, rajomon, dagor, envoy)
├── viz/                       graphviz + matplotlib paper figure
└── envoy-stats-exporter/      Envoy stats sidecar image
```

## Usage

```bash
go run ./cmd/gen <callgraph.json> [-o output-dir]
```

Example:
```bash
go run ./cmd/gen ../alibaba-large/callgraph.json -o ../alibaba-large
```

Generates the benchmark and `callgraph.pdf` in the output directory.

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
- `deploy.sh [plain|roshanfer|rajomon|dagor]` — deploy to K8s (uses `TAG`, `REGISTRY`). `./deploy.sh roshanfer debug` enables workload debug (see below). `debug` is only valid with `roshanfer`, not with plain, rajomon, or dagor. For **roshanfer** only: if **`SIDECAR_OVER_COMMIT`** is set (non-empty), every `over_commitment:` field in `sidecar-configs.yaml` is rewritten to that value before apply (needs `perl` on PATH); omit it to keep values from `callgraph.json` codegen. Ingress AIMD knobs can be overridden the same way via **`SIDECAR_AIMD_ERR_D`**, **`SIDECAR_AIMD_ERR_I`**, **`SIDECAR_AIMD_ADJ_D`**, **`SIDECAR_AIMD_ADJ_I`**, **`SIDECAR_SAFE_MULTIPLY`** (written into the embedded `ingress.yaml` block).
- `destroy.sh` — tear down
- `collect_logs.sh` — collect pod logs. If the environment variable `COLLECT_SIDECAR_NANOLOG=1` is set (done by `exec` when `--nanolog-debug` is enabled) and mode is `roshanfer`, the script also `kubectl cp`s `/compressedLog` from each sidecar container into `$OUTPUT_DIR` as `*-sidecar.clog` (plus ingress as `*-ingress-sidecar.clog`). Decompression uses `benchmarks/sidecar/external/NanoLog/runtime/decompressor` from the repo checkout that runs the executor.

### Sidecar deploy debug

Use `./deploy.sh roshanfer debug` after `./build.sh` as usual. Requires [mikefarah yq](https://github.com/mikefarah/yq) v4 on `PATH` (the script exits with an install hint if it is missing).

Debug mode only changes **workload** manifests (`*-sidecar.yaml`, `ingress.yaml`) under a temp copy before `kubectl apply`: every `Pod` gets `spec.restartPolicy: Never`. If you set `SIDECAR_GLOG_V` and/or `SIDECAR_GLOG_VMODULE` (or define them in `k8s/sidecar-debug-glog.env`), the `sidecar` container also gets `GLOG_v` / `GLOG_vmodule` (existing entries with those names are replaced). You can use debug mode for **restart behavior only** without setting any verbosity. **Prometheus** is still applied from `k8s/manifests/prometheus.yaml` exactly like non-debug roshanfer deploy.

Set verbosity via environment (or optional file — see precedence below):

- `SIDECAR_GLOG_V` — maps to glog `GLOG_v` (e.g. `2` enables `VLOG(0)`–`VLOG(2)` globally where other filters allow).
- `SIDECAR_GLOG_VMODULE` — maps to `GLOG_vmodule`; patterns use **source file basenames** (no `.cc`), e.g. `state=2`, `connection=1`, `event_loop=3`.

Examples:

```bash
export SIDECAR_GLOG_V=2
./deploy.sh roshanfer debug
```

```bash
export SIDECAR_GLOG_VMODULE=state=2,connection=1
./deploy.sh roshanfer debug
```

```bash
export SIDECAR_GLOG_V=1
export SIDECAR_GLOG_VMODULE=event_loop=3
./deploy.sh roshanfer debug
```

Optional `k8s/sidecar-debug-glog.env` under **this** benchmark’s `k8s/` (next to that benchmark’s `deploy.sh`). A file in another directory (e.g. `tests/one-service/k8s/...`) is not read when you deploy from `tests/fan-out-dynamic-0-9` — copy, symlink, or recreate it there. Same variable names as above, `#` comments allowed; script reads it only in the debug branch. **Precedence:** values already set in the environment win; the file fills `SIDECAR_GLOG_V` / `SIDECAR_GLOG_VMODULE` only when those variables are unset, so `SIDECAR_GLOG_V=3 ./deploy.sh roshanfer debug` overrides a value from the file.

### Sidecar over-commitment override

For latency-vs-throughput sweeps over admission **over-commitment**, set **`SIDECAR_OVER_COMMIT`** when deploying roshanfer mode (e.g. `0`, `0.2`, `1`). `deploy.sh` stages `sidecar-configs.yaml`, replaces each embedded `over_commitment:` value with `SIDECAR_OVER_COMMIT`, then applies the patched manifest (**requires `perl` on PATH**). After patching, the script checks every `over_commitment:` line matches **`SIDECAR_OVER_COMMIT`** and exits non‑zero otherwise (catch perl/path/format failures). Only snippets that already contain `over_commitment` are affected (ingress blocks typically do not). The experiment executor forwards **`deploy_env`** into the deploy environment.

### Sidecar ingress AIMD override

For one-parameter sensitivity sweeps, set any of **`SIDECAR_AIMD_ERR_D`**, **`SIDECAR_AIMD_ERR_I`**, **`SIDECAR_AIMD_ADJ_D`**, **`SIDECAR_AIMD_ADJ_I`**, **`SIDECAR_SAFE_MULTIPLY`**. `deploy.sh` inserts or replaces that key in the embedded `ingress.yaml` ConfigMap snippet (after `name: ingress` when missing). Unset vars leave the sidecar defaults. The experiment executor forwards **`deploy_env`** into the deploy environment.

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

- **`Alpha` / `Beta`** — global admission-control knobs (see `benchmarks/hotel/dagor`). Defaults are compiled into `dagor_init`; override with env vars **`Alpha`** and **`Beta`**, or add them to `k8s/dagor.env`, or export `Alpha` / `Beta` in the shell before deploy (they are appended to the merged env like `benchmarks/hotel/deploy.sh` dagor branch).
- **`B` / `U`** — per-request integers: **B** from gRPC metadata **`method`** (the entry interface string, same as the HTTP path) via `BusinessMap`; **U** from **`user-id`** (injected by the end-user Dagor client interceptor). Not set via Alpha/Beta.

```bash
./build.sh
# Optional: export Alpha=0.5 Beta=0.02
./deploy.sh dagor
```

Use **`destroy.sh dagor`** to remove `rajomon-client`, the `*-grpc` entry pod, and related services.

## Call Graph Format

- `nodes`: one per microservice; `id`, `interfaces` (see **service time** below), `cpu` (optional, default 1, used for cpu_count in sidecar config and app resources), `sidecar_cpu` (optional, default 1, used for num_threads in sidecar config; k8s sidecar gets 2× this), `over_commitment` (optional, default 0, must be in [0,1]; written to sidecar config)

### Service time per interface

Each workload interface (not the synthetic `USER` node) must specify **exactly one** of:

- **`avg_rt`** (number, ≥ 0): fixed busy-loop service time (same units as before; there is no default—omit nothing).
- **`bimodal`**: `{ "rt": [t0, t1], "prob": [p0, p1] }` with two strictly positive `rt` values and probabilities strictly in `(0,1)` summing to 1. Per request, one mode is chosen at random (`ROUTING_SEED` applies, same as weighted routing).

Do not set both `avg_rt` and `bimodal` on the same interface.

Older graphs that omitted `avg_rt` (previously treated as `1.0`) must add an explicit `avg_rt` on each workload interface.

Example bimodal backend: [`../tests/chain-2-bimodal/callgraph.json`](../tests/chain-2-bimodal/callgraph.json).
- `edges`: `source`, `target` as interface IDs (`microservice:interface`); `USER` → entry node. Optional **`api`**: entry interface name (`f1`, not `frontend:f1`). When omitted on `USER` edges, the `api` is derived from the target. When omitted on internal edges, the edge applies to **all** APIs (legacy). Multi-API benchmarks should set **`api`** on every internal edge so each API’s virtual graph is explicit.

  **Fan-out groups** (two or more outgoing edges from the same `(source, entry-api)`): exactly **one** of:
  1. **Parallel** — every edge has `"parallel": true`, none have `weight`. Generated code busy-loops once then issues all downstream gRPC calls concurrently (`sync.WaitGroup`). Sidecar mapping sets `pfanout: true` for that row.
  2. **Weighted** — every edge has `weight` (positive, sum 1), none use `parallel: true`. Exactly one downstream per request (cumulative draw). `weight` is not allowed on `USER` edges. Reproducible draws: env **`ROUTING_SEED`** (decimal integer for `strconv.ParseInt`); if unset, wall clock.
  3. **Sequential** — neither of the above: no `weight` on any edge in the group, and not every edge has `parallel: true` (omit `parallel` or set `parallel` to `false` on all edges for legacy sequential order).

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
- With **`sidecar=true`**, the entry HTTP server requires inbound headers **`rpc-id`** and **`rpc-local-id`** (injected by the sidecar on the forwarded HTTP/1 request) and forwards both as gRPC metadata so downstream sidecars can correlate with the parent INGRESS RPC (Request Limit Protocol).
- Entry: HTTP/1 server; others: gRPC services
- Entry path: `/{interface}` (e.g. `/Z8trRkp4mp`). See `entry_path.txt` in output dir.
- Busy loop: 320 repeats = 1ms

### Sidecar mapping note

Non-entry sidecar `mapping` lists downstreams per gRPC method as a **union** over APIs. If methods differ by `api`, the config may over-approximate dependencies until the sidecar schema supports per-`api` rows.
