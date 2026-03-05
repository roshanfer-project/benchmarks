package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func sanitizeModule(name string) string {
	s := strings.ReplaceAll(name, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	if s == "" {
		s = "benchmark"
	}
	return s
}

func Generate(callgraphPath string, outDir string, benchmarkName string) error {
	pg, err := ParseCallGraph(callgraphPath)
	if err != nil {
		return fmt.Errorf("parse callgraph: %w", err)
	}
	module := sanitizeModule(benchmarkName)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	if err := writeGoMod(module, outDir); err != nil {
		return err
	}
	if err := GenerateProto(pg, module, outDir); err != nil {
		return fmt.Errorf("generate proto: %w", err)
	}
	if err := GenerateServices(pg, module, outDir); err != nil {
		return fmt.Errorf("generate services: %w", err)
	}
	if err := GenerateK8s(pg, benchmarkName, "farzad1132", outDir); err != nil {
		return fmt.Errorf("generate k8s: %w", err)
	}
	if err := GenerateScripts(pg, benchmarkName, outDir); err != nil {
		return fmt.Errorf("generate scripts: %w", err)
	}
	if err := copyCallgraph(callgraphPath, outDir); err != nil {
		return err
	}
	return nil
}

func writeGoMod(module string, outDir string) error {
	content := fmt.Sprintf("module %s\n\ngo 1.25\n\nrequire (\n\tgoogle.golang.org/grpc v1.74.2\n\tgoogle.golang.org/protobuf v1.36.8\n)\n", module)
	return os.WriteFile(filepath.Join(outDir, "go.mod"), []byte(content), 0644)
}

func copyCallgraph(src string, outDir string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "callgraph.json"), data, 0644)
}
