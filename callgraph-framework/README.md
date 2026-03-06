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

## Requirements

- Go 1.25+
- protoc with protoc-gen-go and protoc-gen-go-grpc plugins

## Scripts

- `build.sh [tag]` — build all images
- `push.sh [tag]` — push all images (run after build)
- `deploy.sh` — deploy to K8s (uses TAG env var)
- `destroy.sh` — tear down
- `collect_logs.sh` — collect pod logs

## Call Graph Format

- `nodes`: one per microservice; `id`, `interfaces` (array of `{name, avg_rt}`), `cpu` (optional, default 1)
- `edges`: source, target as interface IDs (`microservice:interface`); USER -> entry node
- Entry: HTTP/1 server; others: gRPC services
- Entry path: `/{interface}` (e.g. `/Z8trRkp4mp`). See `entry_path.txt` in output dir.
- Busy loop: 320 repeats = 1ms
