package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

func k8sName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "-")
}

func GenerateK8s(pg *ParsedGraph, benchmarkName string, registry string, outDir string) error {
	svcNames := sortedServices(pg)
	if err := generateAppYaml(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := writeEntryPath(pg, outDir); err != nil {
		return err
	}
	if err := generatePlainEnv(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateSidecarConfigs(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateSidecarEnv(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateAppSidecarYaml(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateIngressYaml(pg, benchmarkName, outDir); err != nil {
		return err
	}
	return generatePrometheusYaml(outDir)
}

func writeEntryPath(pg *ParsedGraph, outDir string) error {
	var paths []string
	for _, n := range pg.EntryInterfaces() {
		paths = append(paths, "/"+n.Interface)
	}
	return os.WriteFile(filepath.Join(outDir, "entry_path.txt"), []byte(strings.Join(paths, "\n")), 0644)
}

func generateAppYaml(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	entrySvc := pg.EntryMicroservice()
	entryPodName := k8sName(entrySvc)
	k8sDir := filepath.Join(outDir, "k8s")
	manifestsDir := filepath.Join(k8sDir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return err
	}
	for _, name := range svcNames {
		podName := k8sName(name)
		imgName := prefix + podName
		cpu := pg.CPUForService(name)
		cpuStr := fmt.Sprintf("%d", cpu)
		svcYaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  labels:
    app: %s
spec:
  containers:
  - name: app
    image: %s:latest
    resources:
      requests:
        cpu: "%s"
      limits:
        cpu: "%s"
    env:
    - name: GOMAXPROCS
      value: "%s"
    envFrom:
    - configMapRef:
        name: %s-config
    ports:
    - containerPort: %d
---
apiVersion: v1
kind: Service
metadata:
  name: %s%s
  labels:
    app: %s
spec:
  selector:
    app: %s
  ports:
  - name: http
    port: %d
    targetPort: %d
    protocol: TCP
`,
			podName, podName, imgName, cpuStr, cpuStr, cpuStr, benchmarkName, port,
			prefix, podName, podName, podName, port, port)
		if err := os.WriteFile(filepath.Join(manifestsDir, podName+".yaml"), []byte(svcYaml), 0644); err != nil {
			return err
		}
	}
	entryNodePort := fmt.Sprintf(`---
apiVersion: v1
kind: Service
metadata:
  name: %s
  labels:
    app: %s
spec:
  type: NodePort
  selector:
    app: %s
  ports:
  - name: http
    port: %d
    targetPort: %d
    nodePort: 3000
    protocol: TCP
`,
		prefix+"entry", entryPodName, entryPodName, port, port)
	if err := os.WriteFile(filepath.Join(manifestsDir, "entry.yaml"), []byte(entryNodePort), 0644); err != nil {
		return err
	}
	return nil
}

func generatePlainEnv(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	lines := []string{"plain=true", ""}
	for _, name := range svcNames {
		podName := k8sName(name)
		svcDNS := prefix + podName + ":" + fmt.Sprintf("%d", port)
		lines = append(lines, name+"_ADDR="+svcDNS)
	}
	lines = append(lines, "PORT="+fmt.Sprintf("%d", port))
	lines = append(lines, "", "PROM_ADDR=prometheus-pushgateway:9091")
	k8sDir := filepath.Join(outDir, "k8s")
	return os.WriteFile(filepath.Join(k8sDir, "plain.env"), []byte(strings.Join(lines, "\n")), 0644)
}

const (
	sidecarAppPort     = 2000
	sidecarIngressPort = 3000
	sidecarEgressPort  = 4000
)

func generateSidecarConfigs(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	entrySvc := pg.EntryMicroservice()
	k8sDir := filepath.Join(outDir, "k8s")
	manifestsDir := filepath.Join(k8sDir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return err
	}

	var configs []string
	for _, name := range svcNames {
		kn := k8sName(name)
		cfg := buildSidecarServiceConfig(pg, prefix, name, kn, entrySvc)
		configs = append(configs, fmt.Sprintf("  %s.yaml: |\n%s", kn, indent(cfg, 4)))
	}
	ingressCfg := buildIngressConfig(pg, prefix, entrySvc)
	configs = append(configs, fmt.Sprintf("  ingress.yaml: |\n%s", indent(ingressCfg, 4)))

	cm := `apiVersion: v1
kind: ConfigMap
metadata:
  name: sidecar-configs
data:
` + strings.Join(configs, "\n")
	return os.WriteFile(filepath.Join(manifestsDir, "sidecar-configs.yaml"), []byte(cm), 0644)
}

func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func buildSidecarServiceConfig(pg *ParsedGraph, prefix, svcName, kn, entrySvc string) string {
	var b strings.Builder
	var ringSize, bufCount, bufSize int
	isEntry := svcName == entrySvc
	if isEntry {
		ringSize, bufCount, bufSize = 1000, 4096, 80000
	} else {
		ringSize, bufCount, bufSize = 1000, 1024, 80000
	}
	numThreads := pg.SidecarCPUForService(svcName)

	// routing: for each downstream
	nodes := pg.Services[svcName]
	hasDownstream := false
	for _, n := range nodes {
		targets := pg.Downstream(n.ID)
		if len(targets) > 0 {
			hasDownstream = true
			break
		}
	}
	if hasDownstream {
		b.WriteString("routing:\n")
		seen := make(map[string]bool)
		for _, n := range nodes {
			for _, targetID := range pg.Downstream(n.ID) {
				tn := pg.Nodes[targetID]
				key := tn.FullRPCName()
				if seen[key] {
					continue
				}
				seen[key] = true
				targetKn := k8sName(tn.Microservice)
				host := prefix + targetKn
				b.WriteString(fmt.Sprintf("  %s:\n    upstream:\n      host: %s\n      port: %d\n", key, host, sidecarIngressPort))
			}
		}
	}

	// mapping: entry service uses interface name (f1); non-entry uses FullRPCName
	b.WriteString("mapping:\n")
	for _, n := range nodes {
		targets := pg.Downstream(n.ID)
		key := n.Interface
		if !isEntry {
			key = n.FullRPCName()
		}
		if len(targets) > 0 {
			b.WriteString(fmt.Sprintf("  %s:\n    downstreams:\n", key))
			for _, targetID := range targets {
				tn := pg.Nodes[targetID]
				b.WriteString(fmt.Sprintf("      - %s\n", tn.FullRPCName()))
			}
			if len(targets) > 1 && pg.NodeUsesParallelFanout(n.ID) {
				b.WriteString("    pfanout: true\n")
			} else if len(targets) > 1 && pg.NodeUsesWeightedFanout(n.ID) {
				b.WriteString("    dfanout: true\n")
			}
		} else {
			b.WriteString(fmt.Sprintf("  %s:\n", key))
		}
	}

	b.WriteString(fmt.Sprintf("ring_size: %d\nbuffer_count: %d\nbuffer_size: %d\nnum_threads: %d\n", ringSize, bufCount, bufSize, numThreads))
	b.WriteString(fmt.Sprintf("egress_listener_port: %d\ningress_listener_port: %d\ningress_upstream_host: localhost\ningress_upstream_port: %d\n", sidecarEgressPort, sidecarIngressPort, sidecarAppPort))
	b.WriteString("report_latency: True\n")
	b.WriteString(fmt.Sprintf("name: %s\n", kn))

	cpuCount := pg.CPUForService(svcName)
	overCommitment := pg.OverCommitmentForService(svcName)
	if isEntry {
		b.WriteString(fmt.Sprintf("is_frontend: True\nfrontend_pool_connections: %d\n", pg.EffectiveConnectionPoolSize()))
	}
	b.WriteString(fmt.Sprintf("cpu_count: %d\nover_commitment: %.1f\n", cpuCount, overCommitment))
	return b.String()
}

const sidecarIngressBasePort = 3000

func buildIngressConfig(pg *ParsedGraph, prefix, entrySvc string) string {
	entryKn := k8sName(entrySvc)
	entryHost := prefix + entryKn
	numThreads := pg.UserEntryCount()
	var routing, mapping strings.Builder
	for i, n := range pg.EntryInterfaces() {
		// Ingress gets 80% of graph SLO (floor); remainder budget for path after ingress.
		ingressSLO := (*n.SLO * 8) / 10
		priority := *n.Priority
		listenPort := sidecarIngressBasePort + i
		routing.WriteString(fmt.Sprintf("  %s:\n    upstream:\n      host: %s\n      port: %d\n    slo: %d\n    priority: %d\n", n.Interface, entryHost, sidecarIngressPort, ingressSLO, priority))
		mapping.WriteString(fmt.Sprintf("  %s:\n    downstreams:\n      - %s\n    listen_port: %d\n", n.Interface, n.Interface, listenPort))
	}
	return fmt.Sprintf(`routing:
%s
mapping:
%s
ring_size: 4000
buffer_count: 16384
buffer_size: 10000
num_threads: %d
ingress_pool_connections: %d
egress_listener_port: %d
ingress_listener_port: 4000
ingress_upstream_host: localhost
ingress_upstream_port: %d
is_ingress: True
is_frontend: False
report_latency: True
name: ingress
`, strings.TrimSuffix(routing.String(), "\n"), strings.TrimSuffix(mapping.String(), "\n"),
		numThreads, pg.EffectiveConnectionPoolSize(), sidecarIngressPort, sidecarAppPort)
}

func generateSidecarEnv(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	lines := []string{"sidecar=true", "", "PORT=2000"}
	for _, name := range svcNames {
		lines = append(lines, name+"_PORT=2000", name+"_EGRESS=localhost:4000")
	}
	lines = append(lines, "", "PROM_ADDR=prometheus-pushgateway:9091")
	k8sDir := filepath.Join(outDir, "k8s")
	return os.WriteFile(filepath.Join(k8sDir, "sidecar.env"), []byte(strings.Join(lines, "\n")), 0644)
}

func generateAppSidecarYaml(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return err
	}
	for _, name := range svcNames {
		kn := k8sName(name)
		imgName := prefix + kn
		cpu := pg.CPUForService(name)
		cpuStr := fmt.Sprintf("%d", cpu)
		sidecarCPU := pg.SidecarCPUForService(name)
		sidecarCpuK8s := sidecarCPU * 2
		sidecarCpuStr := fmt.Sprintf("%d", sidecarCpuK8s)
		svcYaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  labels:
    app: %s
spec:
  containers:
  - name: app
    image: %s:latest
    resources:
      requests:
        cpu: "%s"
      limits:
        cpu: "%s"
    env:
    - name: GOMAXPROCS
      value: "%s"
    envFrom:
    - configMapRef:
        name: %s-config
    ports:
    - containerPort: %d
  - name: sidecar
    image: sidecar-sidecar:latest
    env:
    - name: PROC_NAME
      value: "%s-sidecar"
    - name: GLOG_logtostderr
      value: "1"
    resources:
      requests:
        cpu: "%s"
      limits:
        cpu: "%s"
    volumeMounts:
    - name: config-volume
      mountPath: /config.yaml
      subPath: %s.yaml
    ports:
    - containerPort: %d
    - containerPort: %d
    - containerPort: %d
      protocol: UDP
    - containerPort: %d
      protocol: UDP
  volumes:
  - name: config-volume
    configMap:
      name: sidecar-configs
---
apiVersion: v1
kind: Service
metadata:
  name: %s%s
  labels:
    app: %s
spec:
  selector:
    app: %s
  ports:
  - name: sidecar-ingress-tcp
    port: %d
    targetPort: %d
    protocol: TCP
  - name: sidecar-ingress-udp
    port: %d
    targetPort: %d
    protocol: UDP
  - name: sidecar-egress-tcp
    port: %d
    targetPort: %d
    protocol: TCP
  - name: sidecar-egress-udp
    port: %d
    targetPort: %d
    protocol: UDP
`,
			kn, kn, imgName, cpuStr, cpuStr, cpuStr, benchmarkName, sidecarAppPort,
			kn, sidecarCpuStr, sidecarCpuStr, kn,
			sidecarIngressPort, sidecarEgressPort, sidecarIngressPort, sidecarEgressPort,
			prefix, kn, kn, kn,
			sidecarIngressPort, sidecarIngressPort, sidecarIngressPort, sidecarIngressPort,
			sidecarEgressPort, sidecarEgressPort, sidecarEgressPort, sidecarEgressPort)
		svcYaml = strings.TrimSuffix(svcYaml, "\n")
		outPath := filepath.Join(manifestsDir, kn+"-sidecar.yaml")
		if err := os.WriteFile(outPath, []byte(svcYaml), 0644); err != nil {
			return err
		}
	}
	return nil
}

func generateIngressYaml(pg *ParsedGraph, benchmarkName string, outDir string) error {
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	cpu := pg.UserEntryCount() * 2
	nApis := pg.UserEntryCount()
	var portSpecs []string
	for i := 0; i < nApis; i++ {
		p := sidecarIngressBasePort + i
		portSpecs = append(portSpecs, fmt.Sprintf("    - containerPort: %d", p))
		portSpecs = append(portSpecs, fmt.Sprintf("    - containerPort: %d\n      protocol: UDP", p))
	}
	portSpecs = append(portSpecs, "    - containerPort: 4000", "    - containerPort: 4000\n      protocol: UDP")
	var svcPorts []string
	for i := 0; i < nApis; i++ {
		p := sidecarIngressBasePort + i
		svcPorts = append(svcPorts, fmt.Sprintf(`  - name: sidecar-%d-tcp
    port: %d
    targetPort: %d
    nodePort: %d
    protocol: TCP
  - name: sidecar-%d-udp
    port: %d
    targetPort: %d
    nodePort: %d
    protocol: UDP`, p, p, p, p, p, p, p, p))
	}
	svcPorts = append(svcPorts, `  - name: sidecar-4000-tcp
    port: 4000
    targetPort: 4000
    protocol: TCP
  - name: sidecar-4000-udp
    port: 4000
    targetPort: 4000
    protocol: UDP`)
	yaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: ingress
  labels:
    app: ingress
spec:
  restartPolicy: Never
  containers:
  - name: sidecar
    image: sidecar-sidecar:latest
    env:
    - name: PROC_NAME
      value: "ingress-sidecar"
    - name: GLOG_logtostderr
      value: "1"
    resources:
      requests:
        cpu: "%d"
      limits:
        cpu: "%d"
    volumeMounts:
    - name: config-volume
      mountPath: /config.yaml
      subPath: ingress.yaml
    ports:
%s
  volumes:
  - name: config-volume
    configMap:
      name: sidecar-configs
---
apiVersion: v1
kind: Service
metadata:
  name: %s-ingress
  labels:
    app: ingress
spec:
  type: NodePort
  selector:
    app: ingress
  ports:
%s
`, cpu, cpu, strings.Join(portSpecs, "\n"), benchmarkName, strings.Join(svcPorts, "\n"))
	return os.WriteFile(filepath.Join(manifestsDir, "ingress.yaml"), []byte(yaml), 0644)
}

func generatePrometheusYaml(outDir string) error {
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	yaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  labels:
    app: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:v2.50.1
        resources:
          requests:
            cpu: "1"
          limits:
            cpu: "1"
        args:
          - "--config.file=/etc/prometheus/prometheus.yml"
          - "--storage.tsdb.path=/prometheus"
        ports:
          - containerPort: 9090
        volumeMounts:
          - name: prometheus-config
            mountPath: /etc/prometheus
      volumes:
        - name: prometheus-config
          configMap:
            name: prometheus-config
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  labels:
    app: prometheus
spec:
  selector:
    app: prometheus
  ports:
    - port: 9090
      targetPort: 9090
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 1s
    scrape_configs:
      - job_name: 'pushgateway'
        honor_labels: true
        static_configs:
          - targets: ['prometheus-pushgateway:9091']
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-pushgateway
  labels:
    app: prometheus-pushgateway
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus-pushgateway
  template:
    metadata:
      labels:
        app: prometheus-pushgateway
    spec:
      containers:
      - name: prometheus-pushgateway
        image: prom/pushgateway:v1.6.0
        resources:
          requests:
            cpu: "1"
          limits:
            cpu: "1"
        ports:
          - containerPort: 9091
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus-pushgateway
  labels:
    app: prometheus-pushgateway
spec:
  selector:
    app: prometheus-pushgateway
  ports:
    - port: 9091
      targetPort: 9091
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus-external
  labels:
    app: prometheus
spec:
  type: NodePort
  selector:
    app: prometheus
  ports:
    - port: 9090
      targetPort: 9090
`
	return os.WriteFile(filepath.Join(manifestsDir, "prometheus.yaml"), []byte(yaml), 0644)
}

func GenerateScripts(pg *ParsedGraph, benchmarkName string, outDir string) error {
	svcNames := sortedServices(pg)
	if err := generateBuildScript(benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generatePushScript(benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateDeployScript(pg, benchmarkName, outDir); err != nil {
		return err
	}
	if err := generateDestroyScript(benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateCollectLogsScript(svcNames, outDir); err != nil {
		return err
	}
	if err := generateWrapperScripts(pg, outDir); err != nil {
		return err
	}
	return generateDockerfile(outDir)
}

func generateBuildScript(benchmarkName string, svcNames []string, outDir string) error {
	type svcEntry struct {
		Name    string
		K8sName string
	}
	var entries []svcEntry
	for _, s := range svcNames {
		entries = append(entries, svcEntry{s, k8sName(s)})
	}
	tmpl := `#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
STATUS_FILE=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
BENCH={{.BenchmarkName}}

if [ -n "$STATUS_FILE" ]; then
  STATUS_DIR=$(dirname "$STATUS_FILE")
  STATUS_BASE=$(basename "$STATUS_FILE")
  mkdir -p "$STATUS_DIR"
  STATUS_DIR=$(cd "$STATUS_DIR" && pwd)
  STATUS_FILE="${STATUS_DIR}/${STATUS_BASE}"
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

# Find sidecar dir (benchmarks/sidecar) - works for nested benchmarks like tests/one-service
SIDECAR_DIR=""
D="$ROOT_DIR"
while [ -n "$D" ] && [ "$D" != "/" ]; do
  if [ -d "$D/sidecar" ]; then
    SIDECAR_DIR="$D/sidecar"
    break
  fi
  D="$(cd "$D/.." && pwd)"
done

if [ -n "$SIDECAR_DIR" ]; then
  echo "Building sidecar..."
  (cd "$SIDECAR_DIR" && ./build.sh Release)
  docker build -f "$SIDECAR_DIR/Dockerfile" -t "${REGISTRY}/sidecar-sidecar:${TAG}" "$SIDECAR_DIR"
fi

{{range .Entries}}echo "Building {{.Name}}..."
docker build --build-arg SERVICE=services/{{.Name}} -f Dockerfile -t "${REGISTRY}/${BENCH}-{{.K8sName}}:${TAG}" .
{{end}}
echo "Pushing images..."
if [ -n "$SIDECAR_DIR" ]; then
  docker push "${REGISTRY}/sidecar-sidecar:${TAG}"
fi
{{range .Entries}}docker push "${REGISTRY}/${BENCH}-{{.K8sName}}:${TAG}"
{{end}}
if [ -n "$STATUS_FILE" ]; then
  mkdir -p "$(dirname "$STATUS_FILE")"
  touch "$STATUS_FILE"
fi
echo "Build complete."
`
	t, _ := template.New("").Parse(tmpl)
	var b bytes.Buffer
	t.Execute(&b, map[string]interface{}{
		"BenchmarkName": benchmarkName,
		"Entries":       entries,
	})
	return os.WriteFile(filepath.Join(outDir, "build.sh"), b.Bytes(), 0755)
}

func generatePushScript(benchmarkName string, svcNames []string, outDir string) error {
	var k8sNames []string
	for _, s := range svcNames {
		k8sNames = append(k8sNames, k8sName(s))
	}
	tmpl := `#!/bin/bash
set -e
TAG=${1:-${TAG:-latest}}
REGISTRY=${REGISTRY:-farzad1132}
BENCH={{.BenchmarkName}}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

echo "Pushing images..."
{{range .K8sNames}}docker push "${REGISTRY}/${BENCH}-{{.}}:${TAG}"
{{end}}
echo "Push complete."
`
	t, _ := template.New("").Parse(tmpl)
	var b bytes.Buffer
	t.Execute(&b, map[string]interface{}{
		"BenchmarkName": benchmarkName,
		"K8sNames":      k8sNames,
	})
	return os.WriteFile(filepath.Join(outDir, "push.sh"), b.Bytes(), 0755)
}

func generateDeployScript(pg *ParsedGraph, benchmarkName string, outDir string) error {
	depOrder := deploymentOrder(pg)
	var k8sOrder []string
	for _, s := range depOrder {
		k8sOrder = append(k8sOrder, k8sName(s))
	}
	tmpl := `#!/bin/bash
set -e
MODE=${1:-${SYSTEM:-plain}}
ARG2=${2:-}
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH={{.BenchmarkName}}
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

if [ "$MODE" = "plain" ] && [ "$ARG2" = "debug" ]; then
  echo "deploy.sh: debug only with sidecar; use ./deploy.sh sidecar debug" >&2
  exit 1
fi
if [ "$MODE" = "sidecar" ] && [ -n "$ARG2" ] && [ "$ARG2" != "debug" ]; then
  echo "deploy.sh: unknown second argument: $ARG2 (expected: debug)" >&2
  exit 1
fi
SIDECAR_DEBUG=0
if [ "$MODE" = "sidecar" ] && [ "$ARG2" = "debug" ]; then
  SIDECAR_DEBUG=1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"

sidecar_debug_require_yq() {
  command -v yq >/dev/null 2>&1 || {
    echo "deploy.sh sidecar debug needs mikefarah yq v4: https://github.com/mikefarah/yq" >&2
    exit 1
  }
}

sidecar_debug_merge_glog_file() {
  [ ! -f k8s/sidecar-debug-glog.env ] && return 0
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in ''|\#*) continue ;; esac
    k="${line%%=*}"
    v="${line#*=}"
    case "$k" in
      SIDECAR_GLOG_V) [ -z "${SIDECAR_GLOG_V+x}" ] && export SIDECAR_GLOG_V="$v" ;;
      SIDECAR_GLOG_VMODULE) [ -z "${SIDECAR_GLOG_VMODULE+x}" ] && export SIDECAR_GLOG_VMODULE="$v" ;;
    esac
  done < k8s/sidecar-debug-glog.env
}

sidecar_debug_patch_workload_yaml() {
  local f=$1
  yq eval-all 'select(.kind == "Pod") |= (.spec.restartPolicy = "Never")' -i "$f"
  export GV_VAL="${SIDECAR_GLOG_V:-}"
  export VM_VAL="${SIDECAR_GLOG_VMODULE:-}"
  if [ -n "$GV_VAL" ] || [ -n "$VM_VAL" ]; then
    yq eval-all '
select(.kind == "Pod") |= (.spec.containers |= map(
  if .name == "sidecar" then
    (.env // []) as $e |
    ($e | map(select(.name != "GLOG_v" and .name != "GLOG_vmodule"))) as $base |
    .env = $base
      + (if (strenv(GV_VAL) | length) > 0 then [{"name":"GLOG_v","value":strenv(GV_VAL)}] else [] end)
      + (if (strenv(VM_VAL) | length) > 0 then [{"name":"GLOG_vmodule","value":strenv(VM_VAL)}] else [] end)
  else .
  end
))' -i "$f"
  fi
}

if [ "$MODE" = "sidecar" ]; then
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_require_yq
    sidecar_debug_merge_glog_file
  fi
  cat k8s/sidecar.env > "$TMP_DIR/sidecar_merged.env"
  echo "" >> "$TMP_DIR/sidecar_merged.env"
  echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/sidecar-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  for SVC in {{range $i, $e := .K8sOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "k8s/manifests/${SVC}-sidecar.yaml" | \
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" > "$TMP_DIR/${SVC}-sidecar.yaml"
    if [ "$SIDECAR_DEBUG" = "1" ]; then
      sidecar_debug_patch_workload_yaml "$TMP_DIR/${SVC}-sidecar.yaml"
    fi
    kubectl apply -f "$TMP_DIR/${SVC}-sidecar.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_patch_workload_yaml "$TMP_DIR/ingress.yaml"
  fi
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl wait --for=condition=Ready pod -l app=ingress --timeout=30s || true
else
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  for SVC in {{range $i, $e := .K8sOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
    kubectl apply -f "$TMP_DIR/${SVC}.yaml"
    kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
  done

  kubectl apply -f k8s/manifests/entry.yaml
fi

rm -rf "$TMP_DIR"
echo "Deploy complete."
`
	t, _ := template.New("").Parse(tmpl)
	var b bytes.Buffer
	t.Execute(&b, map[string]interface{}{
		"BenchmarkName": benchmarkName,
		"K8sOrder":      k8sOrder,
	})
	return os.WriteFile(filepath.Join(outDir, "deploy.sh"), b.Bytes(), 0755)
}

func generateDestroyScript(benchmarkName string, svcNames []string, outDir string) error {
	var parts []string
	for _, s := range svcNames {
		kn := k8sName(s)
		parts = append(parts, fmt.Sprintf("kubectl delete pod -l app=%s --ignore-not-found --wait=true", kn))
		parts = append(parts, fmt.Sprintf("kubectl delete service -l app=%s --ignore-not-found", kn))
	}
	script := `#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"
` + strings.Join(parts, "\n") + `
kubectl delete configmap ` + benchmarkName + `-config --ignore-not-found
kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap prometheus-config --ignore-not-found
if [ "$MODE" = "sidecar" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap sidecar-configs --ignore-not-found
fi
echo "Destroy complete."
`
	return os.WriteFile(filepath.Join(outDir, "destroy.sh"), []byte(script), 0755)
}

func generateCollectLogsScript(svcNames []string, outDir string) error {
	var svcList string
	for _, s := range svcNames {
		svcList += k8sName(s) + " "
	}
	script := `#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
for svc in ` + strings.TrimSpace(svcList) + `; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
    if [ "$MODE" = "sidecar" ]; then
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
    fi
  done
done
if [ "$MODE" = "sidecar" ]; then
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
  done
fi
echo "Logs collected."
`
	return os.WriteFile(filepath.Join(outDir, "collect_logs.sh"), []byte(script), 0755)
}

func generateWrapperScripts(pg *ParsedGraph, outDir string) error {
	entryNodes := pg.EntryInterfaces()
	if len(entryNodes) == 0 {
		return fmt.Errorf("no entry interfaces found")
	}
	apis := make([]string, 0, len(entryNodes))
	for _, n := range entryNodes {
		apis = append(apis, n.Interface)
	}
	sort.Strings(apis)

	// run.sh (sidecar): each API on its own port
	apiCasesSidecar := ""
	for i, api := range apis {
		port := sidecarIngressBasePort + i
		if i == 0 {
			apiCasesSidecar += fmt.Sprintf("    if [ \"$API\" = %q ]; then\n        url=\"http://$address:%d/%s\"\n", api, port, api)
		} else {
			apiCasesSidecar += fmt.Sprintf("    elif [ \"$API\" = %q ]; then\n        url=\"http://$address:%d/%s\"\n", api, port, api)
		}
	}
	apiCasesSidecar += "    else\n        echo \"Unknown API: $API\"\n        exit 1\n    fi"

	// run-plain.sh: all APIs on port 3000, path-based
	apiCasesPlain := ""
	for i, api := range apis {
		if i == 0 {
			apiCasesPlain += fmt.Sprintf("    if [ \"$API\" = %q ]; then\n        url=\"http://$address:3000/%s\"\n", api, api)
		} else {
			apiCasesPlain += fmt.Sprintf("    elif [ \"$API\" = %q ]; then\n        url=\"http://$address:3000/%s\"\n", api, api)
		}
	}
	apiCasesPlain += "    else\n        echo \"Unknown API: $API\"\n        exit 1\n    fi"

	runSh := `#!/bin/bash
# Usage: $0 PROTOCOL BASE RATE DURATION API OUTPUT_DIR
if [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ -z "$5" ] || [ -z "$6" ]; then
  echo "Error: Missing required arguments"
  echo "Usage: $0 PROTOCOL BASE RATE DURATION API OUTPUT_DIR"
  exit 1
fi

protocol=$1
BASE=$2
RATE=$3
DURATION=$4
API=$5
output_dir="$6/out-$API.csv"
address="${TARGET_ADDR:-192.168.1.100}"

if [ "$protocol" == "grpc" ]; then
    echo "GRPC is not supported"
    exit 1
else
` + apiCasesSidecar + `
    echo "url: $url"
    "$RWG_BINARY" run --url $url -d exp -D 2,$DURATION -r $BASE,$RATE -w 5000 -o $output_dir -t 15
    exit "$?"
fi
`
	if err := os.WriteFile(filepath.Join(outDir, "run.sh"), []byte(runSh), 0755); err != nil {
		return err
	}

	runPlainSh := `#!/bin/bash
# Usage: $0 PROTOCOL BASE RATE DURATION API OUTPUT_DIR [--ignore-errors]
if [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ -z "$5" ] || [ -z "$6" ]; then
  echo "Error: Missing required arguments"
  echo "Usage: $0 PROTOCOL BASE RATE DURATION API OUTPUT_DIR [--ignore-errors]"
  exit 1
fi

if [ "$7" == "--ignore-errors" ]; then
    ignore_errors=true
else
    ignore_errors=false
fi

protocol=$1
BASE=$2
RATE=$3
DURATION=$4
API=$5
output_dir="$6/out-$API.csv"
address="${TARGET_ADDR:-192.168.1.100}"

if [ "$protocol" == "grpc" ]; then
    echo "GRPC is not supported"
    exit 1
else
` + apiCasesPlain + `
    echo "url: $url"
    if [ "$ignore_errors" = true ]; then
        "$RWG_BINARY" run --url $url -d exp -D 2,$DURATION -r $BASE,$RATE -w 10000 -o $output_dir -t 30 --ignore-errors
        exit 0
    else
        "$RWG_BINARY" run --url $url -d exp -D 2,$DURATION -r $BASE,$RATE -w 10000 -o $output_dir -t 30
        exit "$?"
    fi
fi
`
	return os.WriteFile(filepath.Join(outDir, "run-plain.sh"), []byte(runPlainSh), 0755)
}

func generateDockerfile(outDir string) error {
	df := `FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy
ARG SERVICE
RUN go build -o /app/main ./${SERVICE}

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 2000
CMD ["./main"]
`
	return os.WriteFile(filepath.Join(outDir, "Dockerfile"), []byte(df), 0644)
}
