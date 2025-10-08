package main

import (
	"context"
	"fmt"
	"hol"
	oteltool "hol/otel_tool"
	pb "hol/protobuf"
	"hol/utils"
	"math"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats/opentelemetry"
)

type BackendService struct {
	pb.UnimplementedHOLServiceServer

	client pb.HOLServiceClient
}

var chainIndex = utils.StrToInt(utils.GetEnvVar("index", true))
var chainLength = utils.StrToInt(utils.GetEnvVar("length", true))
var procTimeSlow = utils.StrToInt(utils.GetEnvVar("proc_time_slow", true))
var procTimeFast = utils.StrToInt(utils.GetEnvVar("proc_time_fast", true))
var serviceName = "backend" + utils.GetEnvVar("index", true)

var log = utils.GetLogger(serviceName)

func ContextPropagationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		if in, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = metadata.NewOutgoingContext(ctx, in)
		}
		return handler(ctx, req)
	}
}

type CounterState struct {
	failedRPCCounter   atomic.Int64
	acceptedRPCCounter atomic.Int64
	inReq              sync.Map
	outReq             sync.Map
	maxQueue           sync.Map
	lock               sync.Mutex
}

func (s *CounterState) IncrementInReq(method string) {
	count, _ := s.inReq.LoadOrStore(method, int64(0))
	s.inReq.Store(method, count.(int64)+1)
}

func (s *CounterState) IncrementOutReq(method string) {
	count, _ := s.outReq.LoadOrStore(method, int64(0))
	s.outReq.Store(method, count.(int64)+1)
}

func (s *CounterState) IncrementMaxQueue(method string, value int64) {
	count, _ := s.maxQueue.LoadOrStore(method, int64(0))
	if value > count.(int64) {
		s.maxQueue.Store(method, value)
	}
}

func (s *CounterState) GetMaxQueue(method string) int64 {
	count, ok := s.maxQueue.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

func (s *CounterState) GetFailedRPCCounter() int64 {
	return s.failedRPCCounter.Load()
}

func (s *CounterState) IncrementFailedRPCCounter() {
	s.failedRPCCounter.Add(1)
}

func (s *CounterState) GetInReq(method string) int64 {
	count, ok := s.inReq.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

func (s *CounterState) GetOutReq(method string) int64 {
	count, ok := s.outReq.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

var maxQueueGuage metric.Int64Gauge
var failedRPCCounterGauge metric.Int64Counter
var acceptedRPCCounterGauge metric.Int64Counter
var counters = &CounterState{}

func AcceptedRPCInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// get metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Error("metadata not found in context")
			return nil, fmt.Errorf("metadata not found in context")
		}
		method := md.Get("method")
		if len(method) == 0 || len(method) > 1 {
			log.Error("method not found in metadata", "metadata", md)
			return nil, fmt.Errorf("method not found in metadata")
		}

		counters.lock.Lock()
		counters.acceptedRPCCounter.Add(1)
		acceptedRPCCounterGauge.Add(ctx, 1, metric.WithAttributes(
			attribute.String("api", method[0]),
		))
		counters.lock.Unlock()
		return handler(ctx, req)
	}
}

func CountersInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// get metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Error("metadata not found in context")
			return nil, fmt.Errorf("metadata not found in context")
		}
		method := md.Get("method")
		if len(method) == 0 || len(method) > 1 {
			log.Error("method not found in metadata", "metadata", md)
			return nil, fmt.Errorf("method not found in metadata")
		}

		counters.lock.Lock()
		counters.IncrementInReq(method[0])
		queueSize := counters.GetInReq(method[0]) - counters.GetOutReq(method[0])
		if queueSize > counters.GetMaxQueue(method[0]) {
			counters.IncrementMaxQueue(method[0], queueSize)
			maxQueueGuage.Record(ctx, queueSize, metric.WithAttributes(
				attribute.String("api", method[0]),
			))
		}
		counters.lock.Unlock()
		resp, err := handler(ctx, req)
		counters.lock.Lock()
		if err != nil {
			counters.IncrementFailedRPCCounter()
			failedRPCCounterGauge.Add(ctx, 1, metric.WithAttributes(
				attribute.String("api", method[0]),
			))
		}
		counters.IncrementOutReq(method[0])
		counters.lock.Unlock()
		return resp, err
	}
}

func configOTL(ctx context.Context, serviceName string) (grpc.ServerOption, []func(context.Context) error, bool) {
	if shutdownList, ok := oteltool.InitializeOTel(ctx, serviceName, false); ok {
		//tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")
		//meter = otel.GetMeterProvider().Meter(serviceName + "-meter")
		return opentelemetry.ServerOption(opentelemetry.Options{
			MetricsOptions: opentelemetry.MetricsOptions{MeterProvider: otel.GetMeterProvider()}}), shutdownList, true
	} else {
		return nil, nil, false
	}

}

func (s *BackendService) Run() error {
	log.Info("Initializing gRPC server...")

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
	}

	if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
		))
	}

	ctx := context.Background()
	if _, shutdownList, ok := configOTL(ctx, serviceName); ok {
		opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))

		for _, f := range shutdownList {
			defer func() {
				if err := f(ctx); err != nil {
					log.Error("main", "failed to shutdown OpenTelemetry provider", err)
				}
			}()
		}
		log.Info("Successfully initialized OpenTelemetry")
	} else {
		log.Error("Failed to initialize OpenTelemetry")
	}

	srv := grpc.NewServer(opts...)
	pb.RegisterHOLServiceServer(srv, s)

	if chainIndex != chainLength-1 {
		log.Info("Initializing gRPC clients...")
		var backendEnv string
		if utils.GetEnvVar("sidecar", false) == "true" {
			backendEnv = "index" + strconv.Itoa(chainIndex) + "Egress"
		} else {
			backendEnv = "index" + strconv.Itoa(chainIndex+1) + "Addr"
		}

		conn := hol.GetConn(utils.GetEnvVar(backendEnv, true))
		s.client = pb.NewHOLServiceClient(conn)
	}

	log.Info("Successful")

	reflection.Register(srv)

	listenEnv := "index" + strconv.Itoa(chainIndex) + "Port"
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar(listenEnv, true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)

}

func main() {
	var ok error
	maxQueueGuage, ok = otel.GetMeterProvider().Meter(serviceName).Int64Gauge("max_queue",
		metric.WithDescription("Maximum queue length for each RPC method"))
	if ok != nil {
		log.Error("Failed to create max_queue gauge")
		panic("Failed to create max_queue gauge")
	}
	failedRPCCounterGauge, ok = otel.GetMeterProvider().Meter(serviceName).Int64Counter("failed_rpc",
		metric.WithDescription("Total number of failed RPC calls"))
	if ok != nil {
		log.Error("Failed to create failed_rpc counter")
		panic("Failed to create failed_rpc counter")
	}
	acceptedRPCCounterGauge, ok = otel.GetMeterProvider().Meter(serviceName).Int64Counter("accepted_rpc",
		metric.WithDescription("Total number of accepted RPC calls"))
	if ok != nil {
		log.Error("Failed to create accepted_rpc counter")
		panic("Failed to create accepted_rpc counter")
	}
	/* log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("GeoMongoAddress", true))
	defer mongoClose() */

	srv := &BackendService{
		//MongoClient: mongoClient,
		//mutex: sync.Mutex{},
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func busyWait() {
	for i := 1; i < 1000; i++ {
		math.Sqrt(float64(i))
	}
}

func fastRPCProc() {
	remain := float32(procTimeFast)
	for {
		start := time.Now()
		math.Sqrt(float64(remain))
		remain -= float32(time.Since(start).Microseconds())
		if remain <= 0 {
			break
		}
	}
}

func slowRPCProc() {
	remain := float32(procTimeSlow)
	for {
		start := time.Now()
		math.Sqrt(float64(remain))
		remain -= float32(time.Since(start).Microseconds())
		if remain <= 0 {
			break
		}
	}
}

func (s *BackendService) FastRPC(ctx context.Context, req *pb.FastRPCRequest) (*pb.FastRPCResponse, error) {
	if chainIndex != chainLength-1 {
		resp, err := s.client.FastRPC(ctx, &pb.FastRPCRequest{Test: strconv.Itoa(chainIndex)})
		if err != nil {
			log.Error("FastRPC", "error", err)
			return nil, err
		}
		return resp, nil
	} else {
		fastRPCProc()
		return &pb.FastRPCResponse{Test: "FastRPC"}, nil
	}
}

func (s *BackendService) SlowRPC(ctx context.Context, req *pb.SlowRPCRequest) (*pb.SlowRPCResponse, error) {
	if chainIndex != chainLength-1 {
		resp, err := s.client.SlowRPC(ctx, &pb.SlowRPCRequest{Test: strconv.Itoa(chainIndex)})
		if err != nil {
			log.Error("SlowRPC", "error", err)
			return nil, err
		}
		return resp, nil
	} else {
		slowRPCProc()
		return &pb.SlowRPCResponse{Test: "SlowRPC"}, nil
	}
}
