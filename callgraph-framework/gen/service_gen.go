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
	if err := writeUtils(pg, outDir); err != nil {
		return err
	}
	if err := writeGRPCClient(pg, module, outDir); err != nil {
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
	if err := generateEntryGrpcService(pg, module, outDir); err != nil {
		return err
	}
	return generateRajomonClientService(pg, module, outDir)
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

func writeUtils(pg *ParsedGraph, outDir string) error {
	utilsDir := filepath.Join(outDir, "utils")
	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		return err
	}
	counterContent, err := renderCounterGo(pg)
	if err != nil {
		return err
	}
	files := map[string]string{
		"busy.go":       busyGo,
		"ennvars.go":    ennvarsGo,
		"log.go":        logGo,
		"propagator.go": propagatorGo,
		"counter.go":    counterContent,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(utilsDir, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	_ = os.Remove(filepath.Join(utilsDir, "queuing_delay.go"))
	return nil
}

func renderCounterGo(pg *ParsedGraph) (string, error) {
	t, err := template.New("counter").Parse(counterGoTmpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]bool{
		"WRR":                 pg.LoadBalancingPolicy == "weighted_round_robin",
		"QueueingDelayExport": pg.HasFeature("queueing_delay_export"),
	}); err != nil {
		return "", err
	}
	return b.String(), nil
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

func StrToFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatalf("Failed to convert string to float64: %s", err)
	}
	return f
}

func ParseFloatString(value string) float64 {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return f
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

const counterGoTmpl = `package utils

import (
	"context"
	"net/http"
{{if .WRR}}	"os"
	"runtime"
{{end}}	"strings"
	"sync"
{{if .WRR}}	"syscall"
{{end}}	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
{{if .QueueingDelayExport}}	"google.golang.org/grpc/tap"
{{end}}{{if .WRR}}	"google.golang.org/grpc/orca"
{{end}}
)

var logCounter = GetLogger("counter")

{{if .QueueingDelayExport}}type tapTimeKey struct{}

func TapHandler(serviceName string) tap.ServerInHandle {
	return func(ctx context.Context, _ *tap.Info) (context.Context, error) {
		return context.WithValue(ctx, tapTimeKey{}, time.Now()), nil
	}
}

{{end}}func replicaInstanceSuffix(serviceName string) string {
	if GetEnvVar("plain_lb", false) != "true" {
		return ""
	}
	pod := GetEnvVar("POD_NAME", false)
	if pod == "" {
		return ""
	}
	if strings.HasPrefix(pod, serviceName+"-") {
		return strings.TrimPrefix(pod, serviceName+"-")
	}
	return pod
}

type CounterState struct {
	failedRPCCounter        map[string]int64
	acceptedRPCCounter      map[string]int64
	inReq                   map[string]int64
	outReq                  map[string]int64
	maxQueue                map[string]int64
	queueIntegral           map[string]float64
{{if .QueueingDelayExport}}	queuingDelay            *prometheus.HistogramVec
{{end}}	lock                    sync.Mutex
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
{{if .WRR}}	orcaWindowStart         time.Time
	orcaWindowCount         int64
	orcaQPS                 float64
	orcaErrorWindowStart    time.Time
	orcaErrorWindowCount    int64
	orcaEPS                 float64
	orcaCPUUtil             float64
	orcaLastCPUTime         float64
	orcaLastSample          time.Time
{{end}}}

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
{{if .QueueingDelayExport}}	s.queuingDelay = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:                            "queuing_delay_microseconds",
			Help:                            "gRPC queue delay before CounterState (tap to interceptor entry), in microseconds",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
		},
		[]string{"api"},
	)
{{end}}	s.registry.MustRegister(s.maxQueueGauge)
	s.registry.MustRegister(s.avgQueueGauge)
	s.registry.MustRegister(s.acceptedRPCCounterGauge)
	s.registry.MustRegister(s.failedRPCCounterGauge)
{{if .QueueingDelayExport}}	s.registry.MustRegister(s.queuingDelay)
{{end}}	return s
}

func (s *CounterState) start() {
	s.startOnce.Do(func() {
		if s.promAddr != "" {
			go s.PushAll()
		}
{{if .WRR}}		if os.Getenv("plain_lb") == "true" {
			go s.sampleCPULoop()
		}
{{end}}	})
}

{{if .WRR}}func rusageSeconds(ru *syscall.Rusage) float64 {
	return float64(ru.Utime.Sec) + float64(ru.Utime.Usec)/1e6 +
		float64(ru.Stime.Sec) + float64(ru.Stime.Usec)/1e6
}

func (s *CounterState) sampleCPULoop() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		var ru syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
			continue
		}
		cpuTime := rusageSeconds(&ru)
		now := time.Now()
		s.lock.Lock()
		if !s.orcaLastSample.IsZero() {
			deltaCPU := cpuTime - s.orcaLastCPUTime
			deltaWall := now.Sub(s.orcaLastSample).Seconds()
			if deltaWall > 0 {
				gmp := float64(runtime.GOMAXPROCS(0))
				if gmp < 1 {
					gmp = 1
				}
				util := deltaCPU / (deltaWall * gmp)
				if util > 1 {
					util = 1
				}
				s.orcaCPUUtil = util
			}
		}
		s.orcaLastCPUTime = cpuTime
		s.orcaLastSample = now
		s.lock.Unlock()
	}
}

func (s *CounterState) tickOrcaQPS() float64 {
	now := time.Now()
	if s.orcaWindowStart.IsZero() {
		s.orcaWindowStart = now
	}
	s.orcaWindowCount++
	elapsed := now.Sub(s.orcaWindowStart).Seconds()
	if elapsed >= 1.0 {
		s.orcaQPS = float64(s.orcaWindowCount) / elapsed
		s.orcaWindowCount = 0
		s.orcaWindowStart = now
		return s.orcaQPS
	}
	if s.orcaQPS > 0 {
		return s.orcaQPS
	}
	if elapsed < 1e-3 {
		elapsed = 1e-3
	}
	return float64(s.orcaWindowCount) / elapsed
}

func (s *CounterState) tickOrcaEPS(isError bool) float64 {
	now := time.Now()
	if s.orcaErrorWindowStart.IsZero() {
		s.orcaErrorWindowStart = now
	}
	if isError {
		s.orcaErrorWindowCount++
	}
	elapsed := now.Sub(s.orcaErrorWindowStart).Seconds()
	if elapsed >= 1.0 {
		s.orcaEPS = float64(s.orcaErrorWindowCount) / elapsed
		s.orcaErrorWindowCount = 0
		s.orcaErrorWindowStart = now
		return s.orcaEPS
	}
	if s.orcaEPS > 0 {
		return s.orcaEPS
	}
	if elapsed < 1e-3 {
		elapsed = 1e-3
	}
	return float64(s.orcaErrorWindowCount) / elapsed
}

func (s *CounterState) recordOrcaMetrics(ctx context.Context, isError bool) {
	if os.Getenv("plain_lb") != "true" {
		return
	}
	r := orca.CallMetricsRecorderFromContext(ctx)
	if r == nil {
		return
	}
	s.lock.Lock()
	qps := s.tickOrcaQPS()
	eps := s.tickOrcaEPS(isError)
	cpu := s.orcaCPUUtil
	s.lock.Unlock()
	r.SetQPS(qps)
	r.SetCPUUtilization(cpu)
	if eps > 0 {
		r.SetEPS(eps)
	} else {
		r.DeleteEPS()
	}
}

{{end}}func (s *CounterState) GetInterceptor() grpc.UnaryServerInterceptor {
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
{{if .QueueingDelayExport}}		if t0, ok := ctx.Value(tapTimeKey{}).(time.Time); ok {
			s.queuingDelay.WithLabelValues(api).Observe(float64(time.Since(t0).Microseconds()))
		}
{{end}}		s.IncrementInReq(api)
		s.IncrementAcceptedRPCCounter(api)
		resp, err := handler(ctx, req)
		if err != nil {
			s.IncrementFailedRPCCounter(api)
		}
		s.IncrementOutReq(api)
{{if .WRR}}		s.recordOrcaMetrics(ctx, err != nil)
{{end}}		return resp, err
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
		pusher := push.New(s.promAddr, s.serviceName).Gatherer(s.registry)
		if instance := replicaInstanceSuffix(s.serviceName); instance != "" {
			pusher = pusher.Grouping("instance", instance)
		}
		if err := pusher.Push(); err != nil {
			logCounter.Error("Could not push to Pushgateway", "error", err)
		} else {
{{if .QueueingDelayExport}}			s.queuingDelay.Reset()
{{end}}		}
	}
}
`

func writeGRPCClient(pg *ParsedGraph, module string, outDir string) error {
	tmpl := `package pkg

import (
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
{{if .WRR}}	"google.golang.org/grpc/orca"
	_ "google.golang.org/grpc/balancer/weightedroundrobin"
{{end}}{{if .LeastRequest}}	_ "google.golang.org/grpc/balancer/leastrequest"
{{end}}	_ "google.golang.org/grpc/resolver/dns"
)

func GetConn(addr string, extra ...grpc.DialOption) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if os.Getenv("plain_lb") == "true" {
		addr = "dns:///" + addr
		opts = append(opts, grpc.WithDefaultServiceConfig(` + "`{{.LBConfig}}`" + `))
	}
	opts = append(opts, extra...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}

func GetRajomonClient(addr string, interceptor grpc.DialOption) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), interceptor}
	if os.Getenv("plain_lb") == "true" {
		addr = "dns:///" + addr
		opts = append([]grpc.DialOption{grpc.WithDefaultServiceConfig(` + "`{{.LBConfig}}`" + `)}, opts...)
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}

func GetServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
	}
{{if .WRR}}	if os.Getenv("plain_lb") == "true" {
		opts = append(opts, orca.CallMetricsServerOption(nil))
	}
{{end}}	return opts
}
`
	lbConfig := `{"loadBalancingConfig":[{"least_request_experimental":{}}]}`
	switch pg.LoadBalancingPolicy {
	case "weighted_round_robin":
		lbConfig = `{"loadBalancingConfig":[{"weighted_round_robin":{"blackoutPeriod":"1s"}}]}`
	case "round_robin":
		lbConfig = `{"loadBalancingConfig":[{"round_robin":{}}]}`
	}
	t, err := template.New("grpc").Parse(tmpl)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, map[string]interface{}{
		"WRR":          pg.LoadBalancingPolicy == "weighted_round_robin",
		"LeastRequest": pg.LoadBalancingPolicy != "round_robin" && pg.LoadBalancingPolicy != "weighted_round_robin",
		"LBConfig":     lbConfig,
	}); err != nil {
		return err
	}
	pkgDir := filepath.Join(outDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pkgDir, "grpc.go"), b.Bytes(), 0644)
}

type entryServiceData struct {
	Module        string
	ServiceName   string
	Port          int
	Handlers      []entryHandlerData
	Clients       []clientRef
	EgressEnv     string
	PortEnv       string
	UseSingleConn bool
	NeedBenchRng  bool
	NeedParallel  bool
}

type entryHandlerData struct {
	Interface       string
	BusyLoopRepeats int
	Exponential     bool
	ExponentialMean float64
	Bimodal         bool
	BimodalP0       float64
	BimodalR0       int
	BimodalR1       int
	HasWeighted     bool
	ParallelFanout  bool
	WeightedArms    []weightedArm
	Downstreams     []downstreamCall
}

type clientRef struct {
	Microservice      string
	ProtoMicroservice string
	AddrEnv           string
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
		} else if entryNode.Exponential {
			needBenchRng = true
			hd.Exponential = true
			hd.ExponentialMean = entryNode.ExponentialMean
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
	"sync/atomic"
	"time"{{else}}
	"sync/atomic"{{end}}{{if .NeedParallel}}
	"sync"{{end}}
)

var envoyRPCSeq uint64

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

func benchExpBusyLoop(mean float64) {
	benchRng.mu.Lock()
	rt := benchRng.r.ExpFloat64() * mean
	benchRng.mu.Unlock()
	repeats := int(rt * 320)
	if repeats < 1 {
		repeats = 1
	}
	utils.BusyLoop(repeats)
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
	envoy := utils.GetEnvVar("envoy", false) == "true"
	if sidecar && envoy {
		panic("sidecar and envoy cannot both be enabled")
	}
	meshProxy := sidecar || envoy
{{if .Clients}}	var conn *grpc.ClientConn
	if meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("{{.EgressEnv}}", true))
	}
{{range .Clients}}	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("{{.AddrEnv}}", true))
	}
	s.{{.ProtoMicroservice}}Client = pb.New{{.ProtoMicroservice}}Client(conn)
{{end}}
{{end}}
	port := {{.Port}}
	if meshProxy {
		port = utils.StrToInt(utils.GetEnvVar("{{.PortEnv}}", true))
	}
	mux := http.NewServeMux()
	var baseHandler http.Handler = http.HandlerFunc(s.handler)
	plain := utils.GetEnvVar("plain", false) == "true"
	plainLb := utils.GetEnvVar("plain_lb", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if plain || plainLb || (sidecar && queuingExport) {
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
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	envoy := utils.GetEnvVar("envoy", false) == "true"
	var rpcID, rpcLocalID, deadline string
	if sidecar {
		rpcID = r.Header.Get("rpc-id")
		if rpcID == "" {
			http.Error(w, "rpc-id header required", http.StatusBadRequest)
			return
		}
		rpcLocalID = r.Header.Get("rpc-local-id")
		if rpcLocalID == "" {
			http.Error(w, "rpc-local-id header required", http.StatusBadRequest)
			return
		}
		deadline = r.Header.Get("deadline")
		if deadline == "" {
			http.Error(w, "deadline header required", http.StatusBadRequest)
			return
		}
	} else if envoy {
		rpcID = r.Header.Get("rpc-id")
		rpcLocalID = r.Header.Get("rpc-local-id")
		if rpcID == "" {
			rpcID = fmt.Sprintf("%d", atomic.AddUint64(&envoyRPCSeq, 1))
		}
		if rpcLocalID == "" {
			rpcLocalID = rpcID
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	ctx := r.Context()
	switch path {
{{range .Handlers}}	case "{{.Interface}}":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "{{.Interface}}", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
{{if .Exponential}}		benchExpBusyLoop({{printf "%.9g" .ExponentialMean}})
{{else if .Bimodal}}		u := benchFloat()
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
{{$api := .Interface}}{{range .Downstreams}}		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "{{$api}}", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.{{.ProtoMicroservice}}Client.{{.MethodName}}(ctx, req)
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
	Module                string
	ServiceName           string
	ProtoServiceName      string
	Port                  int
	Handlers              []grpcHandler
	Clients               []clientRef
	EgressEnv             string
	PortEnv               string
	NeedBenchRng          bool
	NeedParallel          bool
	QueueingDelayExport   bool
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

func buildGRPCServiceData(pg *ParsedGraph, module string, svcName string) grpcServiceData {
	nodes := pg.Services[svcName]
	allClients := make(map[string]bool)
	var handlers []grpcHandler
	needBenchRng := false
	needPar := false
	for _, n := range nodes {
		if n.Bimodal || n.Exponential {
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
	return grpcServiceData{
		Module:              module,
		ServiceName:         svcName,
		ProtoServiceName:    protoServiceName(svcName),
		Port:                port,
		Handlers:            handlers,
		Clients:             clients,
		EgressEnv:           svcName + "_EGRESS",
		PortEnv:             svcName + "_PORT",
		NeedBenchRng:        needBenchRng,
		NeedParallel:        needPar,
		QueueingDelayExport: pg.HasFeature("queueing_delay_export"),
	}
}

func generateGRPCService(pg *ParsedGraph, module string, svcName string, outDir string) error {
	data := buildGRPCServiceData(pg, module, svcName)
	return renderTemplate(grpcServiceTmpl, data, filepath.Join(outDir, "services", svcName, "main.go"))
}

type entryGrpcBundle struct {
	grpcServiceData
	EntryGrpcK8s string
}

func generateEntryGrpcService(pg *ParsedGraph, module string, outDir string) error {
	entry := pg.EntryMicroservice()
	data := buildGRPCServiceData(pg, module, entry)
	bundle := entryGrpcBundle{grpcServiceData: data, EntryGrpcK8s: EntryGrpcK8s(pg)}
	return renderTemplate(entryGrpcServiceTmpl, bundle, filepath.Join(outDir, "services", bundle.EntryGrpcK8s, "main.go"))
}

type rajomonClientHandler struct {
	Interface  string
	MethodName string
}

type rajomonClientData struct {
	Module           string
	ProtoServiceName string
	Handlers         []rajomonClientHandler
}

func generateRajomonClientService(pg *ParsedGraph, module string, outDir string) error {
	entry := pg.EntryMicroservice()
	var handlers []rajomonClientHandler
	for _, n := range pg.EntryInterfaces() {
		handlers = append(handlers, rajomonClientHandler{Interface: n.Interface, MethodName: n.GoMethodName()})
	}
	sort.Slice(handlers, func(i, j int) bool { return handlers[i].Interface < handlers[j].Interface })
	data := rajomonClientData{
		Module:           module,
		ProtoServiceName: protoServiceName(entry),
		Handlers:         handlers,
	}
	return renderTemplate(rajomonClientTmpl, data, filepath.Join(outDir, "services", "rajomon-client", "main.go"))
}

var grpcServiceTmpl = `package main

import (
	"context"
	"fmt"
	"net"
	"{{.Module}}/pkg"
	pb "{{.Module}}/protobuf"
	dagor "{{.Module}}/dagor"
	dagorinit "{{.Module}}/dagor_init"
	rajomoninit "{{.Module}}/rajomon_init"
	"{{.Module}}/utils"

	"github.com/pennsail/rajomon"
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

func benchExpBusyLoop(mean float64) {
	benchRng.mu.Lock()
	rt := benchRng.r.ExpFloat64() * mean
	benchRng.mu.Unlock()
	repeats := int(rt * 320)
	if repeats < 1 {
		repeats = 1
	}
	utils.BusyLoop(repeats)
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
	envoy := utils.GetEnvVar("envoy", false) == "true"
	if sidecar && envoy {
		panic("sidecar and envoy cannot both be enabled")
	}
	meshProxy := sidecar || envoy
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if !meshProxy && useRajomon && useDagor {
		panic("rajomon and dagor cannot both be enabled")
	}
	var priceTable *rajomon.PriceTable
	var dagorNode *dagor.Dagor
	if useRajomon && !meshProxy {
		priceTable = rajomoninit.GetPriceTable(rajomoninit.InstanceName(serviceName), false)
	}
	if useDagor && !meshProxy {
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
	}
	if meshProxy {
		if sidecar && queuingExport {
{{if .QueueingDelayExport}}			opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
{{end}}			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				utils.NewCounterState(serviceName).GetInterceptor()))
		} else {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor()))
		}
	} else if useRajomon {
{{if .QueueingDelayExport}}		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
{{end}}		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			priceTable.UnaryInterceptor))
	} else if useDagor {
{{if .QueueingDelayExport}}		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
{{end}}		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	} else {
{{if .QueueingDelayExport}}		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
{{end}}		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.Register{{.ProtoServiceName}}Server(srv, s)
{{if .Clients}}	var conn *grpc.ClientConn
	if meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("{{.EgressEnv}}", true))
	}
{{range .Clients}}	if !meshProxy {
		addr := utils.GetEnvVar("{{.AddrEnv}}", true)
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
		} else if useDagor {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr)
		}
	}
	s.{{.ProtoMicroservice}}Client = pb.New{{.ProtoMicroservice}}Client(conn)
{{end}}
{{end}}
	reflection.Register(srv)
	port := {{.Port}}
	if meshProxy {
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
{{if .Node.Exponential}}	benchExpBusyLoop({{printf "%.9g" .Node.ExponentialMean}})
{{else if .Node.Bimodal}}	u := benchFloat()
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

var entryGrpcServiceTmpl = `package main

import (
	"context"
	"fmt"
	"net"
	"{{.Module}}/pkg"
	pb "{{.Module}}/protobuf"
	dagor "{{.Module}}/dagor"
	dagorinit "{{.Module}}/dagor_init"
	rajomoninit "{{.Module}}/rajomon_init"
	"{{.Module}}/utils"

	"github.com/pennsail/rajomon"
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

func benchExpBusyLoop(mean float64) {
	benchRng.mu.Lock()
	rt := benchRng.r.ExpFloat64() * mean
	benchRng.mu.Unlock()
	repeats := int(rt * 320)
	if repeats < 1 {
		repeats = 1
	}
	utils.BusyLoop(repeats)
}

{{end}}
type Server struct {
	pb.Unimplemented{{.ProtoServiceName}}Server
{{- range .Clients}}
	{{.ProtoMicroservice}}Client pb.{{.ProtoMicroservice}}Client
{{- end}}
}

const serviceName = "{{.EntryGrpcK8s}}"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	if useRajomon == useDagor {
		panic("entry-grpc requires exactly one of rajomon=true or dagor=true")
	}
	opts := pkg.GetServerOptions()
	var pt *rajomon.PriceTable
	var dn *dagor.Dagor
	if useRajomon {
		pt = rajomoninit.GetPriceTable(rajomoninit.InstanceName(serviceName), false)
{{if .QueueingDelayExport}}		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
{{end}}		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			pt.UnaryInterceptor))
	} else {
		dn = dagorinit.GetDagorNode(serviceName, true, false)
{{if .QueueingDelayExport}}		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
{{end}}		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dn.UnaryInterceptorServer))
	}
	srv := grpc.NewServer(opts...)
	pb.Register{{.ProtoServiceName}}Server(srv, s)
{{if .Clients}}{{range .Clients}}	{
		addr := utils.GetEnvVar("{{.AddrEnv}}", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
		s.{{.ProtoMicroservice}}Client = pb.New{{.ProtoMicroservice}}Client(conn)
	}
{{end}}{{end}}
	reflection.Register(srv)
	listenPort := utils.StrToInt(utils.GetEnvVar("EntryGRPCPort", true))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}

{{range .Handlers}}
func (s *Server) {{.MethodName}}(ctx context.Context, req *pb.Request) (*pb.Response, error) {
{{if .Node.Exponential}}	benchExpBusyLoop({{printf "%.9g" .Node.ExponentialMean}})
{{else if .Node.Bimodal}}	u := benchFloat()
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

var rajomonClientTmpl = `package main

import (
	"fmt"
	"net/http"
	"os"
	"{{.Module}}/pkg"
	pb "{{.Module}}/protobuf"
	dagorinit "{{.Module}}/dagor_init"
	rajomoninit "{{.Module}}/rajomon_init"
	"{{.Module}}/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var log = utils.GetLogger("rajomon-client")

type Server struct {
	client     pb.{{.ProtoServiceName}}Client
	grpcTarget string
}

func (s *Server) Run() error {
	log.Info("Initializing rajomon-client...")
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	if useRajomon == useDagor {
		panic("rajomon-client requires exactly one of rajomon=true or dagor=true")
	}
	addr := utils.GetEnvVar("EntryGRPCAddr", true)
	clientPort := utils.GetEnvVar("ClientPort", true)
	deployment := utils.GetEnvVar("deployment", false)
	if hn, err := os.Hostname(); err == nil {
		log.Info("pod identity", "hostname", hn)
	}
	log.Info("rajomon-client config",
		"EntryGRPCAddr", addr,
		"ClientPort", clientPort,
		"deployment", deployment,
		"protoService", "{{.ProtoServiceName}}",
	)
	var conn *grpc.ClientConn
	if useRajomon {
		pt := rajomoninit.GetPriceTable("client", true)
		log.Info("creating gRPC client (connection is lazy until first RPC)", "target", addr)
		conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorEnduser))
	} else {
		dn := dagorinit.GetDagorNode("client", false, true)
		log.Info("creating gRPC client (connection is lazy until first RPC)", "target", addr)
		conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
	}
	s.grpcTarget = addr
	s.client = pb.New{{.ProtoServiceName}}Client(conn)
	log.Info("gRPC stub ready", "target", addr)
	mux := http.NewServeMux()
{{range .Handlers}}	mux.HandleFunc("/{{.Interface}}", s.handle_{{.MethodName}})
{{end}}
	httpPort := utils.StrToInt(clientPort)
	log.Info("Serving HTTP", "listenAddr", fmt.Sprintf(":%d", httpPort), "grpcTarget", addr)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: mux,
	}
	return srv.ListenAndServe()
}

{{range .Handlers}}
func (s *Server) handle_{{.MethodName}}(w http.ResponseWriter, r *http.Request) {
	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "{{.Interface}}", "api", "{{.Interface}}")
	_, err := s.client.{{.MethodName}}(ctx, &pb.Request{})
	if err != nil {
		st := status.Code(err)
		log.Error("RPC failed",
			"error", err,
			"grpcCode", st.String(),
			"grpcTarget", s.grpcTarget,
			"grpcMethod", "{{.MethodName}}",
			"api", "{{.Interface}}",
		)
		if st == codes.ResourceExhausted {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}
{{end}}

func main() {
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start", "error", err)
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
