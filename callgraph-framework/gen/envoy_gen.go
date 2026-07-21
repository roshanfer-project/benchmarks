package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	envoyImage                = "envoyproxy/envoy:v1.32-latest"
	envoyIngressConcurrency   = 1
	envoyMaxConcurrentStreams = 100
	envoyAdminPort            = 9901
	envoyRouteTimeout         = "10s"
	envoyConnectTimeout       = "2s"
	envoyHighCircuitBreakers  = 100000
)

const envoyStatsContainerYAML = `  - name: envoy-stats
    image: envoy-stats-exporter:latest
    env:
    - name: APP_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.labels['app']
    - name: POLL_INTERVAL_MS
      value: "200"
    - name: OUTPUT_PATH
      value: "/tmp/envoy_stats.csv"
    resources:
      requests:
        memory: 16Mi
        cpu: 10m
      limits:
        memory: 32Mi
        cpu: 50m
`

func envoyServiceStatsConfig() string {
	return `stats_config:
  stats_matcher:
    inclusion_list:
      patterns:
      - prefix: "http.inbound_"
stats_flush_interval: 5s

`
}

func envoyIngressStatsConfig() string {
	return `stats_config:
  stats_matcher:
    inclusion_list:
      patterns:
      - prefix: "http.ingress_"
stats_flush_interval: 5s

`
}

func envoyAdminBlock() string {
	return fmt.Sprintf(`admin:
  address:
    socket_address: { address: 0.0.0.0, port_value: %d }

`, envoyAdminPort)
}

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
	ingressCfg := buildEnvoyIngressConfig(pg, prefix, entrySvc, 0, "ROUND_ROBIN", false)
	configs = append(configs, fmt.Sprintf("  ingress.yaml: |\n%s", indent(ingressCfg, 4)))
	cm := `apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-configs
data:
` + strings.Join(configs, "\n")
	return os.WriteFile(filepath.Join(manifestsDir, "envoy-configs.yaml"), []byte(cm), 0644)
}

func generatePlainLbEnvoyConfigs(pg *ParsedGraph, benchmarkName string, outDir string) error {
	prefix := benchmarkName + "-"
	entrySvc := pg.EntryMicroservice()
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return err
	}
	ingressCfg := buildEnvoyIngressConfig(pg, prefix, entrySvc, sidecarAppPort, "LEAST_REQUEST", true)
	cm := `apiVersion: v1
kind: ConfigMap
metadata:
  name: plain-lb-envoy-configs
data:
  ingress.yaml: |
` + indent(ingressCfg, 4)
	return os.WriteFile(filepath.Join(manifestsDir, "plain-lb-envoy-configs.yaml"), []byte(cm), 0644)
}

func buildEnvoyServiceConfig(pg *ParsedGraph, prefix, svcName, kn, entrySvc string) string {
	isEntry := svcName == entrySvc
	var b strings.Builder
	fmt.Fprintf(&b, "node:\n  id: %s\n  cluster: local\n\n", kn)
	b.WriteString(envoyServiceStatsConfig())
	b.WriteString(envoyAdminBlock())
	b.WriteString("static_resources:\n  listeners:\n")
	b.WriteString(envoyInboundListeners(pg, svcName, isEntry))
	b.WriteString(envoyOutboundListener(pg, prefix, svcName, isEntry))
	b.WriteString("  clusters:\n")
	b.WriteString(envoyLocalAppCluster(isEntry))
	for _, cl := range envoyUpstreamClusters(pg, prefix, svcName) {
		b.WriteString(cl)
	}
	return b.String()
}

func inboundPortForHandler(pg *ParsedGraph, svcName, iface string) int {
	for i, n := range pg.Services[svcName] {
		if n.Interface == iface {
			return sidecarIngressBasePort + i
		}
	}
	return sidecarIngressPort
}

func envoyRPCInboundClusterName(microservice, iface string) string {
	return k8sName(microservice) + "-" + iface + "-in"
}

func envoyInboundListeners(pg *ParsedGraph, svcName string, isEntry bool) string {
	var b strings.Builder
	for i, n := range pg.Services[svcName] {
		b.WriteString(envoyInboundHandlerListener(n, sidecarIngressBasePort+i, isEntry))
	}
	return b.String()
}

func envoyInboundHandlerListener(n *Node, port int, isEntry bool) string {
	h2 := ""
	if !isEntry {
		h2 = "\n          http2_protocol_options: {}"
	}
	iface := n.Interface
	statPrefix := "inbound_" + iface
	type routeSpec struct {
		prefix string
	}
	routes := []routeSpec{
		{"/" + n.FullRPCName()},
		{"/" + iface},
	}
	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].prefix) > len(routes[j].prefix)
	})
	var routeYAML strings.Builder
	for _, r := range routes {
		fmt.Fprintf(&routeYAML, `              - match: { prefix: "%s" }
                route: { cluster: local_app, timeout: %s }
`, r.prefix, envoyRouteTimeout)
	}
	fmt.Fprintf(&routeYAML, `              - match: { prefix: "/" }
                direct_response: { status: 404, body: { inline_string: "unknown inbound path" } }
`)
	return fmt.Sprintf(`  - name: listener_inbound_%s
    address:
      socket_address: { address: 0.0.0.0, port_value: %d }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: %s
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
%s%s

`, iface, port, statPrefix, routeYAML.String(), h2)
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
			cluster := envoyRPCInboundClusterName(tn.Microservice, tn.Interface)
			fmt.Fprintf(&routes, `              - match: { prefix: "/%s" }
                route: { cluster: %s, timeout: %s }
`, key, cluster, envoyRouteTimeout)
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

func envoyLocalAppCluster(isEntry bool) string {
	h2 := ""
	if !isEntry {
		h2 = "\n    http2_protocol_options: {}"
	}
	return fmt.Sprintf(`  - name: local_app
    connect_timeout: %s
    type: STATIC
    load_assignment:
      cluster_name: local_app
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address: { address: 127.0.0.1, port_value: %d }%s

`, envoyConnectTimeout, sidecarAppPort, h2)
}

func envoyUpstreamCluster(prefix, targetKn, clusterName string, port int, h2Streams bool) string {
	return envoyUpstreamClusterOpts(prefix, targetKn, clusterName, port, h2Streams, "ROUND_ROBIN", false)
}

func envoyUpstreamClusterOpts(prefix, targetKn, clusterName string, port int, h2Streams bool, lbPolicy string, highCB bool) string {
	extra := ""
	if h2Streams {
		extra += fmt.Sprintf(`
    http2_protocol_options:
      max_concurrent_streams: %d`, envoyMaxConcurrentStreams)
	}
	if highCB {
		extra += fmt.Sprintf(`
    circuit_breakers:
      thresholds:
      - priority: DEFAULT
        max_connections: %d
        max_pending_requests: %d
        max_requests: %d`, envoyHighCircuitBreakers, envoyHighCircuitBreakers, envoyHighCircuitBreakers)
	}
	host := prefix + targetKn
	return fmt.Sprintf(`  - name: %s
    connect_timeout: %s
    type: STRICT_DNS
    lb_policy: %s
    load_assignment:
      cluster_name: %s
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: %s
                port_value: %d%s

`, clusterName, envoyConnectTimeout, lbPolicy, clusterName, host, port, extra)
}

func envoyUpstreamClusters(pg *ParsedGraph, prefix, svcName string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range pg.Services[svcName] {
		for _, targetID := range pg.Downstream(n.ID) {
			tn := pg.Nodes[targetID]
			cn := envoyRPCInboundClusterName(tn.Microservice, tn.Interface)
			if seen[cn] {
				continue
			}
			seen[cn] = true
			port := inboundPortForHandler(pg, tn.Microservice, tn.Interface)
			out = append(out, envoyUpstreamCluster(prefix, k8sName(tn.Microservice), cn, port, true))
		}
	}
	return out
}

// upstreamPort 0 means per-handler inbound ports (envoy mesh). Otherwise all
// entry clusters use that fixed port (plain-lb app :2000).
func buildEnvoyIngressConfig(pg *ParsedGraph, prefix, entrySvc string, upstreamPort int, lbPolicy string, highCB bool) string {
	entryKn := k8sName(entrySvc)
	var b strings.Builder
	fmt.Fprintf(&b, "node:\n  id: ingress\n  cluster: local\n\n")
	b.WriteString(envoyIngressStatsConfig())
	b.WriteString(envoyAdminBlock())
	b.WriteString("static_resources:\n  listeners:\n")
	for i, n := range pg.EntryInterfaces() {
		p := sidecarIngressBasePort + i
		cn := "entry-" + n.Interface
		b.WriteString(envoyIngressListener(p, n.Interface, cn))
	}
	b.WriteString("  clusters:\n")
	for _, n := range pg.EntryInterfaces() {
		cn := "entry-" + n.Interface
		port := upstreamPort
		if port == 0 {
			port = inboundPortForHandler(pg, entrySvc, n.Interface)
		}
		b.WriteString(envoyUpstreamClusterOpts(prefix, entryKn, cn, port, false, lbPolicy, highCB))
	}
	return b.String()
}

func envoyIngressListener(port int, apiPath, clusterName string) string {
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
                route: { cluster: %s, timeout: %s }

`, apiPath, port, apiPath, apiPath, clusterName, envoyRouteTimeout)
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
		cpuStr := fmt.Sprintf("%d", int(cpu))
		concurrency := pg.SidecarCPUForService(name)
		envoyCpuStr := fmt.Sprintf("%d", concurrency)
		nHandlers := len(pg.Services[name])
		var envoyPortSpecs, svcIngressPorts []string
		for i := 0; i < nHandlers; i++ {
			p := sidecarIngressBasePort + i
			envoyPortSpecs = append(envoyPortSpecs, fmt.Sprintf("    - containerPort: %d", p))
			svcIngressPorts = append(svcIngressPorts, fmt.Sprintf(`  - name: envoy-ingress-%d
    port: %d
    targetPort: %d
    protocol: TCP`, p, p, p))
		}
		envoyPortSpecs = append(envoyPortSpecs,
			fmt.Sprintf("    - containerPort: %d", sidecarEgressPort),
			fmt.Sprintf("    - containerPort: %d", envoyAdminPort))
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
%s
%s  volumes:
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
%s
  - name: envoy-egress
    port: %d
    targetPort: %d
    protocol: TCP
`,
			kn, kn, imgName, cpuStr, cpuStr, cpuStr, benchmarkName, sidecarAppPort,
			envoyImage, concurrency, envoyCpuStr, envoyCpuStr, kn,
			strings.Join(envoyPortSpecs, "\n"),
			envoyStatsContainerYAML,
			prefix, kn, kn, kn,
			strings.Join(svcIngressPorts, "\n"),
			sidecarEgressPort, sidecarEgressPort)
		outPath := filepath.Join(manifestsDir, kn+"-envoy.yaml")
		if err := os.WriteFile(outPath, []byte(svcYaml), 0644); err != nil {
			return err
		}
	}
	return nil
}

func generateIngressEnvoyYaml(pg *ParsedGraph, benchmarkName string, outDir string) error {
	return writeIngressEnvoyYaml(pg, benchmarkName, outDir, "ingress-envoy.yaml", "envoy-configs", envoyIngressConcurrency, true)
}

func generateIngressEnvoyLbYaml(pg *ParsedGraph, benchmarkName string, outDir string) error {
	concurrency := pg.UserEntryCount() * 2
	return writeIngressEnvoyYaml(pg, benchmarkName, outDir, "ingress-envoy-lb.yaml", "plain-lb-envoy-configs", concurrency, false)
}

func writeIngressEnvoyYaml(pg *ParsedGraph, benchmarkName, outDir, filename, configMapName string, concurrency int, withStats bool) error {
	manifestsDir := filepath.Join(outDir, "k8s", "manifests")
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
	portSpecs = append(portSpecs, fmt.Sprintf("    - containerPort: %d", envoyAdminPort))
	statsYAML := ""
	if withStats {
		statsYAML = envoyStatsContainerYAML
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
%s  volumes:
  - name: config-volume
    configMap:
      name: %s
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
`, envoyImage, concurrency, concurrency, concurrency, strings.Join(portSpecs, "\n"),
		statsYAML, configMapName,
		benchmarkName, strings.Join(svcPorts, "\n"))
	return os.WriteFile(filepath.Join(manifestsDir, filename), []byte(yaml), 0644)
}
