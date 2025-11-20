# Benchmark Generator Tool

This tool generates microservice benchmarks for testing service mesh configurations, including both plain (direct communication) and Roshanfer (sidecar-based) modes. It generates all necessary artifacts including Go services, Kubernetes manifests, Docker images, and configuration files.

## Features

- **Dual Mode Support**: Generate artifacts that can be deployed in either plain or roshanfer mode
- **Automatic Code Generation**: Generates frontend and backend services with proper gRPC/HTTP handling
- **Kubernetes Integration**: Creates complete Kubernetes manifests (Deployments, Services, ConfigMaps)
- **CPU Pinning**: Supports CPU core assignment for performance testing
- **Flexible Configuration**: YAML-based configuration for easy customization

## Requirements

- Go 1.24+ 
- Docker (for building images)
- Kubernetes cluster (k3s, microk8s, or standard)
- kubectl (or k3s/microk8s with kubectl alias)
- protoc (for protobuf generation)

## Installation

```bash
# Clone the repository
cd benchmark-gen

# Build the tool
go build -o benchmark-gen .

# Or run directly
go run . [command]
```

## Commands

### Generate

Generates all benchmark artifacts from an input YAML configuration.

```bash
go run . generate -i <input-yaml> -o <output-dir>
```

**Options:**
- `-i, --input <path>`: Input YAML file path (required)
- `-o, --output <dir>`: Output directory (required)

**Example:**
```bash
go run . generate -i example-benchmark.yaml -o testbench
```

### Deploy

Deploys the generated benchmark to Kubernetes.

```bash
go run . deploy -o <output-dir> -c <deploy-config> -i <input-yaml> [--mode <mode>]
```

**Options:**
- `-o, --output <dir>`: Output directory containing generated artifacts (required)
- `-c, --config <file>`: Deployment config file (required)
- `-i, --input <path>`: Input YAML file path (required)
- `-m, --mode <mode>`: Deployment mode: `plain` or `roshanfer` (default: `plain`)

**Example:**
```bash
# Deploy in plain mode (direct service communication)
go run . deploy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode plain

# Deploy in roshanfer mode (with sidecars)
go run . deploy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode roshanfer
```

### Destroy

Removes the deployed benchmark from Kubernetes.

```bash
go run . destroy -o <output-dir> -c <deploy-config> -i <input-yaml> [--mode <mode>]
```

**Options:**
- `-o, --output <dir>`: Output directory containing generated artifacts (required)
- `-c, --config <file>`: Deployment config file (required)
- `-i, --input <path>`: Input YAML file path (required)
- `-m, --mode <mode>`: Deployment mode: `plain` or `roshanfer` (default: `plain`)

**Example:**
```bash
go run . destroy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode roshanfer
```

## Configuration Files

### Input YAML Configuration

The input YAML file describes your microservice architecture:

```yaml
name: my-benchmark          # Benchmark name
namespace: app              # Kubernetes namespace (default: "app")

services:
  frontend:
    type: frontend          # Service type: "frontend" or "backend"
    pre_repeat: 0          # CPU busy loop iterations before processing
    post_repeat: 100       # CPU busy loop iterations after processing
    response_size: 5000    # Response payload size in bytes
    http_endpoints:        # HTTP endpoints (required for frontend)
      - /api
      - /health
    cpu_cores: 2          # Number of CPU cores for the service
    sidecar_cpu_cores: 2  # Number of CPU cores for sidecar (default: 2)

  backend1:
    type: backend
    backend_repeat: 100    # CPU busy loop iterations for backend processing
    cpu_cores: 2
    sidecar_cpu_cores: 2

edges:
  - from: frontend         # Source service (must be frontend)
    to: backend1           # Target service (must be backend)
    method: SimpleCall     # gRPC method name
    pre_repeat: 50         # Optional: Override pre_repeat for this RPC
    post_repeat: 150       # Optional: Override post_repeat for this RPC

roshanfer:                 # Optional: Roshanfer configuration
  limits:
    frontend:
      service_limits:      # Service-level concurrency limits
        frontend: 5
    backend1:
      service_limits:
        backend1: 4
  slos:                    # Service Level Objectives (ms)
    /api: 36
    /health: 10
  ppm:                     # Packets Per Millisecond limits
    ingress: 2
    frontend: 150
    backend1: 120

cpu:
  start_core: 0           # Starting CPU core number
  total_cores: 32         # Total CPU cores available
```

**Service Types:**
- **frontend**: HTTP service that can call backend services via gRPC
- **backend**: gRPC service that processes requests

**Edge Configuration:**
- `from`: Must be a frontend service
- `to`: Must be a backend service
- `method`: gRPC method name (will be generated in protobuf)
- `pre_repeat` / `post_repeat`: Optional per-RPC overrides

**Roshanfer Configuration:**
- `limits`: Service-level concurrency limits
- `slos`: Service Level Objectives in milliseconds (endpoint → SLO)
- `ppm`: Packets Per Millisecond limits (service → PPM)

**Note:** The `roshanfer` section is optional. If present, it enables roshanfer mode generation. However, you can still deploy in either mode using the `--mode` flag.

### Deployment Config

The deployment config file (`deploy.yaml`) specifies deployment settings:

```yaml
namespace: app                    # Kubernetes namespace (default: "app")
kubeconfig: /path/to/kubeconfig  # Optional: Path to kubeconfig file
build_images: true               # Whether to build Docker images before deploying
```

**Fields:**
- `namespace`: Kubernetes namespace (defaults to "app" if not specified)
- `kubeconfig`: Optional path to kubeconfig file (uses default if not specified)
- `build_images`: Whether to build Docker images (default: false)

## Deployment Modes

### Plain Mode

In plain mode, services communicate directly without sidecars:

- Services listen on their primary ports (HTTPPort for frontend, GRPCPort for backend)
- Direct gRPC connections between frontend and backend services
- No sidecar containers
- No ingress deployment
- Environment variable `sidecar=false`

**Use Case:** Baseline performance testing, comparing against sidecar overhead

### Roshanfer Mode

In roshanfer mode, services communicate through sidecars:

- Services listen on upstream ports (sidecar handles ingress/egress)
- All traffic flows through sidecar containers
- Sidecar configuration from `roshanfer` section in YAML
- Ingress deployment included
- Environment variable `sidecar=true`

**Use Case:** Testing service mesh functionality, rate limiting, SLO enforcement

**Important:** The same generated artifacts can be deployed in either mode. The mode is selected at deployment time using the `--mode` flag.

## Output Structure

The generator creates the following directory structure:

```
<output-dir>/
├── protobuf/
│   ├── services.proto          # Protobuf definitions
│   └── services.pb.go          # Generated Go code
├── services/
│   ├── frontend/
│   │   └── main.go            # Frontend service implementation
│   └── backend/
│       └── main.go            # Backend service implementation
├── test/
│   ├── test/
│   │   ├── grpc-client.go      # gRPC client utilities
│   │   └── grpc-server.go     # gRPC server utilities
│   └── utils/
│       ├── ennvars.go          # Environment variable utilities
│       └── log.go              # Logging utilities
├── k8s/
│   ├── namespace.yaml          # Kubernetes namespace
│   ├── services.yaml          # Kubernetes Service objects
│   ├── <service>-deployment-plain.yaml      # Plain mode deployments
│   ├── <service>-deployment-roshanfer.yaml # Roshanfer mode deployments
│   ├── <service>-configmap.yaml             # Sidecar configmaps
│   └── ingress-deployment.yaml             # Ingress deployment (roshanfer only)
├── sidecar-configs/
│   ├── <service>.yaml         # Service sidecar configurations
│   └── ingress.yaml          # Ingress sidecar configuration
├── env/
│   ├── plain.env              # Environment variables for plain mode
│   └── sidecar.env            # Environment variables for roshanfer mode
├── go.mod                     # Go module file
└── go.sum                     # Go checksums
```

## Port Assignment

Each service is assigned multiple ports:

- **HTTPPort**: Frontend HTTP listening port (plain mode)
- **GRPCPort**: Backend gRPC listening port (plain mode)
- **IngressPort**: Sidecar ingress port (roshanfer mode)
- **EgressPort**: Sidecar egress port (roshanfer mode)
- **UpstreamPort**: Service listening port (roshanfer mode)

Ports are assigned sequentially starting from 3000, incrementing by 10 per service.

## CPU Core Assignment

CPU cores are assigned sequentially starting from `start_core`:

- Service cores: Assigned first
- Sidecar cores: Assigned after service cores
- Total cores per service: `cpu_cores + sidecar_cpu_cores`

**Example:** With `start_core: 0`, a service with 2 CPU cores and 2 sidecar cores:
- Service cores: 0, 1
- Sidecar cores: 2, 3
- Next service starts at core 4

## Examples

### Simple Frontend-Only Benchmark

```yaml
name: simple-frontend
namespace: app

services:
  app:
    type: frontend
    pre_repeat: 0
    post_repeat: 100
    response_size: 5000
    http_endpoints:
      - /app
    cpu_cores: 2
    sidecar_cpu_cores: 2

cpu:
  start_core: 0
  total_cores: 32
```

### Frontend with Multiple Backends

```yaml
name: multi-backend
namespace: app

services:
  frontend:
    type: frontend
    pre_repeat: 0
    post_repeat: 100
    response_size: 5000
    http_endpoints:
      - /api
    cpu_cores: 2
    sidecar_cpu_cores: 2

  backend1:
    type: backend
    backend_repeat: 100
    cpu_cores: 2
    sidecar_cpu_cores: 2

  backend2:
    type: backend
    backend_repeat: 300
    cpu_cores: 4
    sidecar_cpu_cores: 2

edges:
  - from: frontend
    to: backend1
    method: GetData
  - from: frontend
    to: backend2
    method: ProcessData

roshanfer:
  limits:
    frontend:
      service_limits:
        frontend: 5
    backend1:
      service_limits:
        backend1: 4
    backend2:
      service_limits:
        backend2: 7
  slos:
    /api: 36
  ppm:
    ingress: 2
    frontend: 150
    backend1: 120
    backend2: 120

cpu:
  start_core: 0
  total_cores: 32
```

### Complete Workflow

```bash
# 1. Generate artifacts
go run . generate -i example-benchmark.yaml -o testbench

# 2. Deploy in plain mode
go run . deploy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode plain

# 3. Test your benchmark...

# 4. Destroy plain deployment
go run . destroy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode plain

# 5. Deploy in roshanfer mode
go run . deploy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode roshanfer

# 6. Test with sidecars...

# 7. Destroy roshanfer deployment
go run . destroy -o testbench -c deploy.yaml -i example-benchmark.yaml --mode roshanfer
```

## Troubleshooting

### kubectl not found

If you're using k3s or microk8s, the tool will automatically detect and use `sudo k3s kubectl` or `sudo microk8s kubectl`. Make sure your cluster is installed and configured.

### Backend service not generated

Backend services are only generated if there are backend services defined in the YAML configuration. If all backend services are commented out, only the frontend service will be generated.

### Port conflicts

Ports are assigned automatically. If you need specific ports, modify the generated Kubernetes Service manifests.

### CPU core assignment errors

Ensure `total_cores` is sufficient for all services:
```
Total needed = sum(service.cpu_cores + service.sidecar_cpu_cores) for all services
```

### Missing protobuf files

Make sure `protoc` is installed and in your PATH. The tool will generate protobuf Go code automatically.

### Docker build failures

Ensure Docker is running and you have permission to build images. The tool builds images in the output directory.

## Notes

- The same generated artifacts can be deployed in either plain or roshanfer mode
- Mode selection happens at deployment time, not generation time
- If `roshanfer` section is missing from YAML, roshanfer mode deployments will still be generated but may lack sidecar configuration
- Frontend services must have at least one HTTP endpoint
- Edges can only connect frontend → backend (not backend → backend or frontend → frontend)
