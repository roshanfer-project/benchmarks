package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func generateKubernetesManifests(genConfig *GeneratedConfig, outputDir string) error {
	k8sDir := filepath.Join(outputDir, "k8s")
	if err := os.MkdirAll(k8sDir, 0755); err != nil {
		return err
	}

	// Generate namespace
	if err := generateNamespace(genConfig, k8sDir); err != nil {
		return err
	}

	// Generate services (K8s Service objects)
	if err := generateK8sServices(genConfig, k8sDir); err != nil {
		return err
	}

	// Generate NodePort services for both plain and roshanfer modes
	if err := generateNodePortServices(genConfig, k8sDir, false); err != nil {
		return err
	}
	if err := generateNodePortServices(genConfig, k8sDir, true); err != nil {
		return err
	}

	// Generate deployments for both plain and roshanfer modes
	if err := generateDeployments(genConfig, k8sDir, false); err != nil {
		return err
	}
	if err := generateDeployments(genConfig, k8sDir, true); err != nil {
		return err
	}

	// Generate configmaps for sidecar configs (always generate, even if not used in plain mode)
	if err := generateConfigMaps(genConfig, k8sDir); err != nil {
		return err
	}

	// Generate ingress deployment (only if roshanfer config exists)
	if genConfig.Roshanfer != nil {
		if err := generateIngressDeployment(genConfig, k8sDir); err != nil {
			return err
		}
	}

	return nil
}

func generateNamespace(genConfig *GeneratedConfig, outputDir string) error {
	var sb strings.Builder
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: Namespace\n")
	sb.WriteString("metadata:\n")
	sb.WriteString(fmt.Sprintf("  name: %s\n", genConfig.Namespace))

	file := filepath.Join(outputDir, "namespace.yaml")
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

func generateK8sServices(genConfig *GeneratedConfig, outputDir string) error {
	var services []string
	for name := range genConfig.Services {
		services = append(services, name)
	}
	sort.Strings(services)

	var sb strings.Builder
	for _, name := range services {
		svc := genConfig.Services[name]
		sb.WriteString("---\n")
		sb.WriteString("apiVersion: v1\n")
		sb.WriteString("kind: Service\n")
		sb.WriteString("metadata:\n")
		sb.WriteString(fmt.Sprintf("  name: %s\n", name))
		sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
		sb.WriteString("spec:\n")
		sb.WriteString("  type: ClusterIP\n")
		sb.WriteString("  selector:\n")
		sb.WriteString(fmt.Sprintf("    app: %s\n", name))
		sb.WriteString("  ports:\n")

		if svc.Type == "frontend" {
			ports := genConfig.Ports[name]
			sb.WriteString("  - name: http\n")
			sb.WriteString(fmt.Sprintf("    port: %d\n", ports.HTTPPort))
			sb.WriteString("    targetPort: http\n")
			// In roshanfer mode, also expose the sidecar ingress port for ingress sidecar to connect
			// Check if roshanfer mode by checking if Roshanfer config exists
			if genConfig.Roshanfer != nil {
				sb.WriteString("  - name: ingress\n")
				sb.WriteString(fmt.Sprintf("    port: %d\n", ports.IngressPort))
				sb.WriteString("    targetPort: ingress\n")
				// Add UDP port for sidecar ingress listener
				// Use numeric targetPort for UDP to ensure proper routing to sidecar container
				sb.WriteString("  - name: ingress-udp\n")
				sb.WriteString(fmt.Sprintf("    port: %d\n", ports.IngressPort))
				sb.WriteString(fmt.Sprintf("    targetPort: %d\n", ports.IngressPort))
				sb.WriteString("    protocol: UDP\n")
			}
		} else {
			ports := genConfig.Ports[name]
			sb.WriteString("  - name: grpc\n")
			sb.WriteString(fmt.Sprintf("    port: %d\n", ports.GRPCPort))
			sb.WriteString("    targetPort: grpc\n")
			// In roshanfer mode, add UDP port for sidecar ingress listener
			// Use numeric targetPort for UDP to ensure proper routing to sidecar container
			if genConfig.Roshanfer != nil {
				sb.WriteString("  - name: ingress-udp\n")
				sb.WriteString(fmt.Sprintf("    port: %d\n", ports.IngressPort))
				sb.WriteString(fmt.Sprintf("    targetPort: %d\n", ports.IngressPort))
				sb.WriteString("    protocol: UDP\n")
			}
		}
		sb.WriteString("\n")
	}

	file := filepath.Join(outputDir, "services.yaml")
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

func generateNodePortServices(genConfig *GeneratedConfig, outputDir string, roshanferMode bool) error {
	var sb strings.Builder

	if roshanferMode && genConfig.Roshanfer != nil {
		// In roshanfer mode, create NodePort service for ingress egress port
		// For ingress, the egress port (3000) is where clients connect (special case)
		var frontends []string
		for name, svc := range genConfig.Services {
			if svc.Type == "frontend" {
				frontends = append(frontends, name)
			}
		}
		if len(frontends) > 0 {
			sort.Strings(frontends)
			firstFrontendPorts := genConfig.Ports[frontends[0]]
			ingressEgressPort := firstFrontendPorts.HTTPPort // Ingress egress port is the frontend HTTP port

			sb.WriteString("---\n")
			sb.WriteString("apiVersion: v1\n")
			sb.WriteString("kind: Service\n")
			sb.WriteString("metadata:\n")
			sb.WriteString("  name: ingress-nodeport\n")
			sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
			sb.WriteString("spec:\n")
			sb.WriteString("  type: NodePort\n")
			sb.WriteString("  selector:\n")
			sb.WriteString("    app: ingress-sidecar\n")
			sb.WriteString("  ports:\n")
			sb.WriteString("  - name: egress\n")
			sb.WriteString(fmt.Sprintf("    port: %d\n", ingressEgressPort))
			sb.WriteString("    targetPort: egress\n")
			sb.WriteString("    nodePort: 3000\n")
			// Add UDP port for ingress sidecar's ingress listener (port 4000)
			sb.WriteString("  - name: ingress-udp\n")
			sb.WriteString("    port: 4000\n")
			sb.WriteString("    targetPort: ingress\n")
			sb.WriteString("    protocol: UDP\n")
			sb.WriteString("\n")
		}
	} else {
		// In plain mode, create NodePort service for frontend HTTP port
		var frontends []string
		for name, svc := range genConfig.Services {
			if svc.Type == "frontend" {
				frontends = append(frontends, name)
			}
		}
		sort.Strings(frontends)

		for _, name := range frontends {
			ports := genConfig.Ports[name]
			sb.WriteString("---\n")
			sb.WriteString("apiVersion: v1\n")
			sb.WriteString("kind: Service\n")
			sb.WriteString("metadata:\n")
			sb.WriteString(fmt.Sprintf("  name: %s-nodeport\n", name))
			sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
			sb.WriteString("spec:\n")
			sb.WriteString("  type: NodePort\n")
			sb.WriteString("  selector:\n")
			sb.WriteString(fmt.Sprintf("    app: %s\n", name))
			sb.WriteString("  ports:\n")
			sb.WriteString("  - name: http\n")
			sb.WriteString(fmt.Sprintf("    port: %d\n", ports.HTTPPort))
			sb.WriteString("    targetPort: http\n")
			sb.WriteString("    nodePort: 3000\n")
			sb.WriteString("\n")
			// Only create NodePort for first frontend (use port 3000)
			break
		}
	}

	if sb.Len() > 0 {
		var filename string
		if roshanferMode {
			filename = "nodeport-services-roshanfer.yaml"
		} else {
			filename = "nodeport-services-plain.yaml"
		}
		file := filepath.Join(outputDir, filename)
		return os.WriteFile(file, []byte(sb.String()), 0644)
	}

	return nil
}

func generateDeployments(genConfig *GeneratedConfig, outputDir string, roshanferMode bool) error {
	var services []string
	for name := range genConfig.Services {
		services = append(services, name)
	}
	sort.Strings(services)

	for _, name := range services {
		svc := genConfig.Services[name]
		ports := genConfig.Ports[name]
		cpuCores := genConfig.CPUCores[name]

		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("apiVersion: v1\n")
		sb.WriteString("kind: Pod\n")
		sb.WriteString("metadata:\n")
		sb.WriteString(fmt.Sprintf("  name: %s\n", name))
		sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
		sb.WriteString("  labels:\n")
		sb.WriteString(fmt.Sprintf("    app: %s\n", name))
		sb.WriteString("spec:\n")
		sb.WriteString("  restartPolicy: Never\n")

		// Add initContainers to wait for dependencies and service port
		hasInitContainers := false
		initContainerContent := strings.Builder{}

		// Wait for dependency services (services this service depends on via edges)
		if svc.Type == "frontend" {
			for _, edge := range genConfig.Edges {
				if edge.From == name {
					// This frontend depends on the backend service
					depPorts := genConfig.Ports[edge.To]
					hasInitContainers = true
					initContainerContent.WriteString(fmt.Sprintf("  - name: wait-for-%s\n", edge.To))
					initContainerContent.WriteString("    image: busybox:latest\n")
					initContainerContent.WriteString("    imagePullPolicy: Never\n")
					initContainerContent.WriteString("    command:\n")
					initContainerContent.WriteString("    - sh\n")
					initContainerContent.WriteString("    - -c\n")
					if roshanferMode && genConfig.Roshanfer != nil {
						// In roshanfer mode, wait for the backend's ingress port
						initContainerContent.WriteString(fmt.Sprintf("    - until nc -z %s %d; do sleep 0.1; done\n", edge.To, depPorts.IngressPort))
					} else {
						// In plain mode, wait for the backend's gRPC port
						initContainerContent.WriteString(fmt.Sprintf("    - until nc -z %s %d; do sleep 0.1; done\n", edge.To, depPorts.GRPCPort))
					}
				}
			}
		}

		if hasInitContainers {
			sb.WriteString("  initContainers:\n")
			sb.WriteString(initContainerContent.String())
		}

		sb.WriteString("  containers:\n")

		// App container
		sb.WriteString("      - name: app\n")
		if svc.Type == "frontend" {
			sb.WriteString("        image: benchmark-frontend:latest\n")
		} else {
			sb.WriteString("        image: benchmark-backend:latest\n")
		}
		sb.WriteString("        imagePullPolicy: Never\n")
		sb.WriteString("        env:\n")
		sb.WriteString("        - name: SERVICE_NAME\n")
		sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", name))
		sb.WriteString("        - name: LISTEN_PORT\n")
		if roshanferMode && genConfig.Roshanfer != nil {
			// In roshanfer mode, services listen on upstream port (sidecar handles ingress/egress)
			sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", ports.UpstreamPort))
		} else {
			// Plain mode - direct connections
			if svc.Type == "frontend" {
				sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", ports.HTTPPort))
			} else {
				sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", ports.GRPCPort))
			}
		}
		sb.WriteString("        - name: RESPONSE_SIZE\n")
		sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", svc.ResponseSize))
		sb.WriteString("        - name: PRE_REPEAT\n")
		sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", svc.PreRepeat))
		sb.WriteString("        - name: POST_REPEAT\n")
		sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", svc.PostRepeat))

		if svc.Type == "backend" {
			sb.WriteString("        - name: BACKEND_REPEAT\n")
			sb.WriteString(fmt.Sprintf("          value: \"%d\"\n", svc.BackendRepeat))
			sb.WriteString("        - name: SERVICE_NAMES\n")
			sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", name))
		}

		// Add sidecar mode and endpoint configs if roshanfer mode
		if roshanferMode && genConfig.Roshanfer != nil {
			sb.WriteString("        - name: sidecar\n")
			sb.WriteString("          value: \"true\"\n")

			// Build endpoint configuration
			if svc.Type == "frontend" {
				endpointConfigs := buildEndpointConfigs(name, genConfig)
				for endpoint, config := range endpointConfigs {
					envName := strings.ReplaceAll(strings.ToUpper(endpoint), "/", "_")
					envName = strings.ReplaceAll(envName, "__", "_")
					if strings.HasPrefix(envName, "_") {
						envName = envName[1:]
					}
					sb.WriteString(fmt.Sprintf("        - name: ENDPOINT_%s\n", envName))
					sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", config))
				}
				// Add default endpoint config if none generated (e.g. no edges)
				if len(endpointConfigs) == 0 {
					for _, endpoint := range svc.HTTPEndpoints {
						envName := strings.ReplaceAll(strings.ToUpper(endpoint), "/", "_")
						envName = strings.ReplaceAll(envName, "__", "_")
						if strings.HasPrefix(envName, "_") {
							envName = envName[1:]
						}
						// Format: pre:N:post:N (no calls)
						config := fmt.Sprintf("pre:%d:post:%d", svc.PreRepeat, svc.PostRepeat)
						sb.WriteString(fmt.Sprintf("        - name: ENDPOINT_%s\n", envName))
						sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", config))
					}
				}

				// Build client endpoints
				var clientEndpoints []string
				for _, edge := range genConfig.Edges {
					if edge.From == name {
						ports := genConfig.Ports[edge.To]
						clientEndpoints = append(clientEndpoints, fmt.Sprintf("%s=%s:%d", edge.To, edge.To, ports.EgressPort))
					}
				}
				if len(clientEndpoints) > 0 {
					sb.WriteString("        - name: ENDPOINTS\n")
					sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", strings.Join(clientEndpoints, ",")))
				}
			}
		} else {
			sb.WriteString("        - name: sidecar\n")
			sb.WriteString("          value: \"false\"\n")

			// Plain mode - direct connections
			if svc.Type == "frontend" {
				var clientEndpoints []string
				for _, edge := range genConfig.Edges {
					if edge.From == name {
						ports := genConfig.Ports[edge.To]
						clientEndpoints = append(clientEndpoints, fmt.Sprintf("%s=%s:%d", edge.To, edge.To, ports.GRPCPort))
					}
				}
				if len(clientEndpoints) > 0 {
					sb.WriteString("        - name: ENDPOINTS\n")
					sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", strings.Join(clientEndpoints, ",")))
				}

				// Add Endpoint Configs for Plain Mode too!
				endpointConfigs := buildEndpointConfigs(name, genConfig)
				for endpoint, config := range endpointConfigs {
					envName := strings.ReplaceAll(strings.ToUpper(endpoint), "/", "_")
					// Fix: replace double underscore with single if it exists (e.g. from leading slash)
					envName = strings.ReplaceAll(envName, "__", "_")
					// If name starts with underscore (from leading slash), remove it
					if strings.HasPrefix(envName, "_") {
						envName = envName[1:]
					}

					sb.WriteString(fmt.Sprintf("        - name: ENDPOINT_%s\n", envName))
					sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", config))
				}
				// Add default endpoint config if none generated (e.g. no edges)
				if len(endpointConfigs) == 0 {
					for _, endpoint := range svc.HTTPEndpoints {
						envName := strings.ReplaceAll(strings.ToUpper(endpoint), "/", "_")
						envName = strings.ReplaceAll(envName, "__", "_")
						if strings.HasPrefix(envName, "_") {
							envName = envName[1:]
						}
						// Format: pre:N:post:N (no calls)
						config := fmt.Sprintf("pre:%d:post:%d", svc.PreRepeat, svc.PostRepeat)
						sb.WriteString(fmt.Sprintf("        - name: ENDPOINT_%s\n", envName))
						sb.WriteString(fmt.Sprintf("          value: \"%s\"\n", config))
					}
				}
			}
		}

		// CPU pinning
		// We explicitly set CPU requests and limits to ensure pinning.
		// The numbering assumes that the node has enough cores and static CPU management policy.
		if len(cpuCores.ServiceCores) > 0 {
			sb.WriteString("        resources:\n")
			sb.WriteString("          requests:\n")
			sb.WriteString(fmt.Sprintf("            cpu: \"%d\"\n", len(cpuCores.ServiceCores)))
			sb.WriteString("          limits:\n")
			sb.WriteString(fmt.Sprintf("            cpu: \"%d\"\n", len(cpuCores.ServiceCores)))
		}

		// Ports
		sb.WriteString("        ports:\n")
		if svc.Type == "frontend" {
			sb.WriteString("        - name: http\n")
			sb.WriteString(fmt.Sprintf("          containerPort: %d\n", ports.HTTPPort))
		} else {
			sb.WriteString("        - name: grpc\n")
			sb.WriteString(fmt.Sprintf("          containerPort: %d\n", ports.GRPCPort))
		}

		// Sidecar container (if roshanfer mode)
		if roshanferMode && genConfig.Roshanfer != nil {
			sb.WriteString("      - name: sidecar\n")
			sb.WriteString("        image: sidecar-sidecar:latest\n")
			sb.WriteString("        imagePullPolicy: Never\n")
			sb.WriteString("        securityContext:\n")
			sb.WriteString("          privileged: true\n")
			sb.WriteString("        env:\n")
			sb.WriteString("        - name: PROC_NAME\n")
			sb.WriteString(fmt.Sprintf("          value: \"%s-sidecar\"\n", name))
			sb.WriteString("        - name: GLOG_logtostderr\n")
			sb.WriteString("          value: \"1\"\n")
			sb.WriteString("        volumeMounts:\n")
			sb.WriteString("        - name: config\n")
			sb.WriteString("          mountPath: /config.yaml\n")
			sb.WriteString("          subPath: config.yaml\n")
			sb.WriteString("        - name: usr-local-lib\n")
			sb.WriteString("          mountPath: /usr/local/lib\n")
			sb.WriteString("          readOnly: true\n")
			sb.WriteString("        - name: lib-x86-64\n")
			sb.WriteString("          mountPath: /lib/x86_64-linux-gnu\n")
			sb.WriteString("          readOnly: true\n")
			sb.WriteString("        command:\n")
			sb.WriteString("        - /sidecar\n")
			sb.WriteString("        - /config.yaml\n")
			sb.WriteString("        startupProbe:\n")
			sb.WriteString("          tcpSocket:\n")
			sb.WriteString(fmt.Sprintf("            port: %d\n", ports.UpstreamPort))
			sb.WriteString("          initialDelaySeconds: 0\n")
			sb.WriteString("          periodSeconds: 1\n")
			sb.WriteString("          timeoutSeconds: 1\n")
			sb.WriteString("          successThreshold: 1\n")
			sb.WriteString("          failureThreshold: 300\n")

			// CPU pinning for sidecar
			if len(cpuCores.SidecarCores) > 0 {
				sb.WriteString("        resources:\n")
				sb.WriteString("          requests:\n")
				sb.WriteString(fmt.Sprintf("            cpu: \"%d\"\n", len(cpuCores.SidecarCores)))
				sb.WriteString("          limits:\n")
				sb.WriteString(fmt.Sprintf("            cpu: \"%d\"\n", len(cpuCores.SidecarCores)))
			}

			// Ports for sidecar
			sb.WriteString("        ports:\n")
			sb.WriteString("        - name: ingress\n")
			sb.WriteString(fmt.Sprintf("          containerPort: %d\n", ports.IngressPort))
			sb.WriteString("        - name: egress\n")
			sb.WriteString(fmt.Sprintf("          containerPort: %d\n", ports.EgressPort))
		}

		// Volumes for sidecar config
		if roshanferMode && genConfig.Roshanfer != nil {
			sb.WriteString("  volumes:\n")
			sb.WriteString("  - name: config\n")
			sb.WriteString("    configMap:\n")
			sb.WriteString(fmt.Sprintf("      name: %s-sidecar-config\n", name))
			sb.WriteString("  - name: usr-local-lib\n")
			sb.WriteString("    hostPath:\n")
			sb.WriteString("      path: /usr/local/lib\n")
			sb.WriteString("      type: Directory\n")
			sb.WriteString("  - name: lib-x86-64\n")
			sb.WriteString("    hostPath:\n")
			sb.WriteString("      path: /lib/x86_64-linux-gnu\n")
			sb.WriteString("      type: Directory\n")
		}

		var filename string
		if roshanferMode {
			filename = fmt.Sprintf("%s-pod-roshanfer.yaml", name)
		} else {
			filename = fmt.Sprintf("%s-pod-plain.yaml", name)
		}
		file := filepath.Join(outputDir, filename)
		if err := os.WriteFile(file, []byte(sb.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func generateIngressDeployment(genConfig *GeneratedConfig, outputDir string) error {
	// Find frontend services
	var frontends []string
	for name, svc := range genConfig.Services {
		if svc.Type == "frontend" {
			frontends = append(frontends, name)
		}
	}
	if len(frontends) == 0 {
		return nil // No ingress needed
	}
	sort.Strings(frontends)

	// Verify ingress config exists
	ingressConfigFile := filepath.Join(outputDir, "..", "sidecar-configs", "ingress.yaml")
	if _, err := os.Stat(ingressConfigFile); err != nil {
		return fmt.Errorf("ingress config file not found: %w", err)
	}

	// Determine ingress ports from first frontend
	firstFrontendPorts := genConfig.Ports[frontends[0]]
	ingressPort := firstFrontendPorts.HTTPPort

	// Get CPU cores for ingress (use first frontend's sidecar cores as default)
	var ingressCPUCores []int
	if len(frontends) > 0 {
		firstFrontendCores := genConfig.CPUCores[frontends[0]]
		ingressCPUCores = firstFrontendCores.SidecarCores
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: Pod\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: ingress-sidecar\n")
	sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
	sb.WriteString("  labels:\n")
	sb.WriteString("    app: ingress-sidecar\n")
	sb.WriteString("spec:\n")
	sb.WriteString("  restartPolicy: Never\n")
	sb.WriteString("  containers:\n")
	sb.WriteString("      - name: sidecar\n")
	sb.WriteString("        image: sidecar-sidecar:latest\n")
	sb.WriteString("        imagePullPolicy: Never\n")
	sb.WriteString("        securityContext:\n")
	sb.WriteString("          privileged: true\n")
	sb.WriteString("        env:\n")
	sb.WriteString("        - name: PROC_NAME\n")
	sb.WriteString("          value: \"ingress-sidecar\"\n")
	sb.WriteString("        - name: GLOG_logtostderr\n")
	sb.WriteString("          value: \"1\"\n")
	sb.WriteString("        volumeMounts:\n")
	sb.WriteString("        - name: config\n")
	sb.WriteString("          mountPath: /config.yaml\n")
	sb.WriteString("          subPath: config.yaml\n")
	sb.WriteString("        - name: usr-local-lib\n")
	sb.WriteString("          mountPath: /usr/local/lib\n")
	sb.WriteString("          readOnly: true\n")
	sb.WriteString("        - name: lib-x86-64\n")
	sb.WriteString("          mountPath: /lib/x86_64-linux-gnu\n")
	sb.WriteString("          readOnly: true\n")
	sb.WriteString("        command:\n")
	sb.WriteString("        - /sidecar\n")
	sb.WriteString("        - /config.yaml\n")

	// CPU pinning for ingress sidecar
	if len(ingressCPUCores) > 0 {
		sb.WriteString("        resources:\n")
		sb.WriteString("          requests:\n")
		sb.WriteString(fmt.Sprintf("            cpu: \"%d\"\n", len(ingressCPUCores)))
		sb.WriteString("          limits:\n")
		sb.WriteString(fmt.Sprintf("            cpu: \"%d\"\n", len(ingressCPUCores)))
	}

	// Ports for ingress sidecar
	sb.WriteString("        ports:\n")
	sb.WriteString("        - name: egress\n")
	sb.WriteString(fmt.Sprintf("          containerPort: %d\n", ingressPort))
	sb.WriteString("        - name: ingress\n")
	sb.WriteString("          containerPort: 4000\n")

	// Volumes for ingress config
	sb.WriteString("  volumes:\n")
	sb.WriteString("  - name: config\n")
	sb.WriteString("    configMap:\n")
	sb.WriteString("      name: ingress-sidecar-config\n")
	sb.WriteString("  - name: usr-local-lib\n")
	sb.WriteString("    hostPath:\n")
	sb.WriteString("      path: /usr/local/lib\n")
	sb.WriteString("      type: Directory\n")
	sb.WriteString("  - name: lib-x86-64\n")
	sb.WriteString("    hostPath:\n")
	sb.WriteString("      path: /lib/x86_64-linux-gnu\n")
	sb.WriteString("      type: Directory\n")

	file := filepath.Join(outputDir, "ingress-pod.yaml")
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

func generateConfigMaps(genConfig *GeneratedConfig, outputDir string) error {
	// Always generate configmaps if roshanfer config exists (even if not used in plain mode)
	if genConfig.Roshanfer == nil {
		return nil // No sidecar configs if roshanfer config not provided
	}

	var services []string
	for name := range genConfig.Services {
		services = append(services, name)
	}
	sort.Strings(services)

	// Generate ingress configmap
	ingressConfigFile := filepath.Join(outputDir, "..", "sidecar-configs", "ingress.yaml")
	if _, err := os.Stat(ingressConfigFile); err == nil {
		configData, err := os.ReadFile(ingressConfigFile)
		if err == nil {
			var sb strings.Builder
			sb.WriteString("---\n")
			sb.WriteString("apiVersion: v1\n")
			sb.WriteString("kind: ConfigMap\n")
			sb.WriteString("metadata:\n")
			sb.WriteString("  name: ingress-sidecar-config\n")
			sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
			sb.WriteString("data:\n")
			sb.WriteString("  config.yaml: |\n")
			lines := strings.Split(string(configData), "\n")
			for _, line := range lines {
				sb.WriteString("    ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
			file := filepath.Join(outputDir, "ingress-configmap.yaml")
			if err := os.WriteFile(file, []byte(sb.String()), 0644); err != nil {
				return err
			}
		}
	}

	for _, name := range services {
		// Read the sidecar config file we generated
		configFile := filepath.Join(outputDir, "..", "sidecar-configs", fmt.Sprintf("%s.yaml", name))
		configData, err := os.ReadFile(configFile)
		if err != nil {
			continue // Skip if file doesn't exist
		}

		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("apiVersion: v1\n")
		sb.WriteString("kind: ConfigMap\n")
		sb.WriteString("metadata:\n")
		sb.WriteString(fmt.Sprintf("  name: %s-sidecar-config\n", name))
		sb.WriteString(fmt.Sprintf("  namespace: %s\n", genConfig.Namespace))
		sb.WriteString("data:\n")
		sb.WriteString("  config.yaml: |\n")
		// Indent the config content
		lines := strings.Split(string(configData), "\n")
		for _, line := range lines {
			sb.WriteString("    ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}

		file := filepath.Join(outputDir, fmt.Sprintf("%s-configmap.yaml", name))
		if err := os.WriteFile(file, []byte(sb.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func buildEndpointConfigs(frontendName string, genConfig *GeneratedConfig) map[string]string {
	configs := make(map[string]string)
	svc := genConfig.Services[frontendName]

	// Group edges by endpoint
	endpointEdges := make(map[string][]Edge)
	for _, edge := range genConfig.Edges {
		if edge.From == frontendName {
			// For now, map to first HTTP endpoint
			// TODO: allow explicit mapping
			endpoint := svc.HTTPEndpoints[0]
			endpointEdges[endpoint] = append(endpointEdges[endpoint], edge)
		}
	}

	for endpoint, edges := range endpointEdges {
		var parts []string
		// Use default pre-repeat or per-RPC override
		defaultPreRepeat := svc.PreRepeat
		parts = append(parts, fmt.Sprintf("pre:%d", defaultPreRepeat))

		for _, edge := range edges {
			parts = append(parts, edge.Method)
			parts = append(parts, edge.To)
			// Add per-RPC pre/post if specified
			if edge.PreRepeat != nil {
				parts = append(parts, fmt.Sprintf("pre:%d", *edge.PreRepeat))
			}
			if edge.PostRepeat != nil {
				parts = append(parts, fmt.Sprintf("post:%d", *edge.PostRepeat))
			}
		}
		// Use default post-repeat
		defaultPostRepeat := svc.PostRepeat
		parts = append(parts, fmt.Sprintf("post:%d", defaultPostRepeat))
		configs[endpoint] = strings.Join(parts, ":")
	}

	return configs
}
