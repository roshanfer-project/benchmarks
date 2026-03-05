package gen

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

const port = 2000

func GenerateServices(pg *ParsedGraph, module string, outDir string) error {
	if err := writeUtils(outDir); err != nil {
		return err
	}
	if err := writeGRPCClient(module, outDir); err != nil {
		return err
	}
	svcNames := sortedServices(pg)
	for _, svcName := range svcNames {
		if svcName == pg.EntryMicroservice() {
			if err := generateEntryService(pg, module, svcName, outDir); err != nil {
				return err
			}
		} else {
			if err := generateGRPCService(pg, module, svcName, outDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedServices(pg *ParsedGraph) []string {
	names := make([]string, 0, len(pg.Services))
	for n := range pg.Services {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// deploymentOrder returns services in lead-to-root order: leaves first, entry last.
func deploymentOrder(pg *ParsedGraph) []string {
	visited := make(map[string]bool)
	var order []string
	var visit func(svc string)
	visit = func(svc string) {
		if visited[svc] {
			return
		}
		visited[svc] = true
		for _, node := range pg.Services[svc] {
			for _, targetID := range pg.Downstream(node.ID) {
				if targetNode, ok := pg.Nodes[targetID]; ok {
					visit(targetNode.Microservice)
				}
			}
		}
		order = append(order, svc)
	}
	visit(pg.EntryMicroservice())
	// Append any services not reachable from entry
	for svc := range pg.Services {
		if !visited[svc] {
			order = append(order, svc)
		}
	}
	return order
}

func writeUtils(outDir string) error {
	utilsDir := filepath.Join(outDir, "utils")
	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		return err
	}
	files := map[string]string{
		"busy.go":    busyGo,
		"ennvars.go": ennvarsGo,
		"log.go":     logGo,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(utilsDir, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

const busyGo = `package utils

func BusyLoop(repeat int) {
	for range repeat {
		for range 10000 {
		}
	}
}
`

const ennvarsGo = `package utils

import (
	"log"
	"os"
	"strconv"
)

func GetEnvVar(key string, required bool) string {
	value := os.Getenv(key)
	if value == "" && required {
		log.Fatalf("Environment variable %s is required", key)
	}
	return value
}

func StrToInt(s string) int {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Fatalf("Failed to convert string to int: %s", err)
	}
	return int(i)
}
`

const logGo = `package utils

import (
	"log"
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
	default:
		log.Printf("Invalid LOG_LEVEL '%s', defaulting to INFO\n", logLevel)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key != slog.TimeKey {
				return a
			}
			t := a.Value.Time()
			a.Value = slog.StringValue(t.Format("04:05.000000"))
			return a
		},
	}))
	return logger.With("package", name)
}
`

func writeGRPCClient(module string, outDir string) error {
	tmpl := `package pkg

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func GetConn(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}

func GetServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
	}
}
`
	t, _ := template.New("").Parse(tmpl)
	var b bytes.Buffer
	t.Execute(&b, map[string]string{"Module": module})
	pkgDir := filepath.Join(outDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pkgDir, "grpc.go"), b.Bytes(), 0644)
}

type entryServiceData struct {
	Module      string
	ServiceName string
	Port        int
	EntryNode   *Node
	Clients     []clientRef
	Downstreams []downstreamCall
}

type clientRef struct {
	Microservice string
	AddrEnv      string
}

type downstreamCall struct {
	Microservice string
	MethodName   string
}

func generateEntryService(pg *ParsedGraph, module string, svcName string, outDir string) error {
	entryNodeID := pg.EntryNodeID
	entryNode := pg.Nodes[entryNodeID]
	targets := pg.Downstream(entryNodeID)
	seen := make(map[string]bool)
	var clients []clientRef
	for _, t := range targets {
		n := pg.Nodes[t]
		if !seen[n.Microservice] {
			seen[n.Microservice] = true
			clients = append(clients, clientRef{n.Microservice, n.Microservice + "_ADDR"})
		}
	}
	var downstreams []downstreamCall
	for _, t := range targets {
		n := pg.Nodes[t]
		downstreams = append(downstreams, downstreamCall{n.Microservice, n.GoMethodName()})
	}
	data := entryServiceData{
		Module:      module,
		ServiceName: svcName,
		Port:        port,
		EntryNode:   entryNode,
		Clients:     clients,
		Downstreams: downstreams,
	}
	return renderTemplate(entryServiceTmpl, data, filepath.Join(outDir, "services", svcName, "main.go"))
}

var entryServiceTmpl = `package main

import (
	"fmt"
	"net/http"
	"{{.Module}}/pkg"
	pb "{{.Module}}/protobuf"
	"{{.Module}}/utils"
)

type Server struct {
{{range .Clients}}	{{.Microservice}}Client pb.{{.Microservice}}Client
{{end}}
}

const serviceName = "{{.ServiceName}}"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing HTTP server...")
{{range $i, $c := .Clients}}{{if eq $i 0}}	conn := pkg.GetConn(utils.GetEnvVar("{{$c.AddrEnv}}", true))
{{else}}	conn = pkg.GetConn(utils.GetEnvVar("{{$c.AddrEnv}}", true))
{{end}}	s.{{$c.Microservice}}Client = pb.New{{$c.Microservice}}Client(conn)
{{end}}
	mux := http.NewServeMux()
	mux.Handle("/{{.EntryNode.Interface}}", http.HandlerFunc(s.handler))
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", {{.Port}}),
		Handler: mux,
	}
	log.Info("Serving HTTP")
	return srv.ListenAndServe()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	utils.BusyLoop({{.EntryNode.BusyLoopRepeats}})
	ctx := r.Context()
	req := &pb.Request{}
	var err error
{{range .Downstreams}}	_, err = s.{{.Microservice}}Client.{{.MethodName}}(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}
{{end}}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func main() {
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start server", "error", err)
	}
}
`

type grpcServiceData struct {
	Module      string
	ServiceName string
	Port        int
	Handlers    []grpcHandler
	Clients     []clientRef
}

type grpcHandler struct {
	Node        *Node
	MethodName  string
	Downstreams []downstreamCall
}

func generateGRPCService(pg *ParsedGraph, module string, svcName string, outDir string) error {
	nodes := pg.Services[svcName]
	allClients := make(map[string]bool)
	var handlers []grpcHandler
	for _, n := range nodes {
		targets := pg.Downstream(n.ID)
		var downstreams []downstreamCall
		for _, t := range targets {
			tn := pg.Nodes[t]
			allClients[tn.Microservice] = true
			downstreams = append(downstreams, downstreamCall{tn.Microservice, tn.GoMethodName()})
		}
		handlers = append(handlers, grpcHandler{Node: n, MethodName: n.GoMethodName(), Downstreams: downstreams})
	}
	var clients []clientRef
	for ms := range allClients {
		clients = append(clients, clientRef{ms, ms + "_ADDR"})
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].Microservice < clients[j].Microservice })
	data := grpcServiceData{
		Module:      module,
		ServiceName: svcName,
		Port:        port,
		Handlers:    handlers,
		Clients:     clients,
	}
	return renderTemplate(grpcServiceTmpl, data, filepath.Join(outDir, "services", svcName, "main.go"))
}

var grpcServiceTmpl = `package main

import (
	"context"
	"fmt"
	"net"
	"{{.Module}}/pkg"
	pb "{{.Module}}/protobuf"
	"{{.Module}}/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	pb.Unimplemented{{.ServiceName}}Server
{{range .Clients}}	{{.Microservice}}Client pb.{{.Microservice}}Client
{{end}}
}

const serviceName = "{{.ServiceName}}"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	srv := grpc.NewServer(opts...)
	pb.Register{{.ServiceName}}Server(srv, s)
{{range $i, $c := .Clients}}{{if eq $i 0}}	var conn *grpc.ClientConn
	conn = pkg.GetConn(utils.GetEnvVar("{{$c.AddrEnv}}", true))
{{else}}	conn = pkg.GetConn(utils.GetEnvVar("{{$c.AddrEnv}}", true))
{{end}}	s.{{$c.Microservice}}Client = pb.New{{$c.Microservice}}Client(conn)
{{end}}
	reflection.Register(srv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", {{.Port}}))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}

{{range .Handlers}}
func (s *Server) {{.MethodName}}(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop({{.Node.BusyLoopRepeats}})
{{if .Downstreams}}	var err error
{{range .Downstreams}}	_, err = s.{{.Microservice}}Client.{{.MethodName}}(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
{{end}}{{end}}
	return &pb.Response{}, nil
}
{{end}}

func main() {
	s := &Server{}
	log.Info("Starting server...")
	if err := s.Run(); err != nil {
		log.Error("Server failed", "error", err)
	}
}
`

func renderTemplate(tmplStr string, data interface{}, path string) error {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0644)
}
