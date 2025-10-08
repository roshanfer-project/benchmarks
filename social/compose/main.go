package main

import (
	"context"
	"fmt"
	"net"
	"social"
	"sync"
	"sync/atomic"
	"time"

	breakwaterinit "social/breakwater-init"
	"social/dagor"
	dagorinit "social/dagor_init"
	oteltool "social/otel_tool"
	pb "social/protobuf"
	rajomoninit "social/rajomon_init"
	"social/utils"

	bw "social/breakwater"

	"github.com/pennsail/rajomon"
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

type ComposePostServer struct {
	pb.UnimplementedComposePostServer

	postClient pb.PostStorageClient
	homeClient pb.HomeTimelineClient
	userClient pb.UserTimelineClient
}

const serviceName = "compose"

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

func (s *ComposePostServer) Run() error {
	log.Info("Initializing gRPC server...")

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
	}

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
			priceTable.UnaryInterceptor,
			AcceptedRPCInterceptor()))
	}

	var dagorNode *dagor.Dagor
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
			dagorNode.UnaryInterceptorServer,
			AcceptedRPCInterceptor()))
	}

	var breakwater *bw.Breakwater
	if utils.GetEnvVar("breakwater", false) == "true" {
		log.Info("breakwater is enabled, configuring breakwater interceptor")
		breakwater = breakwaterinit.GetBreakwater(serviceName, true)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
			breakwater.UnaryInterceptor))
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
	pb.RegisterComposePostServer(srv, s)

	log.Info("Initializing gRPC clients...")
	var postsEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		postsEnv = "ComposeEgress"
	} else {
		postsEnv = "PostsAddr"
	}
	options := []grpc.DialOption{}
	if priceTable != nil {
		log.Debug("Using rajomon interceptor for posts client")
		options = append(options, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
	} else if dagorNode != nil {
		log.Debug("Using dagor interceptor for posts client")
		options = append(options, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}
	conn := social.GetConn(utils.GetEnvVar(postsEnv, true), options...)
	s.postClient = pb.NewPostStorageClient(conn)

	var home string
	if utils.GetEnvVar("sidecar", false) == "true" {
		home = "ComposeEgress"
	} else {
		home = "HomeAddr"
	}
	conn = social.GetConn(utils.GetEnvVar(home, true), options...)
	s.homeClient = pb.NewHomeTimelineClient(conn)

	var userEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		userEnv = "ComposeEgress"
	} else {
		userEnv = "UserAddr"
	}
	conn = social.GetConn(utils.GetEnvVar(userEnv, true), options...)
	s.userClient = pb.NewUserTimelineClient(conn)

	log.Info("Successful")

	log.Info("Successful")

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("ComposePort", true))))
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

	srv := &ComposePostServer{
		//MongoClient: mongoClient,
		//mutex: sync.Mutex{},
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func (s *ComposePostServer) ComposePost(ctx context.Context, req *pb.ComposePostRequest) (*pb.ComposePostResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "composepost")
	// Invoke store_post method in poststorage service
	req1 := &pb.StorePostRequest{
		CreatorId: req.CreatorId,
		Text:      req.Text,
	}
	//config.DebugLog("Composing post: %+v", req1)
	log.Debug("Compose posts", "post", req1)
	//resp1, err := invoke.Invoke[*pb.StorePostResponse](ctx, "poststorage", "StorePost", req1)
	resp1, err := s.postClient.StorePost(ctx, req1)
	if err != nil {
		log.Error("Error storing post", "error", err)
		return nil, err
	}
	log.Debug("Post stored successfully", "post", resp1)

	postId := resp1.PostId

	// Write to user timeline
	req2 := &pb.WriteUserTimelineRequest{
		UserId:  req.CreatorId,
		PostIds: []string{postId},
	}
	log.Debug("Writing to user timeline", "request", req2)
	//_, err = invoke.Invoke[*pb.WriteUserTimelineResponse](ctx, "usertimeline", "WriteUserTimeline", req2)
	_, err = s.userClient.WriteUserTimeline(ctx, req2)
	if err != nil {
		//config.DebugLog("Error writing to user timeline: %v", err)
		log.Error("Error writing to user timeline", "error", err)
		return nil, err
	}
	log.Debug("User timeline updated successfully")

	// Write to home timeline
	req3 := &pb.WriteHomeTimelineRequest{
		UserId:  req.CreatorId,
		PostIds: []string{postId},
	}
	log.Debug("Writing to home timeline", "request", req3)
	//_, err = invoke.Invoke[*pb.WriteHomeTimelineResponse](ctx, "hometimeline", "WriteHomeTimeline", req3)
	_, err = s.homeClient.WriteHomeTimeline(ctx, req3)
	if err != nil {
		log.Error("Error writing to home timeline", "error", err)
		return nil, err
	}
	log.Debug("Home timeline updated successfully")

	return &pb.ComposePostResponse{PostId: postId}, nil
}

func (s *ComposePostServer) ComposePostMulti(ctx context.Context, req *pb.ComposePostMultiRequest) (*pb.ComposePostMultiResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "composepost")
	// Invoke store_post_multi method in poststorage service
	req1 := &pb.StorePostMultiRequest{
		CreatorId: req.CreatorId,
		Text:      req.Text,
		Number:    req.Number,
	}
	log.Debug("Composing multiple posts", "request", req1)
	//resp1, err := invoke.Invoke[*pb.StorePostMultiResponse](ctx, "poststorage", "StorePostMulti", req1)
	resp1, err := s.postClient.StorePostMulti(ctx, req1)
	if err != nil {
		log.Error("Error storing multiple posts", "error", err)
		return nil, err
	}
	log.Debug("Multiple posts stored successfully", "response", resp1)

	postIds := resp1.PostIds

	// Write to user timeline
	req2 := &pb.WriteUserTimelineRequest{
		UserId:  req.CreatorId,
		PostIds: postIds,
	}
	log.Debug("Writing to user timeline", "request", req2)
	//_, err = invoke.Invoke[*pb.WriteUserTimelineResponse](ctx, "usertimeline", "WriteUserTimeline", req2)
	_, err = s.userClient.WriteUserTimeline(ctx, req2)
	if err != nil {
		log.Error("Error writing to user timeline", "error", err)
		return nil, err
	}
	log.Debug("User timeline updated successfully")

	// Write to home timeline
	req3 := &pb.WriteHomeTimelineRequest{
		UserId:  req.CreatorId,
		PostIds: postIds,
	}
	log.Debug("Writing to home timeline", "request", req3)
	//_, err = invoke.Invoke[*pb.WriteHomeTimelineResponse](ctx, "hometimeline", "WriteHomeTimeline", req3)
	_, err = s.homeClient.WriteHomeTimeline(ctx, req3)
	if err != nil {
		log.Error("Error writing to home timeline", "error", err)
		return nil, err
	}
	log.Debug("Home timeline updated successfully")

	return &pb.ComposePostMultiResponse{PostIds: postIds}, nil
}
