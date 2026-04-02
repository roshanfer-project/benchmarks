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
- `deploy.sh [plain|sidecar] [debug]` — deploy to K8s (uses `TAG`, `REGISTRY`). `./deploy.sh sidecar debug` enables workload debug (see below). `debug` is only valid with `sidecar`, not with plain mode.
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

## Call Graph Format

- `nodes`: one per microservice; `id`, `interfaces` (array of `{name, avg_rt}`), `cpu` (optional, default 1, used for cpu_count in sidecar config and app resources), `sidecar_cpu` (optional, default 1, used for num_threads in sidecar config; k8s sidecar gets 2× this), `over_commitment` (optional, default 0, must be in [0,1]; written to sidecar config)
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
