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
	return generatePlainEnv(pg, benchmarkName, svcNames, outDir)
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
	var b bytes.Buffer
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
		b.WriteString(svcYaml)
		b.WriteString("\n---\n")
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
	b.WriteString(entryNodePort)
	if err := os.WriteFile(filepath.Join(manifestsDir, "entry.yaml"), []byte(entryNodePort), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(k8sDir, "app.yaml"), b.Bytes(), 0644)
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

{{range .Entries}}echo "Building {{.Name}}..."
docker build --build-arg SERVICE=services/{{.Name}} -f Dockerfile -t "${REGISTRY}/${BENCH}-{{.K8sName}}:${TAG}" .
{{end}}
echo "Pushing images..."
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
REGISTRY=${REGISTRY:-farzad1132}
TAG=${TAG:-latest}
BENCH={{.BenchmarkName}}
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TMP_DIR="k8s/tmp_apply"
mkdir -p "$TMP_DIR"
kubectl create configmap {{.BenchmarkName}}-config --from-env-file=k8s/plain.env --dry-run=client -o yaml > "$TMP_DIR/configmap.yaml"
kubectl apply -f "$TMP_DIR/configmap.yaml"

for SVC in {{range .K8sOrder}} {{.}} {{end}}; do
  sed "s|${BENCH}-${SVC}:latest|${REGISTRY}/${BENCH}-${SVC}:${TAG}|g" "k8s/manifests/${SVC}.yaml" > "$TMP_DIR/${SVC}.yaml"
  kubectl apply -f "$TMP_DIR/${SVC}.yaml"
  kubectl wait --for=condition=Ready pod -l app=${SVC} --timeout=${WAIT_TIMEOUT}s
done

kubectl apply -f k8s/manifests/entry.yaml
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
		parts = append(parts, fmt.Sprintf("kubectl delete pod -l app=%s --ignore-not-found --wait=false", kn))
		parts = append(parts, fmt.Sprintf("kubectl delete service -l app=%s --ignore-not-found", kn))
	}
	script := `#!/bin/bash
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"
` + strings.Join(parts, "\n") + `
kubectl delete configmap ` + benchmarkName + `-config --ignore-not-found
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
OUTPUT_DIR=${OUTPUT_DIR:-./logs}
mkdir -p "$OUTPUT_DIR"
for svc in ` + strings.TrimSpace(svcList) + `; do
  for pod in $(kubectl get pods -l app=$svc -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$pod" > "$OUTPUT_DIR/${pod}.log" 2>&1
  done
done
echo "Logs collected."
`
	return os.WriteFile(filepath.Join(outDir, "collect_logs.sh"), []byte(script), 0755)
}

func generateDockerfile(outDir string) error {
	df := `FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
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
