package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func generateSidecarConfigs(genConfig *GeneratedConfig, outputDir string) error {
	if genConfig.Roshanfer == nil {
		return nil // No sidecar configs in plain mode
	}

	configsDir := filepath.Join(outputDir, "sidecar-configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return err
	}

	// Identify frontend services (for ingress sidecar)
	var frontends []string
	for name, svc := range genConfig.Services {
		if svc.Type == "frontend" {
			frontends = append(frontends, name)
		}
	}
	sort.Strings(frontends)

	// Generate ingress sidecar config
	if len(frontends) > 0 {
		if err := generateIngressSidecarConfig(genConfig, configsDir, frontends); err != nil {
			return err
		}
	}

	// Generate sidecar configs for each service
	var services []string
	for name := range genConfig.Services {
		services = append(services, name)
	}
	sort.Strings(services)

	for _, name := range services {
		svc := genConfig.Services[name]
		if svc.Type == "frontend" {
			if err := generateFrontendSidecarConfig(genConfig, configsDir, name); err != nil {
				return err
			}
		} else {
			if err := generateBackendSidecarConfig(genConfig, configsDir, name); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateIngressSidecarConfig(genConfig *GeneratedConfig, outputDir string, frontends []string) error {
	var sb strings.Builder

	// Routing block - maps frontend endpoints to their sidecars
	hasRouting := false
	routingContent := strings.Builder{}
	for _, frontendName := range frontends {
		ports := genConfig.Ports[frontendName]
		svc := genConfig.Services[frontendName]
		for _, endpoint := range svc.HTTPEndpoints {
			hasRouting = true
			cleanEndpoint := strings.TrimPrefix(endpoint, "/")
			routingContent.WriteString(fmt.Sprintf("  %s:\n", cleanEndpoint))
			routingContent.WriteString("    upstream:\n")
			routingContent.WriteString(fmt.Sprintf("      host: %s\n", frontendName))
			routingContent.WriteString(fmt.Sprintf("      port: %d\n", ports.IngressPort))
			// Add limit and SLO if specified
			// Check for endpoint-level limit first, then service-level limit
			if genConfig.Roshanfer.Limits != nil {
				if limitCfg, ok := genConfig.Roshanfer.Limits["ingress"]; ok {
					limitAdded := false
					// Check endpoint-level limit first
					if endpointLimit, ok := limitCfg.EndpointLimits[endpoint]; ok {
						routingContent.WriteString(fmt.Sprintf("    limit: %d\n", endpointLimit))
						limitAdded = true
					}
					// Check service-level limit (nested structure: service_limits)
					if !limitAdded && limitCfg.ServiceLimits != nil {
						if serviceLimit, ok := limitCfg.ServiceLimits[frontendName]; ok {
							routingContent.WriteString(fmt.Sprintf("    limit: %d\n", serviceLimit))
							limitAdded = true
						}
					}
				}
			}
			if genConfig.Roshanfer.SLOs != nil {
				if slo, ok := genConfig.Roshanfer.SLOs[endpoint]; ok {
					routingContent.WriteString(fmt.Sprintf("    slo: %d\n", slo))
				}
			}
		}
	}
	if hasRouting {
		sb.WriteString("routing:\n")
		sb.WriteString(routingContent.String())
	}

	// Mapping block - maps ingress endpoints to frontend services
	hasMapping := false
	mappingContent := strings.Builder{}
	for _, frontendName := range frontends {
		svc := genConfig.Services[frontendName]
		for _, endpoint := range svc.HTTPEndpoints {
			hasMapping = true
			cleanEndpoint := strings.TrimPrefix(endpoint, "/")
			mappingContent.WriteString(fmt.Sprintf("  %s:\n", cleanEndpoint))
			mappingContent.WriteString("    downstreams:\n")
			mappingContent.WriteString(fmt.Sprintf("      - %s\n", cleanEndpoint))
			mappingContent.WriteString("    min_max_concurrency: 6\n")
			ports := genConfig.Ports[frontendName]
			mappingContent.WriteString(fmt.Sprintf("    listen_port: %d\n", ports.HTTPPort))
			mappingContent.WriteString("    limit: -1\n")
			if genConfig.Roshanfer.SLOs != nil {
				if slo, ok := genConfig.Roshanfer.SLOs[endpoint]; ok {
					mappingContent.WriteString(fmt.Sprintf("    slo: %d\n", slo))
				}
			}
		}
	}
	if hasMapping {
		sb.WriteString("mapping:\n")
		sb.WriteString(mappingContent.String())
	}

	// Common config
	sb.WriteString("ring_size: 4000\n")
	sb.WriteString("buffer_count: 2000\n")
	sb.WriteString("buffer_size: 10000\n")
	// Set num_threads to the number of frontend services exposed by ingress
	numThreads := len(frontends)
	if numThreads == 0 {
		numThreads = 1 // Default to 1 if no frontends (shouldn't happen, but be safe)
	}
	sb.WriteString(fmt.Sprintf("num_threads: %d\n", numThreads))
	sb.WriteString("ingress_pool_connections: 100\n")
	sb.WriteString("frontend_pool_connections: 100\n")

	// Ingress-specific ports
	// For ingress, egress_listener_port is the port that receives external traffic
	// ingress_listener_port is not used
	ingressPort := genConfig.Ports[frontends[0]].HTTPPort // Use first frontend's HTTP port
	sb.WriteString(fmt.Sprintf("egress_listener_port: %d\n", ingressPort))
	sb.WriteString("ingress_listener_port: 4000\n")
	sb.WriteString("ingress_upstream_host: 127.0.0.1\n")
	sb.WriteString("ingress_upstream_port: 2000\n")
	sb.WriteString("is_ingress: True\n")
	sb.WriteString("is_frontend: False\n")
	if genConfig.Roshanfer.PPM != nil {
		if ppm, ok := genConfig.Roshanfer.PPM["ingress"]; ok {
			sb.WriteString(fmt.Sprintf("ppm_limit: %d\n", ppm))
		} else {
			sb.WriteString("ppm_limit: 2\n") // default
		}
	} else {
		sb.WriteString("ppm_limit: 2\n")
	}
	sb.WriteString("report_latency: True\n")
	sb.WriteString("name: ingress\n")

	file := filepath.Join(outputDir, "ingress.yaml")
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

func generateFrontendSidecarConfig(genConfig *GeneratedConfig, outputDir, serviceName string) error {
	var sb strings.Builder
	ports := genConfig.Ports[serviceName]
	svc := genConfig.Services[serviceName]

	// Routing block - maps RPC calls to backend sidecars
	hasRouting := false
	routingContent := strings.Builder{}
	for _, edge := range genConfig.Edges {
		if edge.From == serviceName {
			hasRouting = true
			backendPorts := genConfig.Ports[edge.To]
			serviceID := fmt.Sprintf("protobuf.%s/%s", edge.To, edge.Method)
			routingContent.WriteString(fmt.Sprintf("  %s:\n", serviceID))
			routingContent.WriteString("    upstream:\n")
			routingContent.WriteString(fmt.Sprintf("      host: %s\n", edge.To))
			routingContent.WriteString(fmt.Sprintf("      port: %d\n", backendPorts.IngressPort))
		}
	}
	if hasRouting {
		sb.WriteString("routing:\n")
		sb.WriteString(routingContent.String())
	}

	// Mapping block - maps HTTP endpoints to RPC calls
	hasMapping := false
	mappingContent := strings.Builder{}
	for _, endpoint := range svc.HTTPEndpoints {
		hasMapping = true
		cleanEndpoint := strings.TrimPrefix(endpoint, "/")
		mappingContent.WriteString(fmt.Sprintf("  %s:\n", cleanEndpoint))
		
		// Find all RPC calls for this endpoint
		hasDownstreams := false
		downstreamsContent := strings.Builder{}
		for _, edge := range genConfig.Edges {
			if edge.From == serviceName {
				hasDownstreams = true
				serviceID := fmt.Sprintf("protobuf.%s/%s", edge.To, edge.Method)
				downstreamsContent.WriteString(fmt.Sprintf("      - %s\n", serviceID))
			}
		}
		if hasDownstreams {
			mappingContent.WriteString("    downstreams:\n")
			mappingContent.WriteString(downstreamsContent.String())
		}
		
		mappingContent.WriteString("    min_max_concurrency: 20\n")
		// Get limit for this service
		if genConfig.Roshanfer.Limits != nil {
			if limitCfg, ok := genConfig.Roshanfer.Limits[serviceName]; ok {
				if limit, ok := limitCfg.ServiceLimits[serviceName]; ok {
					mappingContent.WriteString(fmt.Sprintf("    limit: %d\n", limit))
				} else {
					mappingContent.WriteString("    limit: 5\n") // default
				}
			} else {
				mappingContent.WriteString("    limit: 5\n")
			}
		} else {
			mappingContent.WriteString("    limit: 5\n")
		}
	}
	if hasMapping {
		sb.WriteString("mapping:\n")
		sb.WriteString(mappingContent.String())
	}

	// Common config
	sb.WriteString("ring_size: 1000\n")
	sb.WriteString("buffer_count: 1000\n")
	sb.WriteString("buffer_size: 80000\n")
	sb.WriteString("num_threads: 1\n")
	sb.WriteString("ingress_pool_connections: 100\n")
	sb.WriteString("frontend_pool_connections: 150\n")
	sb.WriteString(fmt.Sprintf("egress_listener_port: %d\n", ports.EgressPort))
	sb.WriteString(fmt.Sprintf("ingress_listener_port: %d\n", ports.IngressPort))
	sb.WriteString("ingress_upstream_host: 127.0.0.1\n")
	sb.WriteString(fmt.Sprintf("ingress_upstream_port: %d\n", ports.UpstreamPort))
	sb.WriteString("is_ingress: False\n")
	sb.WriteString("is_frontend: True\n")
	if genConfig.Roshanfer.PPM != nil {
		if ppm, ok := genConfig.Roshanfer.PPM[serviceName]; ok {
			sb.WriteString(fmt.Sprintf("ppm_limit: %d\n", ppm))
		} else {
			sb.WriteString("ppm_limit: 150\n") // default
		}
	} else {
		sb.WriteString("ppm_limit: 150\n")
	}
	sb.WriteString("report_latency: True\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", serviceName))

	file := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", serviceName))
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

func generateBackendSidecarConfig(genConfig *GeneratedConfig, outputDir, serviceName string) error {
	var sb strings.Builder
	ports := genConfig.Ports[serviceName]

	// Backend sidecars don't have routing (they're leaf nodes)
	// Mapping block - maps RPC methods to the backend service
	hasMapping := false
	mappingContent := strings.Builder{}
	methods := genConfig.ProtoServices[serviceName]
	for _, method := range methods {
		hasMapping = true
		serviceID := fmt.Sprintf("protobuf.%s/%s", serviceName, method)
		mappingContent.WriteString(fmt.Sprintf("  %s:\n", serviceID))
		mappingContent.WriteString("    min_max_concurrency: 20\n")
		// Get limit for this service/method
		if genConfig.Roshanfer.Limits != nil {
			if limitCfg, ok := genConfig.Roshanfer.Limits[serviceName]; ok {
				if limit, ok := limitCfg.ServiceLimits[serviceName]; ok {
					mappingContent.WriteString(fmt.Sprintf("    limit: %d\n", limit))
				} else {
					mappingContent.WriteString("    limit: 3\n") // default
				}
			} else {
				mappingContent.WriteString("    limit: 3\n")
			}
		} else {
			mappingContent.WriteString("    limit: 3\n")
		}
	}
	if hasMapping {
		sb.WriteString("mapping:\n")
		sb.WriteString(mappingContent.String())
	}

	// Common config
	sb.WriteString("ring_size: 1000\n")
	sb.WriteString("buffer_count: 1000\n")
	sb.WriteString("buffer_size: 80000\n")
	sb.WriteString("num_threads: 1\n")
	sb.WriteString("ingress_pool_connections: 100\n")
	sb.WriteString("frontend_pool_connections: 100\n")
	sb.WriteString(fmt.Sprintf("egress_listener_port: %d\n", ports.EgressPort))
	sb.WriteString(fmt.Sprintf("ingress_listener_port: %d\n", ports.IngressPort))
	sb.WriteString("ingress_upstream_host: 127.0.0.1\n")
	sb.WriteString(fmt.Sprintf("ingress_upstream_port: %d\n", ports.UpstreamPort))
	sb.WriteString("is_ingress: False\n")
	sb.WriteString("is_frontend: False\n")
	if genConfig.Roshanfer.PPM != nil {
		if ppm, ok := genConfig.Roshanfer.PPM[serviceName]; ok {
			sb.WriteString(fmt.Sprintf("ppm_limit: %d\n", ppm))
		} else {
			sb.WriteString("ppm_limit: 120\n") // default
		}
	} else {
		sb.WriteString("ppm_limit: 120\n")
	}
	sb.WriteString("report_latency: True\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", serviceName))

	file := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", serviceName))
	return os.WriteFile(file, []byte(sb.String()), 0644)
}


