# Roshanfer

A microservice benchmarking framework for evaluating admission control and load balancing policies in distributed systems.

## Overview

Roshanfer provides tooling to generate, deploy, and run microservice benchmarks from call graph specifications. The framework supports multiple execution modes including custom admission control sidecars, Rajomon, Dagor, and plain gRPC, enabling systematic evaluation of microservice scheduling and overload protection techniques.

The repository includes a call graph code generator, pre-built benchmarks derived from real-world applications (DeathStarBench hotel/social network) and production traces (Alibaba), plus synthetic test topologies for controlled experiments.

## Project Structure

```
├── callgraph-framework/   # Call graph → Go microservice code generator
│   ├── cmd/gen/          # Generator CLI
│   └── cmd/viz/          # Call graph visualizer (PDF output)
├── hotel/                 # Hotel reservation benchmark (DeathStarBench-derived)
├── social/                # Social network benchmark (DeathStarBench-derived)
├── alibaba-large/         # Large-scale benchmark from Alibaba production traces
├── tests/                 # Synthetic benchmarks (chain, fan-out, fan-in, etc.)
├── k8s/                   # K3s cluster setup scripts with Cilium networking
├── sidecar/               # Roshanfer C++ admission control sidecar (git submodule)
└── provisioning/          # Node provisioning and performance tuning scripts
```

### Key Files
- `LICENSE` - MIT License
- `THIRD_PARTY.md` - Attribution for DeathStarBench and sidecar dependencies
- `AGENTS.md` - Development agent instructions

## Installation

### Prerequisites

- **Go 1.25+**
- **Protocol Buffers**: `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` plugins
- **Graphviz**: For call graph visualization (`apt install graphviz` or `brew install graphviz`)
- **Docker with Buildx**: For building container images (`docker buildx`)
- **Kubernetes cluster**: K3s setup scripts provided in `k8s/` (optional)
- **Python 3 with matplotlib**: For paper-style visualizations (optional, requires `.venv` at repo root)

### Building from Source

Each benchmark is a self-contained Go module. Navigate to any benchmark directory and build:

```bash
cd hotel
go mod download
./build.sh [tag]
```

The `build.sh` script builds the sidecar (if submodule is initialized) and all workload Docker images using `docker buildx bake`.

## Usage

### Generating a New Benchmark

Use the call graph framework to generate runnable benchmarks from JSON specifications:

```bash
cd callgraph-framework
go run ./cmd/gen <callgraph.json> -o <output-dir>
```

Example:
```bash
go run ./cmd/gen ../tests/fan-out/callgraph.json -o ../tests/fan-out
```

After generation, run `go mod tidy` in the output directory.

### Visualizing Call Graphs

Generate a PDF visualization of any call graph:

```bash
cd callgraph-framework
go run ./cmd/viz <callgraph.json> -o callgraph.pdf
```

For ACM-style paper figures (unlabeled circles, directed edges):
```bash
go run ./cmd/viz -paper <callgraph.json> -o callgraph-service.pdf
```

The `-paper` flag requires Python with matplotlib in a `.venv` at the repository root:
```bash
# From repository root
python3 -m venv .venv
.venv/bin/pip install matplotlib
```

### Deploying Benchmarks

Each generated benchmark includes deployment scripts. Supported modes:

```bash
# Plain gRPC (no sidecar)
./deploy.sh plain

# Roshanfer sidecar with admission control
./deploy.sh sidecar

# Rajomon (Cornell's rate limiting system)
./deploy.sh rajomon

# Dagor (business-aware scheduling)
./deploy.sh dagor
```

Tear down with:
```bash
./destroy.sh [mode]
```

### Kubernetes Cluster Setup

Use the provided K3s scripts to bootstrap a high-performance cluster with Cilium networking:

```bash
cd k8s
# Edit hosts.txt and config.env
./create.sh
```

See `k8s/README.md` for details on static CPU management and cluster configuration.

## Important Features

### Call Graph Code Generation
- JSON-based call graph specification with nodes, edges, and service times
- Support for sequential, parallel (fan-out), and weighted (probabilistic) routing
- Bimodal service time distributions
- Multi-API benchmarks with per-API routing
- Automatic generation of gRPC protobuf definitions, Kubernetes manifests, and deployment scripts

### Execution Modes
- **Plain**: Standard gRPC microservices
- **Sidecar**: Roshanfer admission control with overload protection
- **Rajomon**: Token bucket rate limiting (Pennsylvania/Cornell)
- **Dagor**: Business-aware scheduling with priority classes
- **Envoy** and **Breakwater**: Additional proxy/admission modes

### Deployment Features
- Multi-stage Docker builds with Buildx for fast parallel compilation
- Kubernetes manifests with configurable CPU/memory resources
- Prometheus metrics collection
- Debug mode with configurable glog verbosity
- Over-commitment parameter sweeps for latency-throughput experiments

### Benchmark Corpus
- **Hotel**: 8 microservices (frontend, geo, profile, rate, reservation, search, user)
- **Social Network**: 6 microservices (nginx, compose, home timeline, posts, social graph, user timeline)
- **Alibaba Large**: Production-scale trace-derived topology
- **Synthetic Tests**: 17 test benchmarks covering chain, fan-out, fan-in, ingress stress, and dynamic routing patterns

## License

MIT License. See `LICENSE` for details.

Portions of `hotel/` and `social/` are derived from [DeathStarBench](https://github.com/delimitrou/DeathStarBench) (Apache 2.0). The `sidecar/` submodule and its dependencies retain their respective licenses. See `THIRD_PARTY.md` for complete attribution.

## References

- DeathStarBench: https://github.com/delimitrou/DeathStarBench
- Rajomon: https://github.com/pennsail/rajomon
- Roshanfer sidecar: https://github.com/farzad1132/roshanfer-sidecar
