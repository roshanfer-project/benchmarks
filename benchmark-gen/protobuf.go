package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func generateProtobufGo(genConfig *GeneratedConfig, outputDir string) error {
	protoDir := filepath.Join(outputDir, "protobuf")
	protoFile := filepath.Join(protoDir, "services.proto")

	// Check if protoc is available
	if _, err := exec.LookPath("protoc"); err != nil {
		return fmt.Errorf("protoc not found in PATH. Please install Protocol Buffers compiler")
	}

	// Generate Go code from proto
	cmd := exec.Command("protoc",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
		filepath.Base(protoFile),
	)
	cmd.Dir = protoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate protobuf Go code: %w", err)
	}

	return nil
}

func generateTestUtilities(genConfig *GeneratedConfig, outputDir string) error {
	testDir := filepath.Join(outputDir, "test")
	utilsDir := filepath.Join(testDir, "utils")

	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		return err
	}

	// Copy utils files (simplified versions)
	// In a real implementation, you'd copy from the test directory
	// For now, generate minimal versions

	// Generate ennvars.go
	ennvarsContent := `package utils

import (
	"os"
	"strconv"
)

func GetEnvVar(key string, required bool) string {
	value := os.Getenv(key)
	if value == "" && required {
		panic("Environment variable " + key + " is required")
	}
	return value
}

func StrToInt(s string) int {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("Failed to convert string to int: " + s)
	}
	return int(i)
}
`
	if err := os.WriteFile(filepath.Join(utilsDir, "ennvars.go"), []byte(ennvarsContent), 0644); err != nil {
		return err
	}

	// Generate log.go
	logContent := `package utils

import (
	"log/slog"
	"os"
	"strings"
)

func GetLogger(name string) *slog.Logger {
	logLevel := os.Getenv("LOG_LEVEL")
	level := slog.LevelInfo
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	return logger.With("package", name)
}
`
	if err := os.WriteFile(filepath.Join(utilsDir, "log.go"), []byte(logContent), 0644); err != nil {
		return err
	}

	// Generate grpc-client.go
	grpcClientDir := filepath.Join(testDir, "test")
	if err := os.MkdirAll(grpcClientDir, 0755); err != nil {
		return err
	}
	grpcClientContent := `package test

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetConnBasic(serverAddr string) *grpc.ClientConn {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.NewClient(serverAddr, options...)
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}
`
	if err := os.WriteFile(filepath.Join(grpcClientDir, "grpc-client.go"), []byte(grpcClientContent), 0644); err != nil {
		return err
	}

	// Generate grpc-server.go
	grpcServerContent := `package test

import (
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var Opts = []grpc.ServerOption{
	grpc.KeepaliveParams(keepalive.ServerParameters{
		Timeout: 120 * time.Second,
	}),
	grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		PermitWithoutStream: true,
	}),
}
`
	if err := os.WriteFile(filepath.Join(grpcClientDir, "grpc-server.go"), []byte(grpcServerContent), 0644); err != nil {
		return err
	}

	return nil
}
