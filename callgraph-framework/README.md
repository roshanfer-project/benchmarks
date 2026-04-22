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

Generates the benchmark and `callgraph.pdf` in the output directory.

### Visualizer

Generate a PDF visualization of the callgraph:

```bash
go run ./cmd/viz <callgraph.json> [-o callgraph.pdf]
```

Requires [graphviz](https://graphviz.org/) (`dot` on PATH): `apt install graphviz` / `brew install graphviz`

## Requirements

- Go 1.25+
- protoc with protoc-gen-go and protoc-gen-go-grpc plugins
- graphviz (for viz tool and `gen -v`)

## Scripts

- `build.sh [tag]` — build all images
- `push.sh [tag]` — push all images (run after build)
- `deploy.sh [plain|sidecar|rajomon|dagor]` — deploy to K8s (uses `TAG`, `REGISTRY`). `./deploy.sh sidecar debug` enables workload debug (see below). `debug` is only valid with `sidecar`, not with plain, rajomon, or dagor. For **plain** (non-sidecar) HTTP entry, optional client deadline/retry env vars are described under **Client RPC deadline and retry** (not applied in sidecar mode).
- `destroy.sh` — tear down
- `collect_logs.sh` — collect pod logs. If the environment variable `COLLECT_SIDECAR_NANOLOG=1` is set (done by `exec` when `--nanolog-debug` is enabled) and mode is `sidecar`, the script also `kubectl cp`s `/compressedLog` from each sidecar container into `$OUTPUT_DIR` as `*-sidecar.clog` (plus ingress as `*-ingress-sidecar.clog`). Decompression uses `benchmarks/sidecar/external/NanoLog/runtime/decompressor` from the repo checkout that runs the executor.

### Sidecar deploy debug

Use `./deploy.sh sidecar debug` after `./build.sh` / `./push.sh` as usual. Requires [mikefarah yq](https://github.com/mikefarah/yq) v4 on `PATH` (the script exits with an install hint if it is missing).

Debug mode only changes **workload** manifests (`*-sidecar.yaml`, `ingress.yaml`) under a temp copy before `kubectl apply`: every `Pod` gets `spec.restartPolicy: Never`. If you set `SIDECAR_GLOG_V` and/or `SIDECAR_GLOG_VMODULE` (or define them in `k8s/sidecar-debug-glog.env`), the `sidecar` container also gets `GLOG_v` / `GLOG_vmodule` (existing entries with those names are replaced). You can use debug mode for **restart behavior only** without setting any verbosity. **Prometheus** is still applied from `k8s/manifests/prometheus.yaml` exactly like non-debug sidecar deploy.

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

Optional `k8s/sidecar-debug-glog.env` under **this** benchmark’s `k8s/` (next to that benchmark’s `deploy.sh`). A file in another directory (e.g. `tests/one-service/k8s/...`) is not read when you deploy from `tests/fan-out-dynamic-0-9` — copy, symlink, or recreate it there. Same variable names as above, `#` comments allowed; script reads it only in the debug branch. **Precedence:** values already set in the environment win; the file fills `SIDECAR_GLOG_V` / `SIDECAR_GLOG_VMODULE` only when those variables are unset, so `SIDECAR_GLOG_V=3 ./deploy.sh sidecar debug` overrides a value from the file.

### Rajomon mode

Matches `benchmarks/hotel` rajomon layout: all workloads talk gRPC with [Rajomon](https://github.com/pennsail/rajomon) interceptors; the HTTP entry is a separate **`rajomon-client`** pod (NodePort **3000** → `ClientPort` 2007) that forwards to **`<entry>-grpc`** (e.g. `frontend-grpc`). Manifests live in `k8s/manifests/app-grpc.yaml`; env template in `k8s/rajomon.env`; graph for Rajomon in `rajomon_init/msgraph.yaml`.

After `go run ./cmd/gen ...`, run `go mod tidy` in the generated benchmark root if you open it as a module.

```bash
./build.sh
./push.sh
# Optional tunables (shell env, same idea as benchmarks/hotel):
# priceUpdateRate, latencyThreshold, tokenUpdateRate, priceStep, tokenUpdateStep
./deploy.sh rajomon
```

Use **`destroy.sh rajomon`** or **`destroy.sh dagor`** when tearing down that mode so `rajomon-client` and the `*-grpc` entry pod are removed. Load tests can use the same URLs as plain mode: `http://<node>:3000/<interface>` (see `entry_path.txt`). Optional client deadline/retry env is documented under **Client RPC deadline and retry** below.

### Dagor mode

Same gRPC topology as Rajomon: `k8s/manifests/app-grpc.yaml`, HTTP entry **`rajomon-client`** (NodePort **3000**), gRPC entry **`<entry>-grpc`**. Env template: **`k8s/dagor.env`** (`dagor=true`). Codegen adds **`dagor/`** and **`dagor_init/`**; `dagor_init` builds a **`BusinessMap`** from entry HTTP paths (interface names) to business class ids `1..N`.

**Tuning (short):**

- **`Alpha` / `Beta`** — global admission-control knobs (see `benchmarks/hotel/dagor`). Defaults are compiled into `dagor_init`; override with env vars **`Alpha`** and **`Beta`**, or add them to `k8s/dagor.env`, or export `Alpha` / `Beta` in the shell before deploy (they are appended to the merged env like `benchmarks/hotel/deploy.sh` dagor branch).
- **`B` / `U`** — per-request integers: **B** from gRPC metadata **`method`** (the entry interface string, same as the HTTP path) via `BusinessMap`; **U** from **`user-id`** (injected by the end-user Dagor client interceptor). Not set via Alpha/Beta.

```bash
./build.sh
./push.sh
# Optional: export Alpha=0.5 Beta=0.02
./deploy.sh dagor
```

Use **`destroy.sh dagor`** to remove `rajomon-client`, the `*-grpc` entry pod, and related services. Optional client deadline/retry env is documented under **Client RPC deadline and retry** below.

### Client RPC deadline and retry

Applies only to **plain**, **rajomon**, and **dagor** deployments. **Sidecar** workloads do not attach deadline/retry policy on generated egress or internal client paths (`sidecar=true`); behavior there stays as before.

Codegen writes **`pkg/rpcpolicy/rpcpolicy.go`** (policy helpers) and **`pkg/grpc.go`** with **`DialClient`** (chains interceptors). Regenerate with **`go run ./cmd/gen ...`** from this directory; then **`go mod tidy`** in the benchmark root if needed.

**Interceptor chain** on **`DialClient`** (non-sidecar only), outer → inner:

1. **Retry** — if **`BENCH_RPC_RETRY_MODE`** is `fixed` or `token_bucket`.
2. **Deadline `fixed`** — if **`BENCH_RPC_DEADLINE_MODE`** is `fixed` (**2 s** per unary attempt, including each retry).
3. **Rajomon / Dagor / plain** unary interceptors passed into **`DialClient`**, then the real invoker.

**Note on the value `fixed`:** **`BENCH_RPC_DEADLINE_MODE=fixed`** (per-RPC **2 s** timeout) and **`BENCH_RPC_RETRY_MODE=fixed`** (retry + backoff) both use the string **`fixed`** but are **separate** settings on **different** env vars. In **`experiments.json`**, **`deadline_mode`** and **`retry_mode`** map to those two vars independently.

**Deadline (`BENCH_RPC_DEADLINE_MODE`):**

- `none` (default when unset) — no policy deadline on the HTTP entry for plain/`rajomon-client`; no **`DialClient`** deadline interceptor. Downstream timing unchanged unless something else sets **`context`** deadlines.
- `remaining_slo` — the HTTP entry (plain service or `rajomon-client`) sets a context deadline of **60%** of the per-API SLO in milliseconds. The value is read from **`BENCH_RPC_SLO_MS_<interface>`** (same name as the HTTP path / experiment `apis` entry). That deadline propagates on outbound gRPC so child RPCs see the **remaining** budget. If `remaining_slo` is set and a required **`BENCH_RPC_SLO_MS_*`** is missing or invalid, the process **exits with an error at startup** (`rpcpolicy.MustValidatePolicyEnv`; the executor also validates before deploy when this mode is selected).
- `fixed` — each outbound unary gRPC call uses **`context.WithTimeout(..., 2s)`** on the client (see interceptor chain above). **Not** an end-to-end SLO budget: the plain / `rajomon-client` HTTP handler does **not** call **`MaybeDeadlineForAPI`** for this mode; each hop independently caps that hop at **2 s** (still bounded by any stricter parent deadline, e.g. HTTP **`Request.Context`**). Does **not** require **`BENCH_RPC_SLO_MS_*`** for the deadline itself.

**Retry (`BENCH_RPC_RETRY_MODE`):**

- `none` (default) — no retry interceptor.
- `fixed` — up to **4** total attempts (1 initial + 3 retries). Backoff between attempts: **`BENCH_RPC_SLO_MS_<api>`** ms **+** **uniform jitter** in **`[0, BENCH_RPC_SLO_MS_<api>]`** ms (same **`<api>`** as metadata). Outgoing metadata **`api`** or **`method`** must be set, and the matching **`BENCH_RPC_SLO_MS_*`** env must be present and positive; otherwise the client returns an **error** (no silent fallback).
- `token_bucket` — retries consume **one** token per retry attempt (bucket starts full at **`BENCH_RPC_RETRY_BUCKET_CAPACITY`**, default **10**). Each **successful** RPC adds **0.02** token back (capped at capacity); there is no time-based refill. Between retries uses the same **`SLO + [0,SLO]`** ms backoff and metadata/SLO requirements as retry mode **`fixed`**. Optional env: **`BENCH_RPC_RETRY_BUCKET_CAPACITY`**. If there are not enough tokens for the next retry, the call fails without that retry.

Retries apply to **all** unary RPC errors returned by the invoker, including **`ResourceExhausted`** (overload). No further attempts are scheduled if the request context is already done (e.g. deadline expired).

**Environment quick reference**

| Variable | Role |
|----------|------|
| **`BENCH_RPC_DEADLINE_MODE`** | `none` \| `remaining_slo` \| `fixed` |
| **`BENCH_RPC_RETRY_MODE`** | `none` \| `fixed` \| `token_bucket` |
| **`BENCH_RPC_SLO_MS_<api>`** | SLO in ms; **required** for `remaining_slo` (startup + executor); **required** for retry backoff when retry mode is `fixed` or `token_bucket` |
| **`BENCH_RPC_RETRY_BUCKET_CAPACITY`** | Optional; token bucket size (default **10**) |

**Where SLO numbers live:** only in the executor suite **`config.json`** field **`slos`** (existing schema), which the executor maps to **`BENCH_RPC_SLO_MS_<api>`** in the workload env. Do **not** put SLO fields under **`fault-tolerance`** in **`experiments.json`**. You can still override a single API for one run via **`deploy_env`**.

**Optional `experiments.json` `fault-tolerance`** (hyphenated key; **`fault_tolerance`** also accepted when merging): JSON object with **`deadline_mode`**, **`retry_mode`**, and optionally **`retry_bucket_capacity`** (maps to **`BENCH_RPC_RETRY_BUCKET_CAPACITY`**). If the whole object is omitted, policy defaults are **`none`** / **`none`**. Omitted sub-keys inside the object also default to **`none`**.

**Env merge order** for a run (executor): **`config.slos` → `fault-tolerance` → tuner `deploy_params` → experiment `deploy_env`** (later keys win). The executor passes the merged map into **`deploy.sh`** as real environment variables. **`deploy.sh`** appends every defined **`BENCH_RPC_*`** variable from that process environment into the workload **`ConfigMap`** (alongside **`k8s/plain.env`**, **`dagor.env`**, etc.) so pods such as **`frontend-grpc`** receive **`BENCH_RPC_DEADLINE_MODE`** / **`BENCH_RPC_RETRY_MODE`** / SLO keys. Regenerating benchmarks picks up **`deploy.sh`** / **`pkg/rpcpolicy`** changes; editing **`slos`** in **`config.json`** does not require rebuilding images.

**RWG / workload wrappers:** Generated **`run.sh`** and **`run-plain.sh`** invoke RWG with **`RWG_RATES`** and **`RWG_DURATIONS`** (comma-separated; set by **`exec.runner`**). Invocation shape: **`./run.sh <protocol> <api> <output_dir>`** (and optional **`--ignore-errors`** on **`run-plain.sh`**). Do not pass legacy `BASE RATE DURATION` positionals. HTTP client timeouts in RWG (`-t`) are separate from gRPC deadline policy above.

See also **Rajomon mode** and **Dagor mode** above for deploy layout; tunables for this section apply to those HTTP→gRPC entry paths as well.

### Fail-slow injection (gRPC only; `exec`)

Supported only for **`rajomon`** and **`dagor`** workloads (pods in `k8s/manifests/app-grpc.yaml`: inner gRPC services and **`entry-grpc`**). **Plain**, **sidecar**, and **`rajomon-client`** do not run this path.

Each gRPC server starts a loopback admin HTTP server (default **`127.0.0.1:19090`**, override with **`BENCH_FAILSLOW_ADMIN_ADDR`**). **POST `/failslow`** with JSON `{"duration_ms": <int>, "extra_ms": <int>}` arms a window: until `now + duration_ms`, successful unary RPCs sleep an extra **`extra_ms`** after handler work and before the response is sent (unary interceptor; errors are not delayed).

The executor can arm a pod mid-run: in **`experiments.json`**, optional **`failslow`** object:

- **`pod`** — Kubernetes pod name (e.g. `checkout`, `frontend-grpc` for the gRPC entry).
- **`after_sec`** — seconds after **RWG start** before arming.
- **`duration_sec`** — length of the window (converted to `duration_ms` for the admin API).
- **`extra_ms`** — per-request extra delay while the window is active.
- **`container`** — optional, default **`app`**.
- **`kubernetes_namespace`** — optional; if set, passed to **`kubectl exec -n ...`**.

Requires **`kubectl`** on the machine running **`exec`**. The executor arms fail-slow with **`kubectl exec … wget --post-data=…`** (BusyBox **`wget`** ships with the default **`alpine`** runtime image; no extra packages needed). **`app-grpc`** manifests declare **`containerPort` 19090** (`failslow-admin`) on gRPC pods (not on **`rajomon-client`**).

## Call Graph Format

- `nodes`: one per microservice; `id`, `interfaces` (see **service time** below), `cpu` (optional, default 1, used for cpu_count in sidecar config and app resources), `sidecar_cpu` (optional, default 1, used for num_threads in sidecar config; k8s sidecar gets 2× this), `over_commitment` (optional, default 0, must be in [0,1]; written to sidecar config)

### Service time per interface

Each workload interface (not the synthetic `USER` node) must specify **exactly one** of:

- **`avg_rt`** (number, ≥ 0): fixed busy-loop service time (same units as before; there is no default—omit nothing).
- **`bimodal`**: `{ "rt": [t0, t1], "prob": [p0, p1] }` with two strictly positive `rt` values and probabilities strictly in `(0,1)` summing to 1. Per request, one mode is chosen at random (`ROUTING_SEED` applies, same as weighted routing).

Do not set both `avg_rt` and `bimodal` on the same interface.

Older graphs that omitted `avg_rt` (previously treated as `1.0`) must add an explicit `avg_rt` on each workload interface.

Example bimodal backend: [`../tests/chain-2-bimodal/callgraph.json`](../tests/chain-2-bimodal/callgraph.json).
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
- Entry: HTTP/1 server; others: gRPC services
- Entry path: `/{interface}` (e.g. `/Z8trRkp4mp`). See `entry_path.txt` in output dir.
- Busy loop: 320 repeats = 1ms

### Sidecar mapping note

Non-entry sidecar `mapping` lists downstreams per gRPC method as a **union** over APIs. If methods differ by `api`, the config may over-approximate dependencies until the sidecar schema supports per-`api` rows.
