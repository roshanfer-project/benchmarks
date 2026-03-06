package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	entryNode := pg.Nodes[pg.EntryNodeID]
	path := "/" + entryNode.Interface
	return os.WriteFile(filepath.Join(outDir, "entry_path.txt"), []byte(path), 0644)
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
	k8sDir := filepath.Join(outDir, "k8s")
	return os.WriteFile(filepath.Join(k8sDir, "plain.env"), []byte(strings.Join(lines, "\n")), 0644)
}

const (
	sidecarAppPort     = 2000
	sidecarIngressPort = 3000
	sidecarEgressPort  = 4000
)

func generateSidecarConfigs(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	entryNode := pg.Nodes[pg.EntryNodeID]
	if entryNode.SLO == nil || entryNode.Priority == nil {
		return fmt.Errorf("entry interface %s must have slo and priority in call graph (required for sidecar mode)", entryNode.ID)
	}
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
	ingressCfg := buildIngressConfig(pg, prefix, entrySvc, entryNode)
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
		} else {
			b.WriteString(fmt.Sprintf("  %s:\n", key))
		}
	}

	b.WriteString(fmt.Sprintf("ring_size: %d\nbuffer_count: %d\nbuffer_size: %d\nnum_threads: %d\n", ringSize, bufCount, bufSize, numThreads))
	b.WriteString(fmt.Sprintf("egress_listener_port: %d\ningress_listener_port: %d\ningress_upstream_host: localhost\ningress_upstream_port: %d\n", sidecarEgressPort, sidecarIngressPort, sidecarAppPort))
	b.WriteString("report_latency: True\n")
	b.WriteString(fmt.Sprintf("name: %s\n", kn))

	cpuCount := pg.CPUForService(svcName)
	overCommitment := 1.0
	if isEntry {
		b.WriteString("is_frontend: True\nfrontend_pool_connections: 200\n")
	}
	b.WriteString(fmt.Sprintf("cpu_count: %d\nover_commitment: %.1f\n", cpuCount, overCommitment))
	return b.String()
}

func buildIngressConfig(pg *ParsedGraph, prefix, entrySvc string, entryNode *Node) string {
	entryKn := k8sName(entrySvc)
	entryHost := prefix + entryKn
	slo, priority := *entryNode.SLO, *entryNode.Priority
	numThreads := pg.UserEntryCount()
	return fmt.Sprintf(`routing:
  %s:
    upstream:
      host: %s
      port: %d
    slo: %d
    priority: %d
mapping:
  %s:
    downstreams:
      - %s
    listen_port: %d
ring_size: 4000
buffer_count: 16384
buffer_size: 10000
num_threads: %d
ingress_pool_connections: 200
egress_listener_port: %d
ingress_listener_port: 4000
ingress_upstream_host: localhost
ingress_upstream_port: %d
is_ingress: True
is_frontend: False
report_latency: True
name: ingress
`, entryNode.Interface, entryHost, sidecarIngressPort, slo, priority,
		entryNode.Interface, entryNode.Interface, sidecarIngressPort,
		numThreads,
		sidecarIngressPort, sidecarAppPort)
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
	var parts []string
	for _, name := range svcNames {
		kn := k8sName(name)
		imgName := prefix + kn
		cpu := pg.CPUForService(name)
		cpuStr := fmt.Sprintf("%d", cpu)
		sidecarCPU := pg.SidecarCPUForService(name)
		sidecarCpuK8s := sidecarCPU * 2
		sidecarCpuStr := fmt.Sprintf("%d", sidecarCpuK8s)
		podYaml := fmt.Sprintf(`apiVersion: v1
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
---
`,
			kn, kn, imgName, cpuStr, cpuStr, cpuStr, benchmarkName, sidecarAppPort,
			kn, sidecarCpuStr, sidecarCpuStr, kn,
			sidecarIngressPort, sidecarEgressPort, sidecarIngressPort, sidecarEgressPort,
			prefix, kn, kn, kn,
			sidecarIngressPort, sidecarIngressPort, sidecarIngressPort, sidecarIngressPort,
			sidecarEgressPort, sidecarEgressPort, sidecarEgressPort, sidecarEgressPort)
		parts = append(parts, podYaml)
	}
	joined := strings.Join(parts, "\n")
	joined = strings.TrimSuffix(joined, "\n---\n")
	return os.WriteFile(filepath.Join(manifestsDir, "app-sidecar.yaml"), []byte(joined), 0644)
}

func generateIngressYaml(pg *ParsedGraph, benchmarkName string, outDir string) error {
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	cpu := pg.UserEntryCount() * 2
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
    - containerPort: 3000
    - containerPort: 3000
      protocol: UDP
    - containerPort: 4000
    - containerPort: 4000
      protocol: UDP
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
  - name: sidecar-3000-tcp
    port: 3000
    targetPort: 3000
    nodePort: 3000
    protocol: TCP
  - name: sidecar-3000-udp
    port: 3000
    targetPort: 3000
    nodePort: 3000
    protocol: UDP
  - name: sidecar-4000-tcp
    port: 4000
    targetPort: 4000
    protocol: TCP
  - name: sidecar-4000-udp
    port: 4000
    targetPort: 4000
    protocol: UDP
`, cpu, cpu, benchmarkName)
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
	return generateDockerfile(outDir)
}

func generateBuildScript(benchmarkName string, svcNames []string, outDir string) error {
	type svcEntry struct {
		Name   string
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

if [ -d "../sidecar" ]; then
  echo "Building sidecar..."
  (cd ../sidecar && ./build.sh Release)
  docker build -f ../sidecar/Dockerfile -t "${REGISTRY}/sidecar-sidecar:${TAG}" ../sidecar
fi

{{range .Entries}}echo "Building {{.Name}}..."
docker build --build-arg SERVICE=services/{{.Name}} -f Dockerfile -t "${REGISTRY}/${BENCH}-{{.K8sName}}:${TAG}" .
{{end}}
echo "Pushing images..."
if [ -d "../sidecar" ]; then
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
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH={{.BenchmarkName}}
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"

if [ "$MODE" = "sidecar" ]; then
  cat k8s/sidecar.env > "$TMP_DIR/sidecar_merged.env"
  echo "" >> "$TMP_DIR/sidecar_merged.env"
  echo "queuing_export=${queuing_export}" >> "$TMP_DIR/sidecar_merged.env"
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file="$TMP_DIR/sidecar_merged.env" --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"
  kubectl apply -f k8s/manifests/sidecar-configs.yaml

  kubectl apply -f k8s/manifests/prometheus.yaml
  kubectl wait --for=condition=ready pod -l app=prometheus-pushgateway --timeout=60s || true
  kubectl wait --for=condition=ready pod -l app=prometheus --timeout=60s || true

  cp k8s/manifests/app-sidecar.yaml "$TMP_DIR/app-sidecar.yaml"
  sed -i "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  {{range .K8sOrder}}sed -i "s|${BENCH}-{{.}}:latest|${REGISTRY}/${BENCH}-{{.}}:${TAG}|g" "$TMP_DIR/app-sidecar.yaml"
  {{end}}
  kubectl apply -f "$TMP_DIR/app-sidecar.yaml"
  {{range .K8sOrder}}kubectl wait --for=condition=Ready pod -l app={{.}} --timeout=${WAIT_TIMEOUT}s
  {{end}}

  sed "s|sidecar-sidecar:latest|${REGISTRY}/sidecar-sidecar:${TAG}|g" k8s/manifests/ingress.yaml > "$TMP_DIR/ingress.yaml"
  kubectl apply -f "$TMP_DIR/ingress.yaml"
  kubectl wait --for=condition=Ready pod -l app=ingress --timeout=30s || true
else
  kubectl create configmap {{.BenchmarkName}}-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
  kubectl apply -f "$TMP_DIR/configmap.yaml"

  for SVC in {{range .K8sOrder}} {{.}} {{end}}; do
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
if [ "$MODE" = "sidecar" ]; then
  kubectl delete pod -l app=ingress --ignore-not-found --wait=true
  kubectl delete service -l app=ingress --ignore-not-found
  kubectl delete configmap sidecar-configs --ignore-not-found
  kubectl delete deployment prometheus prometheus-pushgateway --ignore-not-found --wait=true
  kubectl delete service prometheus prometheus-pushgateway prometheus-external --ignore-not-found
  kubectl delete configmap prometheus-config --ignore-not-found
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
