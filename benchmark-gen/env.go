package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func generateEnvFiles(genConfig *GeneratedConfig, outputDir string) error {
	envDir := filepath.Join(outputDir, "env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return err
	}

	// Generate plain mode env file
	if err := generatePlainEnv(genConfig, envDir); err != nil {
		return err
	}

	// Generate roshanfer mode env file
	if genConfig.Roshanfer != nil {
		if err := generateRoshanferEnv(genConfig, envDir); err != nil {
			return err
		}
	}

	return nil
}

func generatePlainEnv(genConfig *GeneratedConfig, outputDir string) error {
	var sb strings.Builder

	var services []string
	for name := range genConfig.Services {
		services = append(services, name)
	}
	sort.Strings(services)

	for _, name := range services {
		svc := genConfig.Services[name]
		ports := genConfig.Ports[name]

		sb.WriteString(fmt.Sprintf("# %s\n", name))
		if svc.Type == "frontend" {
			sb.WriteString(fmt.Sprintf("%sListenPort=%d\n", name, ports.HTTPPort))
			sb.WriteString(fmt.Sprintf("%sSize=%d\n", name, svc.ResponseSize))
			sb.WriteString(fmt.Sprintf("%sPreRepeat=%d\n", name, svc.PreRepeat))
			sb.WriteString(fmt.Sprintf("%sPostRepeat=%d\n", name, svc.PostRepeat))

			// Build client connections
			for _, edge := range genConfig.Edges {
				if edge.From == name {
					backendPorts := genConfig.Ports[edge.To]
					sb.WriteString(fmt.Sprintf("%s%sEgress=localhost:%d\n", edge.To, name, backendPorts.GRPCPort))
				}
			}
		} else {
			sb.WriteString(fmt.Sprintf("%sListenPort=%d\n", name, ports.GRPCPort))
			sb.WriteString(fmt.Sprintf("%sRepeat=%d\n", name, svc.BackendRepeat))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("sidecar=false\n")

	file := filepath.Join(outputDir, "plain.env")
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

func generateRoshanferEnv(genConfig *GeneratedConfig, outputDir string) error {
	var sb strings.Builder

	var services []string
	for name := range genConfig.Services {
		services = append(services, name)
	}
	sort.Strings(services)

	for _, name := range services {
		svc := genConfig.Services[name]
		ports := genConfig.Ports[name]

		sb.WriteString(fmt.Sprintf("# %s\n", name))
		if svc.Type == "frontend" {
			sb.WriteString(fmt.Sprintf("%sListenPort=%d\n", name, ports.UpstreamPort))
			sb.WriteString(fmt.Sprintf("%sSize=%d\n", name, svc.ResponseSize))
			sb.WriteString(fmt.Sprintf("%sPreRepeat=%d\n", name, svc.PreRepeat))
			sb.WriteString(fmt.Sprintf("%sPostRepeat=%d\n", name, svc.PostRepeat))

			// In roshanfer mode, connect to sidecar egress port
			for _, edge := range genConfig.Edges {
				if edge.From == name {
					backendPorts := genConfig.Ports[edge.To]
					sb.WriteString(fmt.Sprintf("%s%sEgress=localhost:%d\n", edge.To, name, backendPorts.EgressPort))
				}
			}
		} else {
			sb.WriteString(fmt.Sprintf("%sListenPort=%d\n", name, ports.UpstreamPort))
			sb.WriteString(fmt.Sprintf("%sRepeat=%d\n", name, svc.BackendRepeat))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("sidecar=true\n")

	file := filepath.Join(outputDir, "sidecar.env")
	return os.WriteFile(file, []byte(sb.String()), 0644)
}

