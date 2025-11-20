package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func generate(inputFile, outputDir string) error {
	// Read input YAML
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var config BenchmarkConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults
	if config.Namespace == "" {
		config.Namespace = "app"
	}
	if config.CPU == nil {
		config.CPU = &CPUConfig{StartCore: 0, TotalCores: 64}
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Generate computed values
	genConfig, err := computeGeneratedConfig(&config)
	if err != nil {
		return fmt.Errorf("failed to compute config: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate go.mod and go.sum for services
	if err := generateGoMod(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	// Generate test utilities
	if err := generateTestUtilities(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate test utilities: %w", err)
	}

	// Generate protobuf
	if err := generateProtobuf(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate protobuf: %w", err)
	}

	// Generate protobuf Go code (need protoc)
	if err := generateProtobufGo(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate protobuf Go code: %w", err)
	}

	// Generate generic services
	if err := generateGenericServices(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate services: %w", err)
	}

	// Generate sidecar configs
	if err := generateSidecarConfigs(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate sidecar configs: %w", err)
	}

	// Generate Kubernetes manifests
	if err := generateKubernetesManifests(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate k8s manifests: %w", err)
	}

	// Generate environment files
	if err := generateEnvFiles(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate env files: %w", err)
	}

	// Generate go.mod and go.sum for services
	if err := generateGoMod(genConfig, outputDir); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	fmt.Printf("Successfully generated benchmark artifacts in %s\n", outputDir)
	return nil
}

func generateGoMod(genConfig *GeneratedConfig, outputDir string) error {
	// Copy go.mod from test directory structure
	goModContent := `module test

go 1.24.2

require (
	github.com/alexflint/go-arg v1.6.0
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.8
)
`

	goModPath := filepath.Join(outputDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return err
	}

	// Generate empty go.sum (will be populated by go mod download)
	goSumPath := filepath.Join(outputDir, "go.sum")
	return os.WriteFile(goSumPath, []byte(""), 0644)
}

func validateConfig(config *BenchmarkConfig) error {
	// Check that all services referenced in edges exist
	serviceMap := make(map[string]bool)
	for name := range config.Services {
		serviceMap[name] = true
	}

	for _, edge := range config.Edges {
		if !serviceMap[edge.From] {
			return fmt.Errorf("edge references unknown service: %s", edge.From)
		}
		if !serviceMap[edge.To] {
			return fmt.Errorf("edge references unknown service: %s", edge.To)
		}
		// Check that 'from' is a frontend and 'to' is a backend
		if config.Services[edge.From].Type != "frontend" {
			return fmt.Errorf("edge 'from' must be a frontend service: %s", edge.From)
		}
		if config.Services[edge.To].Type != "backend" {
			return fmt.Errorf("edge 'to' must be a backend service: %s", edge.To)
		}
	}

	// Check that frontend services have HTTP endpoints
	for name, svc := range config.Services {
		if svc.Type == "frontend" && len(svc.HTTPEndpoints) == 0 {
			return fmt.Errorf("frontend service %s must have at least one HTTP endpoint", name)
		}
	}

	return nil
}

func computeGeneratedConfig(config *BenchmarkConfig) (*GeneratedConfig, error) {
	genConfig := &GeneratedConfig{
		BenchmarkConfig: *config,
		Ports:           make(map[string]ServicePorts),
		CPUCores:        make(map[string]CPUCoreAssignment),
		ProtoServices:   make(map[string][]string),
	}

	// Assign ports
	basePort := 3000
	for name := range config.Services {
		ports := ServicePorts{
			HTTPPort:     basePort,
			GRPCPort:     basePort + 1,
			IngressPort:  basePort + 2,
			EgressPort:   basePort + 3,
			UpstreamPort: basePort + 4,
		}
		genConfig.Ports[name] = ports
		basePort += 10
	}

	// Assign CPU cores
	// To ensure services are pinned to the same cores in both plain and roshanfer modes,
	// we reserve cores for sidecars even in plain mode.
	currentCore := config.CPU.StartCore
	for name, svc := range config.Services {
		sidecarCores := svc.SidecarCPUCores
		if sidecarCores == 0 {
			sidecarCores = 2 // default
		}
		totalCores := svc.CPUCores + sidecarCores

		if currentCore+totalCores > config.CPU.StartCore+config.CPU.TotalCores {
			return nil, fmt.Errorf("not enough CPU cores available")
		}

		serviceCores := make([]int, svc.CPUCores)
		sidecarCoresList := make([]int, sidecarCores)
		for i := 0; i < svc.CPUCores; i++ {
			serviceCores[i] = currentCore + i
		}
		for i := 0; i < sidecarCores; i++ {
			sidecarCoresList[i] = currentCore + svc.CPUCores + i
		}

		genConfig.CPUCores[name] = CPUCoreAssignment{
			ServiceCores: serviceCores,
			SidecarCores: sidecarCoresList,
		}

		currentCore += totalCores
	}

	// Build proto services map (backend -> methods)
	for _, edge := range config.Edges {
		genConfig.ProtoServices[edge.To] = append(genConfig.ProtoServices[edge.To], edge.Method)
	}

	return genConfig, nil
}

func generateProtobuf(genConfig *GeneratedConfig, outputDir string) error {
	protoDir := filepath.Join(outputDir, "protobuf")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		return err
	}

	var services []string
	for backendName := range genConfig.ProtoServices {
		services = append(services, backendName)
	}
	sort.Strings(services)

	var sb strings.Builder
	sb.WriteString("syntax = \"proto3\";\n\n")
	sb.WriteString("package protobuf;\n\n")
	sb.WriteString("option go_package = \"test/protobuf\";\n\n")
	sb.WriteString("message Arg {\n")
	sb.WriteString("  string data = 1;\n")
	sb.WriteString("}\n\n")
	sb.WriteString("message Resp {\n")
	sb.WriteString("  string data = 1;\n")
	sb.WriteString("}\n\n")

	for _, backendName := range services {
		methods := genConfig.ProtoServices[backendName]
		sb.WriteString(fmt.Sprintf("service %s {\n", backendName))
		for _, method := range methods {
			sb.WriteString(fmt.Sprintf("  rpc %s(Arg) returns (Resp);\n", method))
		}
		sb.WriteString("}\n\n")
	}

	protoFile := filepath.Join(protoDir, "services.proto")
	return os.WriteFile(protoFile, []byte(sb.String()), 0644)
}
