package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

func k8sName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "-")
}

// workloadBakeEntry is one runnable Go workload Docker image produced from the benchmark repo.
type workloadBakeEntry struct {
	ServicePath string // e.g. services/frontend — forward slashes
	K8sName     string // image suffix and k8s app name (lowercase, underscores → hyphens)
}

func workloadBakeEntries(pg *ParsedGraph, svcNames []string) []workloadBakeEntry {
	var entries []workloadBakeEntry
	for _, s := range svcNames {
		entries = append(entries, workloadBakeEntry{
			ServicePath: filepath.ToSlash(filepath.Join("services", s)),
			K8sName:     k8sName(s),
		})
	}
	ek := EntryGrpcK8s(pg)
	entries = append(entries, workloadBakeEntry{
		ServicePath: filepath.ToSlash(filepath.Join("services", ek)),
		K8sName:     ek,
	})
	return append(entries, workloadBakeEntry{
		ServicePath: filepath.ToSlash(filepath.Join("services", "rajomon-client")),
		K8sName:     "rajomon-client",
	})
}

func generateDockerignore(outDir string) error {
	di := `# Trims workload image build context; keep anything required for Go compile.
.git/
callgraph.json
k8s/
deploy.sh
destroy.sh
collect_logs.sh
docker-bake.hcl
run-plain.sh
run.sh
entry_path.txt
*.pdf
out/
out-*
`
	return os.WriteFile(filepath.Join(outDir, ".dockerignore"), []byte(di), 0644)
}

func generateWorkloadDockerfile(pg *ParsedGraph, svcNames []string, outDir string) error {
	entries := workloadBakeEntries(pg, svcNames)
	var b strings.Builder
	b.WriteString("# syntax=docker/dockerfile:1\n\n")
	b.WriteString("FROM golang:1.25-alpine AS deps\nWORKDIR /app\n")
	b.WriteString("COPY go.mod go.sum ./\n")
	b.WriteString("RUN --mount=type=cache,target=/go/pkg/mod \\\n\tgo mod download\n\n")
	b.WriteString("FROM deps AS builder\nWORKDIR /app\n")
	b.WriteString("COPY . .\n")
	b.WriteString("RUN --mount=type=cache,target=/go/pkg/mod \\\n\tgo mod tidy && ")
	for i, e := range entries {
		if i > 0 {
			b.WriteString(" && \\\n\t")
		}
		fmt.Fprintf(&b, "mkdir -p /out/%s && CGO_ENABLED=0 go build -ldflags=\"-s -w\" -o /out/%s/main ./%s",
			e.K8sName, e.K8sName, e.ServicePath)
	}
	b.WriteString("\n\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "FROM alpine:latest AS svc-%s\n", e.K8sName)
		b.WriteString("WORKDIR /root/\n")
		fmt.Fprintf(&b, "COPY --from=builder /out/%s/main .\n", e.K8sName)
		b.WriteString("COPY --from=builder /app/rajomon_init /rajomon_init\n")
		b.WriteString("COPY --from=builder /app/dagor_init /dagor_init\n")
		b.WriteString("EXPOSE 2000\n")
		b.WriteString("CMD [\"./main\"]\n\n")
	}
	return os.WriteFile(filepath.Join(outDir, "Dockerfile"), []byte(b.String()), 0644)
}

func generateDockerBakeHcl(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	entries := workloadBakeEntries(pg, svcNames)
	var b strings.Builder
	fmt.Fprintf(&b, `variable "REGISTRY" {
  default = "farzad1132"
}
variable "TAG" {
  default = "latest"
}
variable "BENCH" {
  default = "%s"
}

`, benchmarkName)
	b.WriteString("group \"default\" {\n  targets = [")
	for i, e := range entries {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", e.K8sName)
	}
	b.WriteString("]\n}\n\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "target %q {\n", e.K8sName)
		b.WriteString("  context = \".\"\n")
		b.WriteString("  dockerfile = \"Dockerfile\"\n")
		fmt.Fprintf(&b, "  target = \"svc-%s\"\n", e.K8sName)
		fmt.Fprintf(&b, "  tags = [\"${REGISTRY}/${BENCH}-%s:${TAG}\"]\n", e.K8sName)
		b.WriteString("}\n\n")
	}
	return os.WriteFile(filepath.Join(outDir, "docker-bake.hcl"), []byte(b.String()), 0644)
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
	if err := generateEnvoyConfigs(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateEnvoyEnv(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateAppEnvoyYaml(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateIngressEnvoyYaml(pg, benchmarkName, outDir); err != nil {
		return err
	}
	if err := generateRajomonEnv(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateDagorEnv(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateAppGrpcYaml(pg, benchmarkName, svcNames, outDir); err != nil {
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

func generateRajomonEnv(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	ek := EntryGrpcK8s(pg)
	entryMS := pg.EntryMicroservice()
	lines := []string{
		"deployment=" + benchmarkName,
		"rajomon=true",
		"",
		"EntryGRPCPort=" + fmt.Sprintf("%d", port),
		"ClientPort=2007",
		"EntryGRPCAddr=" + prefix + ek + ":" + fmt.Sprintf("%d", port),
		"",
	}
	for _, name := range svcNames {
		var podKn string
		if name == entryMS {
			podKn = ek
		} else {
			podKn = k8sName(name)
		}
		lines = append(lines, name+"_ADDR="+prefix+podKn+":"+fmt.Sprintf("%d", port))
	}
	lines = append(lines, "", "PROM_ADDR=prometheus-pushgateway:9091")
	return os.WriteFile(filepath.Join(outDir, "k8s", "rajomon.env"), []byte(strings.Join(lines, "\n")), 0644)
}

func generateDagorEnv(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	ek := EntryGrpcK8s(pg)
	entryMS := pg.EntryMicroservice()
	lines := []string{
		"deployment=" + benchmarkName,
		"dagor=true",
		"",
		"EntryGRPCPort=" + fmt.Sprintf("%d", port),
		"ClientPort=2007",
		"EntryGRPCAddr=" + prefix + ek + ":" + fmt.Sprintf("%d", port),
		"",
	}
	for _, name := range svcNames {
		var podKn string
		if name == entryMS {
			podKn = ek
		} else {
			podKn = k8sName(name)
		}
		lines = append(lines, name+"_ADDR="+prefix+podKn+":"+fmt.Sprintf("%d", port))
	}
	lines = append(lines, "", "PROM_ADDR=prometheus-pushgateway:9091")
	return os.WriteFile(filepath.Join(outDir, "k8s", "dagor.env"), []byte(strings.Join(lines, "\n")), 0644)
}

func generateAppGrpcYaml(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	entryMS := pg.EntryMicroservice()
	ek := EntryGrpcK8s(pg)
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return err
	}
	var docs []string
	for _, name := range svcNames {
		var kn string
		if name == entryMS {
			kn = ek
		} else {
			kn = k8sName(name)
		}
		imgName := prefix + kn
		cpu := pg.CPUForService(name)
		cpuStr := fmt.Sprintf("%d", cpu)
		doc := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  labels:
    app: %s
spec:
  restartPolicy: Never
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
  - name: grpc
    port: %d
    targetPort: %d
    protocol: TCP
`, kn, kn, imgName, cpuStr, cpuStr, cpuStr, benchmarkName, port,
			prefix, kn, kn, kn, port, port)
		docs = append(docs, doc)
	}
	rcCPU := pg.CPUForService(entryMS) + 1
	rcStr := fmt.Sprintf("%d", rcCPU)
	rcImg := prefix + "rajomon-client"
	rcDoc := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: rajomon-client
  labels:
    app: rajomon-client
spec:
  restartPolicy: Never
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
    - containerPort: 2007
---
apiVersion: v1
kind: Service
metadata:
  name: %sentry
  labels:
    app: rajomon-client
spec:
  type: NodePort
  selector:
    app: rajomon-client
  ports:
  - name: http
    port: 2007
    targetPort: 2007
    nodePort: 3000
    protocol: TCP
`, rcImg, rcStr, rcStr, rcStr, benchmarkName, prefix)
	docs = append(docs, rcDoc)
	// --- between chunks: each chunk is Pod+Service; without this, the next Pod merges into the prior Service doc.
	return os.WriteFile(filepath.Join(manifestsDir, "app-grpc.yaml"), []byte(strings.Join(docs, "\n---\n")), 0644)
}

func rajomonDeployK8sOrder(pg *ParsedGraph) []string {
	dep := deploymentOrder(pg)
	entryMS := pg.EntryMicroservice()
	ek := EntryGrpcK8s(pg)
	var order []string
	for _, s := range dep {
		if s == entryMS {
			continue
		}
		order = append(order, k8sName(s))
	}
	order = append(order, ek)
	order = append(order, "rajomon-client")
	return order
}

func rajomonImageK8sNames(pg *ParsedGraph) []string {
	svcNames := sortedServices(pg)
	entryMS := pg.EntryMicroservice()
	var names []string
	for _, s := range svcNames {
		if s == entryMS {
			names = append(names, EntryGrpcK8s(pg))
		} else {
			names = append(names, k8sName(s))
		}
	}
	names = append(names, "rajomon-client")
	return names
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

const rwgPhasesSh = `#!/bin/bash
# Resolve RWG rate/duration lists. Uses RWG_RATES/RWG_DURATIONS when set; else legacy 2-phase.

resolve_rwg_phases() {
    local base=$1 rate=$2 duration=$3
    if [ -n "$RWG_RATES" ] && [ -n "$RWG_DURATIONS" ]; then
        RESOLVED_RATES="$RWG_RATES"
        RESOLVED_DURATIONS="$RWG_DURATIONS"
    else
        RESOLVED_RATES="$base,$rate"
        RESOLVED_DURATIONS="2,$duration"
    fi
}
`

const rwgPhasesSnippet = `
# Resolve RWG phases (legacy 2-phase or RWG_RATES/RWG_DURATIONS from exec)
# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/rwg_phases.sh"
resolve_rwg_phases "$BASE" "$RATE" "$DURATION"
`

func buildIngressConfig(pg *ParsedGraph, prefix, entrySvc string) string {
	entryKn := k8sName(entrySvc)
	entryHost := prefix + entryKn
	numThreads := pg.UserEntryCount()
	var routing, mapping strings.Builder
	for i, n := range pg.EntryInterfaces() {
		// Ingress routing slo: tighter budget before frontend (95% of graph entry slo, nearest int).
		ingressSLO := (*n.SLO*95 + 50) / 100
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
	_ = os.Remove(filepath.Join(outDir, "push.sh"))
	if err := generateDockerignore(outDir); err != nil {
		return err
	}
	if err := generateWorkloadDockerfile(pg, svcNames, outDir); err != nil {
		return err
	}
	if err := generateDockerBakeHcl(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateBuildScript(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateDeployScript(pg, benchmarkName, outDir); err != nil {
		return err
	}
	if err := generateDestroyScript(pg, benchmarkName, svcNames, outDir); err != nil {
		return err
	}
	if err := generateCollectLogsScript(pg, svcNames, outDir); err != nil {
		return err
	}
	if err := generateWrapperScripts(pg, outDir); err != nil {
		return err
	}
	return nil
}

func generateBuildScript(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	entries := workloadBakeEntries(pg, svcNames)
	tmpl := `#!/bin/bash
set -euo pipefail
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

if [ "${SKIP_SIDECAR_BUILD:-}" != "1" ] && [ -n "$SIDECAR_DIR" ]; then
  echo "Building sidecar..."
  (cd "$SIDECAR_DIR" && ./build.sh Release)
  docker build -f "$SIDECAR_DIR/Dockerfile" -t "${REGISTRY}/sidecar-sidecar:${TAG}" "$SIDECAR_DIR"
fi

ENVOY_STATS_DIR=""
D="$ROOT_DIR"
while [ -n "$D" ] && [ "$D" != "/" ]; do
  if [ -d "$D/callgraph-framework/envoy-stats-exporter" ]; then
    ENVOY_STATS_DIR="$D/callgraph-framework/envoy-stats-exporter"
    break
  fi
  D="$(cd "$D/.." && pwd)"
done
if [ "${SKIP_SIDECAR_BUILD:-}" = "1" ]; then
  if [ -z "$ENVOY_STATS_DIR" ]; then
    echo "build.sh: callgraph-framework/envoy-stats-exporter not found (walk up from $ROOT_DIR)" >&2
    exit 1
  fi
  echo "Building envoy-stats-exporter..."
  REGISTRY="${REGISTRY}" TAG="${TAG}" bash "$ENVOY_STATS_DIR/build.sh"
fi

echo "Building workload images (docker buildx bake)..."
REGISTRY="${REGISTRY}" TAG="${TAG}" BENCH="${BENCH}" docker buildx bake -f docker-bake.hcl

echo "Pushing images..."
PUSH_IMAGES=()
if [ "${SKIP_SIDECAR_BUILD:-}" != "1" ] && [ -n "$SIDECAR_DIR" ]; then
  PUSH_IMAGES+=("${REGISTRY}/sidecar-sidecar:${TAG}")
fi
{{range .Entries}}
PUSH_IMAGES+=("${REGISTRY}/${BENCH}-{{.K8sName}}:${TAG}")
{{end}}
printf '%s\0' "${PUSH_IMAGES[@]}" | xargs -0 -P "$(nproc)" -n1 docker push

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
if { [ "$MODE" = "rajomon" ] || [ "$MODE" = "dagor" ]; } && [ -n "$ARG2" ]; then
  echo "deploy.sh: rajomon and dagor modes do not take a second argument" >&2
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

kubectl_wait_ready_or_fail() {
  local app=$1
  local to=$2
  if kubectl wait --for=condition=Ready pod -l "app=${app}" --timeout="${to}s"; then
    return 0
  fi
  echo "=== deploy.sh: kubectl wait failed for app=${app} (timeout=${to}s) ===" >&2
  kubectl get pods -l "app=${app}" -o wide >&2 || true
  kubectl describe pod -l "app=${app}" >&2 || true
  local p
  while IFS= read -r p; do
    [ -z "$p" ] && continue
    echo "=== logs ${p} (current) ===" >&2
    kubectl logs "$p" --all-containers=true --tail=200 >&2 || true
    echo "=== logs ${p} (previous) ===" >&2
    kubectl logs "$p" --all-containers=true --previous --tail=200 >&2 || true
  done < <(kubectl get pods -l "app=${app}" -o name 2>/dev/null)
  exit 1
}

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
  select(.name == "sidecar") |= (
    (.env // []) as $e |
    ($e | map(select(.name != "GLOG_v" and .name != "GLOG_vmodule"))) as $base |
    ([{"name":"GLOG_v","value":strenv(GV_VAL)},{"name":"GLOG_vmodule","value":strenv(VM_VAL)}]
      | map(select(.value != ""))) as $add |
    .env = $base + $add
  )
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
  cp k8s/manifests/sidecar-configs.yaml "$TMP_DIR/sidecar-configs.yaml"
  if [ -n "${SIDECAR_OVER_COMMIT:-}" ]; then
    echo "deploy.sh: SIDECAR_OVER_COMMIT=${SIDECAR_OVER_COMMIT} (patch all over_commitment in sidecar-configs)"
    export SIDECAR_OVER_COMMIT
    perl -i -pe 's/(over_commitment:\s*)[\d.]+/${1}$ENV{SIDECAR_OVER_COMMIT}/g' "$TMP_DIR/sidecar-configs.yaml"
    awk -v want="$SIDECAR_OVER_COMMIT" '
      /over_commitment:/ {
        if ($2 != want) {
          print "deploy.sh: ERROR expected over_commitment " want " got " $2 > "/dev/stderr"
          exit 1
        }
      }
    ' "$TMP_DIR/sidecar-configs.yaml"
  fi
  kubectl apply -f "$TMP_DIR/sidecar-configs.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  for SVC in {{range $i, $e := .K8sOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "k8s/manifests/${SVC}-sidecar.yaml" | \
    sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" > "$TMP_DIR/${SVC}-sidecar.yaml"
    if [ "$SIDECAR_DEBUG" = "1" ]; then
      sidecar_debug_patch_workload_yaml "$TMP_DIR/${SVC}-sidecar.yaml"
    fi
    kubectl apply -f "$TMP_DIR/${SVC}-sidecar.yaml"
    kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
  done

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  if [ "$SIDECAR_DEBUG" = "1" ]; then
    sidecar_debug_patch_workload_yaml "$TMP_DIR/ingress.yaml"
  fi
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl_wait_ready_or_fail ingress 30
elif [ "$MODE" = "envoy" ]; then
  ENVOY_STATS_IMAGE="${REGISTRY}/envoy-stats-exporter:${TAG}"

  cat k8s/envoy.env > "$TMP_DIR/envoy_merged.env"
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file="$TMP_DIR/envoy_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/envoy-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  for SVC in {{range $i, $e := .K8sOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    sed -e "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" \
        -e "s|envoy-stats-exporter:latest|${ENVOY_STATS_IMAGE}|g" \
        "k8s/manifests/${SVC}-envoy.yaml" > "$TMP_DIR/${SVC}-envoy.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in {{range $i, $e := .K8sOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    (
      kubectl apply -f "$TMP_DIR/${SVC}-envoy.yaml"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (envoy): one or more workloads failed readiness" >&2
    exit 1
  fi

  sed -e "s|envoy-stats-exporter:latest|${ENVOY_STATS_IMAGE}|g" \
      k8s/manifests/ingress-envoy.yaml > "$TMP_DIR/ingress-envoy.yaml"
  kubectl apply -f "$TMP_DIR/ingress-envoy.yaml"
  kubectl_wait_ready_or_fail ingress 30
elif [ "$MODE" = "rajomon" ]; then
  PRICE_UPDATE_RATE=${priceUpdateRate}
  LATENCY_THRESHOLD=${latencyThreshold}
  TOKEN_UPDATE_RATE=${tokenUpdateRate}
  PRICE_STEP=${priceStep}
  TOKEN_UPDATE_STEP=${tokenUpdateStep}
  echo "Using Rajomon config:"
  echo "  priceUpdateRate=$PRICE_UPDATE_RATE latencyThreshold=$LATENCY_THRESHOLD tokenUpdateRate=$TOKEN_UPDATE_RATE priceStep=$PRICE_STEP tokenUpdateStep=$TOKEN_UPDATE_STEP"
  cat k8s/rajomon.env > "$TMP_DIR/rajomon_merged.env"
  echo "" >> "$TMP_DIR/rajomon_merged.env"
  echo "priceUpdateRate=$PRICE_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
  echo "latencyThreshold=$LATENCY_THRESHOLD" >> "$TMP_DIR/rajomon_merged.env"
  echo "tokenUpdateRate=$TOKEN_UPDATE_RATE" >> "$TMP_DIR/rajomon_merged.env"
  echo "priceStep=$PRICE_STEP" >> "$TMP_DIR/rajomon_merged.env"
  echo "tokenUpdateStep=$TOKEN_UPDATE_STEP" >> "$TMP_DIR/rajomon_merged.env"
  K8S_NS=${K8S_NS:-$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)}
  K8S_NS=${K8S_NS:-default}
  sed -i "s|=${BENCH}-\([^=]*\):2000|=${BENCH}-\1.${K8S_NS}.svc.cluster.local:2000|g" "$TMP_DIR/rajomon_merged.env"
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file="$TMP_DIR/rajomon_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  cp k8s/manifests/app-grpc.yaml "$TMP_DIR/app-grpc.yaml"
  for IMG in {{range $i, $e := .RajomonImgK8s}}{{if $i}} {{end}}{{$e}}{{end}}; do
    sed -i "s|${BENCH}-${IMG}:latest|${REGISTRY}/${BENCH}-${IMG}:${TAG}|g" "$TMP_DIR/app-grpc.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in {{range $i, $e := .RajomonDeployOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    (
      kubectl apply -f "$TMP_DIR/app-grpc.yaml" -l app="${SVC}"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (rajomon): one or more workloads failed readiness" >&2
    exit 1
  fi
elif [ "$MODE" = "dagor" ]; then
  cat k8s/dagor.env > "$TMP_DIR/dagor_merged.env"
  echo "" >> "$TMP_DIR/dagor_merged.env"
  if [ -n "$Alpha" ]; then
    echo "Alpha=$Alpha" >> "$TMP_DIR/dagor_merged.env"
  fi
  if [ -n "$Beta" ]; then
    echo "Beta=$Beta" >> "$TMP_DIR/dagor_merged.env"
  fi
  K8S_NS=${K8S_NS:-$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)}
  K8S_NS=${K8S_NS:-default}
  sed -i "s|=${BENCH}-\([^=]*\):2000|=${BENCH}-\1.${K8S_NS}.svc.cluster.local:2000|g" "$TMP_DIR/dagor_merged.env"
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file="$TMP_DIR/dagor_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  cp k8s/manifests/app-grpc.yaml "$TMP_DIR/app-grpc.yaml"
  for IMG in {{range $i, $e := .RajomonImgK8s}}{{if $i}} {{end}}{{$e}}{{end}}; do
    sed -i "s|${BENCH}-${IMG}:latest|${REGISTRY}/${BENCH}-${IMG}:${TAG}|g" "$TMP_DIR/app-grpc.yaml"
  done
  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in {{range $i, $e := .RajomonDeployOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    (
      kubectl apply -f "$TMP_DIR/app-grpc.yaml" -l app="${SVC}"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (dagor): one or more workloads failed readiness" >&2
    exit 1
  fi
else
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl_wait_ready_or_fail prometheus-pushgateway 60
  kubectl_wait_ready_or_fail prometheus 60

  deploy_fail=0
  declare -a deploy_pids=()
  for SVC in {{range $i, $e := .K8sOrder}}{{if $i}} {{end}}{{$e}}{{end}}; do
    (
      sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
      kubectl apply -f "$TMP_DIR/${SVC}.yaml"
      kubectl_wait_ready_or_fail "${SVC}" "${WAIT_TIMEOUT}"
    ) &
    deploy_pids+=($!)
  done
  for pid in "${deploy_pids[@]}"; do
    wait "$pid" || deploy_fail=1
  done
  if [ "$deploy_fail" -ne 0 ]; then
    echo "deploy.sh (plain): one or more workloads failed readiness" >&2
    exit 1
  fi

  kubectl apply -f k8s/manifests/entry.yaml
fi

rm -rf "$TMP_DIR"
echo "Deploy complete."
`
	t, _ := template.New("").Parse(tmpl)
	var b bytes.Buffer
	t.Execute(&b, map[string]interface{}{
		"BenchmarkName":      benchmarkName,
		"K8sOrder":           k8sOrder,
		"RajomonImgK8s":      rajomonImageK8sNames(pg),
		"RajomonDeployOrder": rajomonDeployK8sOrder(pg),
	})
	return os.WriteFile(filepath.Join(outDir, "deploy.sh"), b.Bytes(), 0755)
}

func generateDestroyScript(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	ek := EntryGrpcK8s(pg)

	var b strings.Builder
	b.WriteString(`#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

fail=0
declare -a pids=()
`)
	if len(svcNames) > 0 {
		quoted := make([]string, 0, len(svcNames))
		for _, s := range svcNames {
			quoted = append(quoted, strconv.Quote(k8sName(s)))
		}
		b.WriteString(fmt.Sprintf(`for kn in %s; do
  (
    kubectl delete pod -l app="$kn" --ignore-not-found --wait=true
    kubectl delete service -l app="$kn" --ignore-not-found
  ) &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

`, strings.Join(quoted, " ")))
	}
	b.WriteString(`kubectl delete configmap ` + benchmarkName + `-config --ignore-not-found
kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
kubectl delete configmap prometheus-config --ignore-not-found
if [ "$MODE" = "sidecar" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap sidecar-configs --ignore-not-found
fi
if [ "$MODE" = "envoy" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap envoy-configs --ignore-not-found
fi
if [ "$MODE" = "rajomon" ] || [ "$MODE" = "dagor" ]; then
  kubectl delete pod -l app=rajomon-client --ignore-not-found --wait=true
  kubectl delete service -l app=rajomon-client --ignore-not-found
  kubectl delete service ` + benchmarkName + `-entry --ignore-not-found
  kubectl delete pod -l app=` + ek + ` --ignore-not-found --wait=true
  kubectl delete service -l app=` + ek + ` --ignore-not-found
fi
echo "Destroy complete."
exit "$fail"
`)
	return os.WriteFile(filepath.Join(outDir, "destroy.sh"), []byte(b.String()), 0755)
}

func generateCollectLogsScript(pg *ParsedGraph, svcNames []string, outDir string) error {
	var svcList string
	for _, s := range svcNames {
		svcList += k8sName(s) + " "
	}
	svcList += EntryGrpcK8s(pg) + " rajomon-client "
	trimmed := strings.TrimSpace(svcList)
	script := `#!/bin/bash
MODE=${1:-${SYSTEM:-plain}}
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
declare -a log_pids=()
for svc in ` + trimmed + `; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
      if [ "$MODE" = "sidecar" ]; then
        kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
      fi
      if [ "$MODE" = "envoy" ]; then
        kubectl logs "$pod" -c envoy > "$OUTPUT_DIR/${pod}-envoy.log" 2>&1
      fi
    ) &
    log_pids+=($!)
  done
done
for pid in "${log_pids[@]}"; do wait "$pid" || true; done

if [ "$MODE" = "sidecar" ]; then
  declare -a ing_pids=()
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" -c sidecar > "$OUTPUT_DIR/${pod}-sidecar.log" 2>&1
    ) &
    ing_pids+=($!)
  done
  for pid in "${ing_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "envoy" ]; then
  declare -a ing_pids=()
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}'); do
    (
      kubectl logs "$pod" -c envoy > "$OUTPUT_DIR/${pod}-envoy.log" 2>&1
    ) &
    ing_pids+=($!)
  done
  for pid in "${ing_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "envoy" ]; then
  ENVOY_METRICS_DIR=${ENVOY_METRICS_DIR:-./metrics/envoy}
  mkdir -p "$ENVOY_METRICS_DIR"
  declare -a stats_pids=()
  for svc in ` + trimmed + ` ingress; do
    for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
      [ -z "$pod" ] && continue
      app=$(kubectl get pod "$pod" -o jsonpath='{.metadata.labels.app}' 2>/dev/null)
      [ -z "$app" ] && continue
      ( kubectl cp "$pod:/tmp/envoy_stats.csv" "$ENVOY_METRICS_DIR/${app}.csv" -c envoy-stats 2>/dev/null || true ) &
      stats_pids+=($!)
    done
  done
  for pid in "${stats_pids[@]}"; do wait "$pid" || true; done
fi

if [ "$MODE" = "sidecar" ] && [ "${COLLECT_SIDECAR_NANOLOG:-}" = "1" ]; then
  declare -a cp_pids=()
  for svc in ` + trimmed + `; do
    for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
      ( kubectl cp "$pod:/compressedLog" "$OUTPUT_DIR/${pod}-sidecar.clog" -c sidecar 2>/dev/null || true ) &
      cp_pids+=($!)
    done
  done
  for pod in $(kubectl get pods -l app=ingress -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
    ( kubectl cp "$pod:/compressedLog" "$OUTPUT_DIR/${pod}-ingress-sidecar.clog" -c sidecar 2>/dev/null || true ) &
    cp_pids+=($!)
  done
  for pid in "${cp_pids[@]}"; do wait "$pid" || true; done
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

	if err := os.WriteFile(filepath.Join(outDir, "rwg_phases.sh"), []byte(rwgPhasesSh), 0755); err != nil {
		return err
	}

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
` + rwgPhasesSnippet + `
API=$5
output_dir="$6/out-$API.csv"
address="${TARGET_ADDR:-192.168.1.100}"

if [ "$protocol" == "grpc" ]; then
    echo "GRPC is not supported"
    exit 1
else
` + apiCasesSidecar + `
    echo "url: $url"
    "$RWG_BINARY" run --url $url -d exp -D $RESOLVED_DURATIONS -r $RESOLVED_RATES -w 5000 -o $output_dir -t 15
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
` + rwgPhasesSnippet + `
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
        "$RWG_BINARY" run --url $url -d exp -D $RESOLVED_DURATIONS -r $RESOLVED_RATES -w 10000 -o $output_dir -t 30 --ignore-errors
        exit 0
    else
        "$RWG_BINARY" run --url $url -d exp -D $RESOLVED_DURATIONS -r $RESOLVED_RATES -w 10000 -o $output_dir -t 30
        exit "$?"
    fi
fi
`
	return os.WriteFile(filepath.Join(outDir, "run-plain.sh"), []byte(runPlainSh), 0755)
}
