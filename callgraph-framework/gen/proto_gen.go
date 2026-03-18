package gen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func GenerateProto(pg *ParsedGraph, module string, outDir string) error {
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	b.WriteString("syntax = \"proto3\";\n\n")
	b.WriteString("package benchmark;\n\n")
	b.WriteString("option go_package = \"" + module + "/protobuf\";\n\n")
	b.WriteString("message Request { string payload = 1; }\n")
	b.WriteString("message Response { string payload = 1; }\n\n")

	svcNames := make([]string, 0, len(pg.Services))
	for name := range pg.Services {
		svcNames = append(svcNames, name)
	}
	sort.Strings(svcNames)

	for _, svcName := range svcNames {
		nodes := pg.Services[svcName]
		b.WriteString("service " + svcName + " {\n")
		for _, n := range nodes {
			b.WriteString("  rpc " + n.ProtoMethodName() + "(Request) returns (Response);\n")
		}
		b.WriteString("}\n\n")
	}

	protoPath := filepath.Join(absOutDir, "proto", "benchmark.proto")
	if err := os.MkdirAll(filepath.Dir(protoPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(protoPath, b.Bytes(), 0644); err != nil {
		return err
	}

	protobufDir := filepath.Join(absOutDir, "protobuf")
	if err := os.MkdirAll(protobufDir, 0755); err != nil {
		return err
	}
	cmd := exec.Command("protoc",
		"--go_out="+absOutDir,
		"--go_opt=module="+module,
		"--go-grpc_out="+absOutDir,
		"--go-grpc_opt=module="+module,
		"-I", absOutDir,
		"proto/benchmark.proto")
	cmd.Dir = absOutDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("protoc failed: %w\n%s", err, out)
	}
	return nil
}
