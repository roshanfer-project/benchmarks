package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	envoyImage              = "envoyproxy/envoy:v1.32-latest"
	envoyIngressConcurrency = 1
	envoyMaxConcurrentStreams = 100
)

func generateEnvoyConfigs(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	entrySvc := pg.EntryMicroservice()
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return err
	}
	var configs []string
	for _, name := range svcNames {
		kn := k8sName(name)
		cfg := buildEnvoyServiceConfig(pg, prefix, name, kn, entrySvc)
		configs = append(configs, fmt.Sprintf("  %s.yaml: |\n%s", kn, indent(cfg, 4)))
	}
	ingressCfg := buildEnvoyIngressConfig(pg, prefix, entrySvc)
	configs = append(configs, fmt.Sprintf("  ingress.yaml: |\n%s", indent(ingressCfg, 4)))
	cm := `apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-configs
data:
` + strings.Join(configs, "\n")
	return os.WriteFile(filepath.Join(manifestsDir, "envoy-configs.yaml"), []byte(cm), 0644)
}

func buildEnvoyServiceConfig(pg *ParsedGraph, prefix, svcName, kn, entrySvc string) string {
	isEntry := svcName == entrySvc
	var b strings.Builder
	fmt.Fprintf(&b, "node:\n  id: %s\n  cluster: local\n\n", kn)
	b.WriteString(`stats_config:
  stats_matcher:
    reject_all: true
stats_flush_interval: 30s

static_resources:
  listeners:
`)
	b.WriteString(envoyInboundListener(isEntry))
	b.WriteString(envoyOutboundListener(pg, prefix, svcName, isEntry))
	b.WriteString("  clusters:\n")
	b.WriteString(envoyLocalAppCluster(isEntry))
	for _, cl := range envoyUpstreamClusters(pg, prefix, svcName) {
		b.WriteString(cl)
	}
	return b.String()
}

func envoyInboundListener(isEntry bool) string {
	h2 := ""
	if !isEntry {
		h2 = "\n          http2_protocol_options: {}"
	}
	return fmt.Sprintf(`  - name: listener_inbound
    address:
      socket_address: { address: 0.0.0.0, port_value: %d }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: inbound
          generate_request_id: false
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            virtual_hosts:
            - name: local
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: { cluster: local_app, timeout: 2s }%s

`, sidecarIngressPort, h2)
}

func envoyOutboundListener(pg *ParsedGraph, prefix, svcName string, isEntry bool) string {
	var routes strings.Builder
	seen := make(map[string]bool)
	for _, n := range pg.Services[svcName] {
		for _, targetID := range pg.Downstream(n.ID) {
			tn := pg.Nodes[targetID]
			key := tn.FullRPCName()
			if seen[key] {
				continue
			}
			seen[key] = true
			cluster := envoyClusterName(tn.Microservice)
			fmt.Fprintf(&routes, `              - match: { prefix: "/%s" }
                route: { cluster: %s, timeout: 2s }
`, key, cluster)
		}
	}
	routes.WriteString(`              - match: { prefix: "/" }
                direct_response: { status: 404, body: { inline_string: "no egress route" } }
`)
	h2Listener := "\n          http2_protocol_options: {}"
	return fmt.Sprintf(`  - name: listener_outbound
    address:
      socket_address: { address: 0.0.0.0, port_value: %d }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: outbound
          generate_request_id: false
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            virtual_hosts:
            - name: egress
              domains: ["*"]
              routes:
%s%s

`, sidecarEgressPort, routes.String(), h2Listener)
}

func envoyClusterName(microservice string) string {
	return k8sName(microservice) + "-service"
}

func envoyLocalAppCluster(isEntry bool) string {
	h2 := ""
	if !isEntry {
		h2 = "\n    http2_protocol_options: {}"
	}
	return fmt.Sprintf(`  - name: local_app
    connect_timeout: 2s
    type: STATIC
    load_assignment:
      cluster_name: local_app
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address: { address: 127.0.0.1, port_value: %d }%s

`, sidecarAppPort, h2)
}

func envoyUpstreamCluster(prefix, targetKn, clusterName string, h2Streams bool) string {
	h2 := ""
	if h2Streams {
		h2 = fmt.Sprintf(`
    http2_protocol_options:
      max_concurrent_streams: %d`, envoyMaxConcurrentStreams)
	}
	host := prefix + targetKn
	return fmt.Sprintf(`  - name: %s
    connect_timeout: 2s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: %s
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: %s
                port_value: %d%s

`, clusterName, clusterName, host, sidecarIngressPort, h2)
}

func envoyUpstreamClusters(pg *ParsedGraph, prefix, svcName string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range pg.Services[svcName] {
		for _, targetID := range pg.Downstream(n.ID) {
			tn := pg.Nodes[targetID]
			ms := tn.Microservice
			if seen[ms] {
				continue
			}
			seen[ms] = true
			targetKn := k8sName(ms)
			out = append(out, envoyUpstreamCluster(prefix, targetKn, envoyClusterName(ms), true))
		}
	}
	return out
}

func buildEnvoyIngressConfig(pg *ParsedGraph, prefix, entrySvc string) string {
	entryKn := k8sName(entrySvc)
	var b strings.Builder
	fmt.Fprintf(&b, "node:\n  id: ingress\n  cluster: local\n\n")
	b.WriteString(`stats_config:
  stats_matcher:
    reject_all: true
stats_flush_interval: 30s

static_resources:
  listeners:
`)
	for i, n := range pg.EntryInterfaces() {
		p := sidecarIngressBasePort + i
		b.WriteString(envoyIngressListener(p, n.Interface))
	}
	b.WriteString("  clusters:\n")
	b.WriteString(envoyUpstreamCluster(prefix, entryKn, "entry-service", false))
	return b.String()
}

func envoyIngressListener(port int, apiPath string) string {
	return fmt.Sprintf(`  - name: listener_%s
    address:
      socket_address: { address: 0.0.0.0, port_value: %d }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_%s
          generate_request_id: false
          use_remote_address: true
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            virtual_hosts:
            - name: entry
              domains: ["*"]
              routes:
              - match: { prefix: "/%s" }
                route: { cluster: entry-service, timeout: 2s }

`, apiPath, port, apiPath, apiPath)
}

func generateEnvoyEnv(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	lines := []string{"envoy=true", "", "PORT=2000"}
	for _, name := range svcNames {
		lines = append(lines, name+"_PORT=2000", name+"_EGRESS=localhost:4000")
	}
	lines = append(lines, "", "PROM_ADDR=prometheus-pushgateway:9091")
	k8sDir := filepath.Join(outDir, "k8s")
	return os.WriteFile(filepath.Join(k8sDir, "envoy.env"), []byte(strings.Join(lines, "\n")), 0644)
}

func generateAppEnvoyYaml(pg *ParsedGraph, benchmarkName string, svcNames []string, outDir string) error {
	prefix := benchmarkName + "-"
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	for _, name := range svcNames {
		kn := k8sName(name)
		imgName := prefix + kn
		cpu := pg.CPUForService(name)
		cpuStr := fmt.Sprintf("%d", cpu)
		concurrency := pg.SidecarCPUForService(name)
		envoyCpuStr := fmt.Sprintf("%d", concurrency)
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
  - name: envoy
    image: %s
    args:
    - "-c"
    - "/etc/envoy/envoy.yaml"
    - "--log-level"
    - "warn"
    - "--concurrency"
    - "%d"
    - "--disable-hot-restart"
    resources:
      requests:
        cpu: "%s"
      limits:
        cpu: "%s"
    volumeMounts:
    - name: config-volume
      mountPath: /etc/envoy/envoy.yaml
      subPath: %s.yaml
    ports:
    - containerPort: %d
    - containerPort: %d
  volumes:
  - name: config-volume
    configMap:
      name: envoy-configs
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
  - name: envoy-ingress
    port: %d
    targetPort: %d
    protocol: TCP
  - name: envoy-egress
    port: %d
    targetPort: %d
    protocol: TCP
`,
			kn, kn, imgName, cpuStr, cpuStr, cpuStr, benchmarkName, sidecarAppPort,
			envoyImage, concurrency, envoyCpuStr, envoyCpuStr, kn,
			sidecarIngressPort, sidecarEgressPort,
			prefix, kn, kn, kn,
			sidecarIngressPort, sidecarIngressPort,
			sidecarEgressPort, sidecarEgressPort)
		outPath := filepath.Join(manifestsDir, kn+"-envoy.yaml")
		if err := os.WriteFile(outPath, []byte(svcYaml), 0644); err != nil {
			return err
		}
	}
	return nil
}

func generateIngressEnvoyYaml(pg *ParsedGraph, benchmarkName string, outDir string) error {
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	envoyCpu := envoyIngressConcurrency
	nApis := pg.UserEntryCount()
	var portSpecs []string
	var svcPorts []string
	for i := 0; i < nApis; i++ {
		p := sidecarIngressBasePort + i
		portSpecs = append(portSpecs, fmt.Sprintf("    - containerPort: %d", p))
		svcPorts = append(svcPorts, fmt.Sprintf(`  - name: envoy-%d
    port: %d
    targetPort: %d
    nodePort: %d
    protocol: TCP`, p, p, p, p))
	}
	yaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: ingress
  labels:
    app: ingress
spec:
  restartPolicy: Never
  containers:
  - name: envoy
    image: %s
    args:
    - "-c"
    - "/etc/envoy/envoy.yaml"
    - "--log-level"
    - "warn"
    - "--concurrency"
    - "%d"
    - "--disable-hot-restart"
    resources:
      requests:
        cpu: "%d"
      limits:
        cpu: "%d"
    volumeMounts:
    - name: config-volume
      mountPath: /etc/envoy/envoy.yaml
      subPath: ingress.yaml
    ports:
%s
  volumes:
  - name: config-volume
    configMap:
      name: envoy-configs
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
`, envoyImage, envoyIngressConcurrency, envoyCpu, envoyCpu, strings.Join(portSpecs, "\n"),
		benchmarkName, strings.Join(svcPorts, "\n"))
	return os.WriteFile(filepath.Join(manifestsDir, "ingress-envoy.yaml"), []byte(yaml), 0644)
}
