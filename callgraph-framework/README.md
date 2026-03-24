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
- `deploy.sh` — deploy to K8s (uses TAG env var)
- `destroy.sh` — tear down
- `collect_logs.sh` — collect pod logs

## Call Graph Format

- `nodes`: one per microservice; `id`, `interfaces` (array of `{name, avg_rt}`), `cpu` (optional, default 1, used for cpu_count in sidecar config and app resources), `sidecar_cpu` (optional, default 1, used for num_threads in sidecar config; k8s sidecar gets 2× this), `over_commitment` (optional, default 0, must be in [0,1]; written to sidecar config)
- `edges`: `source`, `target` as interface IDs (`microservice:interface`); `USER` → entry node. Optional **`api`**: entry interface name (`f1`, not `frontend:f1`). Omitted on `USER` edges means “derive from target”. Omitted on internal edges means the edge applies to **all** APIs (legacy). Multi-API benchmarks should set **`api`** on every internal edge so each API’s virtual graph is explicit. Optional **`weight`** (positive float): if set on any outgoing edge from a given `(source, entry-api)` routing group, **every** edge in that group must have `weight`, and weights must sum to 1. The generated service then calls **exactly one** downstream per request (cumulative draw). Omit `weight` on all edges in the group to keep the legacy behavior (call every downstream in JSON order). `weight` is not allowed on `USER` edges. Reproducible draws: set env **`ROUTING_SEED`** to a decimal integer (passed to `strconv.ParseInt`); if unset, seed from wall clock.
- Entry service sets gRPC metadata **`api`** on outbound calls (same string as on edges). Non-entry handlers `switch` on `api` for downstream calls.
- Entry: HTTP/1 server; others: gRPC services
- Entry path: `/{interface}` (e.g. `/Z8trRkp4mp`). See `entry_path.txt` in output dir.
- Busy loop: 320 repeats = 1ms

### Sidecar mapping note

Non-entry sidecar `mapping` lists downstreams per gRPC method as a **union** over APIs. If methods differ by `api`, the config may over-approximate dependencies until the sidecar schema supports per-`api` rows.
