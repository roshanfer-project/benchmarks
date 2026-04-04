package gen

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"unicode"
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

func protoServiceName(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
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
		"busy.go":      busyGo,
		"ennvars.go":   ennvarsGo,
		"log.go":       logGo,
		"propagator.go": propagatorGo,
		"counter.go":   counterGo,
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

const propagatorGo = `package utils

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ContextPropagationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		if in, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = metadata.NewOutgoingContext(ctx, in)
		}
		return handler(ctx, req)
	}
}
`

const counterGo = `package utils

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var logCounter = GetLogger("counter")

type CounterState struct {
	failedRPCCounter        map[string]int64
	acceptedRPCCounter      map[string]int64
	inReq                   map[string]int64
	outReq                  map[string]int64
	maxQueue                map[string]int64
	queueIntegral           map[string]float64
	lock                    sync.Mutex
	startOnce               sync.Once
	startTime               time.Time
	lastEventTime           time.Time
	maxQueueGauge           *prometheus.GaugeVec
	avgQueueGauge           *prometheus.GaugeVec
	failedRPCCounterGauge   *prometheus.GaugeVec
	acceptedRPCCounterGauge *prometheus.GaugeVec
	registry                *prometheus.Registry
	promAddr                string
	serviceName             string
}

func NewCounterState(serviceName string) *CounterState {
	s := &CounterState{
		failedRPCCounter:   make(map[string]int64),
		acceptedRPCCounter: make(map[string]int64),
		inReq:              make(map[string]int64),
		outReq:             make(map[string]int64),
		maxQueue:           make(map[string]int64),
		queueIntegral:      make(map[string]float64),
		lock:               sync.Mutex{},
		registry:           prometheus.NewRegistry(),
		startTime:          time.Now(),
	}
	if strings.HasSuffix(serviceName, "-grpc") {
		s.serviceName = serviceName[:len(serviceName)-len("-grpc")]
	} else {
		s.serviceName = serviceName
	}
	s.promAddr = GetEnvVar("PROM_ADDR", false)

	s.maxQueueGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "max_queue", Help: "Maximum queue length for each RPC method"},
		[]string{"api"},
	)
	s.avgQueueGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "avg_queue", Help: "Time-averaged queue length for each RPC method"},
		[]string{"api"},
	)
	s.acceptedRPCCounterGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "accepted_rpc_counter", Help: "Accepted RPC counter for each RPC method"},
		[]string{"api"},
	)
	s.failedRPCCounterGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "failed_rpc_counter", Help: "Failed RPC counter for each RPC method"},
		[]string{"api"},
	)
	s.registry.MustRegister(s.maxQueueGauge)
	s.registry.MustRegister(s.avgQueueGauge)
	s.registry.MustRegister(s.acceptedRPCCounterGauge)
	s.registry.MustRegister(s.failedRPCCounterGauge)
	return s
}

func (s *CounterState) start() {
	s.startOnce.Do(func() {
		if s.promAddr != "" {
			go s.PushAll()
		}
	})
}

func (s *CounterState) GetInterceptor() grpc.UnaryServerInterceptor {
	s.start()
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			panic("metadata not found in context")
		}
		keys := md.Get("api")
		if len(keys) == 0 || len(keys) > 1 {
			panic("api not found in metadata")
		}
		api := keys[0]
		s.IncrementInReq(api)
		s.IncrementAcceptedRPCCounter(api)
		resp, err := handler(ctx, req)
		if err != nil {
			s.IncrementFailedRPCCounter(api)
		}
		s.IncrementOutReq(api)
		return resp, err
	}
}

func (s *CounterState) GetHTTP1Middleware() func(next http.Handler) http.Handler {
	s.start()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			api := strings.TrimPrefix(r.URL.Path, "/")
			if api == "" {
				api = "unknown"
			}
			s.IncrementInReq(api)
			s.IncrementAcceptedRPCCounter(api)
			next.ServeHTTP(w, r)
			s.IncrementOutReq(api)
		})
	}
}

func (s *CounterState) accumulateIntegral(now time.Time) {
	if s.lastEventTime.IsZero() {
		s.lastEventTime = now
		return
	}
	dt := now.Sub(s.lastEventTime).Seconds()
	if dt <= 0 {
		return
	}
	for m := range s.inReq {
		q := s.inReq[m] - s.outReq[m]
		s.queueIntegral[m] += float64(q) * dt
	}
	s.lastEventTime = now
}

func (s *CounterState) IncrementInReq(method string) {
	s.lock.Lock()
	s.accumulateIntegral(time.Now())
	s.inReq[method]++
	s.UpdateMaxQueue(method)
	s.lock.Unlock()
}

func (s *CounterState) IncrementOutReq(method string) {
	s.lock.Lock()
	s.accumulateIntegral(time.Now())
	s.outReq[method]++
	s.UpdateMaxQueue(method)
	s.lock.Unlock()
}

func (s *CounterState) UpdateMaxQueue(method string) {
	queueSize := s.inReq[method] - s.outReq[method]
	if queueSize > s.maxQueue[method] {
		s.maxQueue[method] = queueSize
	}
}

func (s *CounterState) IncrementAcceptedRPCCounter(method string) {
	s.lock.Lock()
	s.acceptedRPCCounter[method]++
	s.lock.Unlock()
}

func (s *CounterState) PushAcceptedRPCCounter() {
	s.lock.Lock()
	for method, count := range s.acceptedRPCCounter {
		s.acceptedRPCCounterGauge.WithLabelValues(method).Set(float64(count))
	}
	s.lock.Unlock()
}

func (s *CounterState) IncrementFailedRPCCounter(method string) {
	s.lock.Lock()
	s.failedRPCCounter[method]++
	s.lock.Unlock()
}

func (s *CounterState) PushFailedRPCCounter() {
	s.lock.Lock()
	for method, count := range s.failedRPCCounter {
		s.failedRPCCounterGauge.WithLabelValues(method).Set(float64(count))
	}
	s.lock.Unlock()
}

func (s *CounterState) PushAll() {
	t := time.NewTicker(1 * time.Second)
	for range t.C {
		s.lock.Lock()
		now := time.Now()
		s.accumulateIntegral(now)
		for method, queueSize := range s.maxQueue {
			s.maxQueueGauge.WithLabelValues(method).Set(float64(queueSize))
		}
		elapsed := now.Sub(s.startTime).Seconds()
		if elapsed < 1e-9 {
			elapsed = 1e-9
		}
		for method := range s.inReq {
			avg := s.queueIntegral[method] / elapsed
			s.avgQueueGauge.WithLabelValues(method).Set(avg)
		}
		s.lock.Unlock()
		s.PushAcceptedRPCCounter()
		s.PushFailedRPCCounter()
		if err := push.New(s.promAddr, s.serviceName).Gatherer(s.registry).Push(); err != nil {
			logCounter.Error("Could not push to Pushgateway", "error", err)
		}
	}
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
	Module         string
	ServiceName    string
	Port           int
	Handlers       []entryHandlerData
	Clients        []clientRef
	EgressEnv      string
	PortEnv        string
	UseSingleConn  bool
	NeedBenchRng   bool
	NeedParallel   bool
}

type entryHandlerData struct {
	Interface        string
	BusyLoopRepeats  int
	Bimodal          bool
	BimodalP0        float64
	BimodalR0        int
	BimodalR1        int
	HasWeighted      bool
	ParallelFanout   bool
	WeightedArms     []weightedArm
	Downstreams      []downstreamCall
}

type clientRef struct {
	Microservice     string
	ProtoMicroservice string
	AddrEnv          string
}

type downstreamCall struct {
	Microservice      string
	ProtoMicroservice string
	MethodName        string
	Interface         string
}

type weightedArm struct {
	Until             float64
	IsFirst           bool
	IsLast            bool
	ProtoMicroservice string
	MethodName        string
}

func buildWeightedArms(edges []Edge, pg *ParsedGraph) []weightedArm {
	var arms []weightedArm
	cum := 0.0
	for i, e := range edges {
		n := pg.Nodes[e.Target]
		cum += *e.Weight
		a := weightedArm{
			ProtoMicroservice: protoServiceName(n.Microservice),
			MethodName:        n.GoMethodName(),
		}
		if i == 0 {
			a.IsFirst = true
			a.Until = cum
		} else if i == len(edges)-1 {
			a.IsLast = true
		} else {
			a.Until = cum
		}
		arms = append(arms, a)
	}
	return arms
}

// routingFromEdges returns weighted arms, or sequential/parallel downstream list (parallelFanout only when unweighted multi-edge and all edges Parallel).
func routingFromEdges(pg *ParsedGraph, edges []Edge) (hasWeighted bool, parallelFanout bool, arms []weightedArm, seq []downstreamCall) {
	anyW := false
	for _, e := range edges {
		if e.Weight != nil {
			anyW = true
			break
		}
	}
	if !anyW {
		for _, e := range edges {
			tn := pg.Nodes[e.Target]
			seq = append(seq, downstreamCall{tn.Microservice, protoServiceName(tn.Microservice), tn.GoMethodName(), tn.Interface})
		}
		return false, IsParallelFanoutGroup(edges), nil, seq
	}
	if len(edges) == 1 {
		tn := pg.Nodes[edges[0].Target]
		seq = append(seq, downstreamCall{tn.Microservice, protoServiceName(tn.Microservice), tn.GoMethodName(), tn.Interface})
		return false, false, nil, seq
	}
	return true, false, buildWeightedArms(edges, pg), nil
}

func generateEntryService(pg *ParsedGraph, module string, svcName string, outDir string) error {
	seen := make(map[string]bool)
	var clients []clientRef
	egressEnv := svcName + "_EGRESS"
	var handlers []entryHandlerData
	needBenchRng := false
	needPar := false
	for _, entryNode := range pg.EntryInterfaces() {
		edges := pg.OutgoingEdgesForAPI(entryNode.ID, entryNode.Interface)
		for _, e := range edges {
			n := pg.Nodes[e.Target]
			if !seen[n.Microservice] {
				seen[n.Microservice] = true
				clients = append(clients, clientRef{n.Microservice, protoServiceName(n.Microservice), n.Microservice + "_ADDR"})
			}
		}
		hasW, par, arms, seq := routingFromEdges(pg, edges)
		if hasW {
			needBenchRng = true
		}
		if par {
			needPar = true
		}
		hd := entryHandlerData{
			Interface:      entryNode.Interface,
			HasWeighted:    hasW,
			ParallelFanout: par,
			WeightedArms:   arms,
			Downstreams:    seq,
		}
		if entryNode.Bimodal {
			needBenchRng = true
			hd.Bimodal = true
			hd.BimodalP0 = entryNode.BimodalP0
			hd.BimodalR0 = entryNode.BimodalR0
			hd.BimodalR1 = entryNode.BimodalR1
		} else {
			hd.BusyLoopRepeats = entryNode.BusyLoopRepeats()
		}
		handlers = append(handlers, hd)
	}
	data := entryServiceData{
		Module:        module,
		ServiceName:   svcName,
		Port:          port,
		Handlers:      handlers,
		Clients:       clients,
		EgressEnv:     egressEnv,
		PortEnv:       svcName + "_PORT",
		UseSingleConn: true,
		NeedBenchRng:  needBenchRng,
		NeedParallel:  needPar,
	}
	return renderTemplate(entryServiceTmpl, data, filepath.Join(outDir, "services", svcName, "main.go"))
}

var entryServiceTmpl = `package main

import (
	"fmt"
	"net/http"
	"strings"
	"{{.Module}}/utils"

	"google.golang.org/grpc/metadata"{{if .Clients}}
	"{{.Module}}/pkg"
	pb "{{.Module}}/protobuf"
	"google.golang.org/grpc"{{end}}{{if .NeedBenchRng}}
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"{{else if .NeedParallel}}
	"sync"{{end}}
)

{{if .NeedBenchRng}}var benchRng struct {
	mu sync.Mutex
	r  *rand.Rand
}

func init() {
	seed := time.Now().UnixNano()
	if s := os.Getenv("ROUTING_SEED"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			seed = v
		}
	}
	benchRng.r = rand.New(rand.NewSource(seed))
}

func benchFloat() float64 {
	benchRng.mu.Lock()
	defer benchRng.mu.Unlock()
	return benchRng.r.Float64()
}

{{end}}
type Server struct {
{{- range .Clients}}
	{{.ProtoMicroservice}}Client pb.{{.ProtoMicroservice}}Client
{{- end}}
}

const serviceName = "{{.ServiceName}}"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing HTTP server...")
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
{{if .Clients}}	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("{{.EgressEnv}}", true))
	}
{{range .Clients}}	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("{{.AddrEnv}}", true))
	}
	s.{{.ProtoMicroservice}}Client = pb.New{{.ProtoMicroservice}}Client(conn)
{{end}}
{{end}}
	port := {{.Port}}
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("{{.PortEnv}}", true))
	}
	mux := http.NewServeMux()
	var baseHandler http.Handler = http.HandlerFunc(s.handler)
	plain := utils.GetEnvVar("plain", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if plain || (sidecar && queuingExport) {
		counter := utils.NewCounterState(serviceName)
		baseHandler = counter.GetHTTP1Middleware()(baseHandler)
	}
{{range .Handlers}}	mux.Handle("/{{.Interface}}", baseHandler)
{{end}}
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	log.Info("Serving HTTP")
	return srv.ListenAndServe()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	var rpcID string
	if sidecar {
		rpcID = r.Header.Get("rpc-id")
		if rpcID == "" {
			http.Error(w, "rpc-id header required", http.StatusBadRequest)
			return
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	switch path {
{{range .Handlers}}	case "{{.Interface}}":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "{{.Interface}}", "rpc-id", rpcID))
{{if .Bimodal}}		u := benchFloat()
		if u < {{printf "%.9g" .BimodalP0}} {
			utils.BusyLoop({{.BimodalR0}})
		} else {
			utils.BusyLoop({{.BimodalR1}})
		}
{{else}}		utils.BusyLoop({{.BusyLoopRepeats}})
{{end}}
{{if .HasWeighted}}		u := benchFloat()
		req := &pb.Request{}
		var err error
{{range .WeightedArms}}{{if .IsFirst}}		if u < {{printf "%.9g" .Until}} {
{{else if .IsLast}}		} else {
{{else}}		} else if u < {{printf "%.9g" .Until}} {
{{end}}			_, err = s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
{{end}}		}
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
{{else if .ParallelFanout}}		req := &pb.Request{}
		var wg sync.WaitGroup
		var errMu sync.Mutex
		var firstErr error
{{range .Downstreams}}		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
			if e != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				errMu.Unlock()
			}
		}()
{{end}}		wg.Wait()
		if firstErr != nil {
			log.Error("downstream call failed", "error", firstErr)
			http.Error(w, firstErr.Error(), 500)
			return
		}
{{else if .Downstreams}}		req := &pb.Request{}
		var err error
{{range .Downstreams}}		_, err = s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
{{end}}
{{end}}
{{end}}	default:
		http.Error(w, "not found", 404)
		return
	}
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
	Module           string
	ServiceName      string
	ProtoServiceName string
	Port             int
	Handlers         []grpcHandler
	Clients          []clientRef
	EgressEnv        string
	PortEnv          string
	NeedBenchRng     bool
	NeedParallel     bool
}

type grpcAPICase struct {
	APIName        string
	HasWeighted    bool
	ParallelFanout bool
	WeightedArms   []weightedArm
	Downstreams    []downstreamCall
}

type grpcHandler struct {
	Node       *Node
	MethodName string
	APICases   []grpcAPICase
}

func generateGRPCService(pg *ParsedGraph, module string, svcName string, outDir string) error {
	nodes := pg.Services[svcName]
	allClients := make(map[string]bool)
	var handlers []grpcHandler
	needBenchRng := false
	needPar := false
	for _, n := range nodes {
		if n.Bimodal {
			needBenchRng = true
		}
		var cases []grpcAPICase
		for _, api := range pg.APIsReachingNode(n.ID) {
			edges := pg.OutgoingEdgesForAPI(n.ID, api)
			for _, e := range edges {
				tn := pg.Nodes[e.Target]
				allClients[tn.Microservice] = true
			}
			hasW, par, arms, seq := routingFromEdges(pg, edges)
			if hasW {
				needBenchRng = true
			}
			if par {
				needPar = true
			}
			cases = append(cases, grpcAPICase{APIName: api, HasWeighted: hasW, ParallelFanout: par, WeightedArms: arms, Downstreams: seq})
		}
		handlers = append(handlers, grpcHandler{Node: n, MethodName: n.GoMethodName(), APICases: cases})
	}
	var clients []clientRef
	for ms := range allClients {
		clients = append(clients, clientRef{ms, protoServiceName(ms), ms + "_ADDR"})
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].Microservice < clients[j].Microservice })
	data := grpcServiceData{
		Module:           module,
		ServiceName:      svcName,
		ProtoServiceName: protoServiceName(svcName),
		Port:             port,
		Handlers:         handlers,
		Clients:          clients,
		EgressEnv:        svcName + "_EGRESS",
		PortEnv:          svcName + "_PORT",
		NeedBenchRng:     needBenchRng,
		NeedParallel:     needPar,
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"{{if .NeedBenchRng}}
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"{{else if .NeedParallel}}
	"sync"{{end}}
)

{{if .NeedBenchRng}}var benchRng struct {
	mu sync.Mutex
	r  *rand.Rand
}

func init() {
	seed := time.Now().UnixNano()
	if s := os.Getenv("ROUTING_SEED"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			seed = v
		}
	}
	benchRng.r = rand.New(rand.NewSource(seed))
}

func benchFloat() float64 {
	benchRng.mu.Lock()
	defer benchRng.mu.Unlock()
	return benchRng.r.Float64()
}

{{end}}
type Server struct {
	pb.Unimplemented{{.ProtoServiceName}}Server
{{- range .Clients}}
	{{.ProtoMicroservice}}Client pb.{{.ProtoMicroservice}}Client
{{- end}}
}

const serviceName = "{{.ServiceName}}"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if sidecar {
		if queuingExport {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				utils.NewCounterState(serviceName).GetInterceptor()))
		} else {
			opts = append(opts, grpc.UnaryInterceptor(utils.ContextPropagationInterceptor()))
		}
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.Register{{.ProtoServiceName}}Server(srv, s)
{{if .Clients}}	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("{{.EgressEnv}}", true))
	}
{{range .Clients}}	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("{{.AddrEnv}}", true))
	}
	s.{{.ProtoMicroservice}}Client = pb.New{{.ProtoMicroservice}}Client(conn)
{{end}}
{{end}}
	reflection.Register(srv)
	port := {{.Port}}
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("{{.PortEnv}}", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}

{{range .Handlers}}
func (s *Server) {{.MethodName}}(ctx context.Context, req *pb.Request) (*pb.Response, error) {
{{if .Node.Bimodal}}	u := benchFloat()
	if u < {{printf "%.9g" .Node.BimodalP0}} {
		utils.BusyLoop({{.Node.BimodalR0}})
	} else {
		utils.BusyLoop({{.Node.BimodalR1}})
	}
{{else}}	utils.BusyLoop({{.Node.BusyLoopRepeats}})
{{end}}
	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
{{range .APICases}}	case "{{.APIName}}":
{{if .HasWeighted}}		u := benchFloat()
		var err error
{{range .WeightedArms}}{{if .IsFirst}}		if u < {{printf "%.9g" .Until}} {
{{else if .IsLast}}		} else {
{{else}}		} else if u < {{printf "%.9g" .Until}} {
{{end}}			_, err = s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
{{end}}		}
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
{{else if .ParallelFanout}}		var wg sync.WaitGroup
		var errMu sync.Mutex
		var firstErr error
{{range .Downstreams}}		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
			if e != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				errMu.Unlock()
			}
		}()
{{end}}		wg.Wait()
		if firstErr != nil {
			log.Error("downstream call failed", "error", firstErr)
			return nil, firstErr
		}
{{else if .Downstreams}}		var err error
{{range .Downstreams}}		_, err = s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
{{end}}
{{end}}
{{end}}	default:
	}
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
