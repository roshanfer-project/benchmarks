package main

import (
	"context"
	"fmt"
	"net"
	"social"
	"strconv"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats/opentelemetry"
)

type HomeTimelineServer struct {
	pb.UnimplementedHomeTimelineServer

	postClient  pb.PostStorageClient
	graphClient pb.SocialGraphClient
}

const serviceName = "home"

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

func (s *HomeTimelineServer) Run() error {
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

	if utils.GetEnvVar("sidecar", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			//CountersInterceptor(),
			ContextPropagationInterceptor(),
		))
	}

	/* ctx := context.Background()
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
	} */

	srv := grpc.NewServer(opts...)
	pb.RegisterHomeTimelineServer(srv, s)

	log.Info("Initializing gRPC clients...")
	var graphEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		graphEnv = "HomeEgress"
	} else {
		graphEnv = "GraphAddr"
	}
	options := []grpc.DialOption{}
	if priceTable != nil {
		log.Debug("Using rajomon interceptor for graph client")
		options = append(options, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
	} else if dagorNode != nil {
		log.Debug("Using dagor interceptor for graph client")
		options = append(options, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}
	conn := social.GetConn(utils.GetEnvVar(graphEnv, true), options...)
	s.graphClient = pb.NewSocialGraphClient(conn)

	var postsEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		postsEnv = "HomeEgress"
	} else {
		postsEnv = "PostsAddr"
	}
	options = []grpc.DialOption{}
	if priceTable != nil {
		log.Debug("Using rajomon interceptor for posts client")
		options = append(options, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
	} else if dagorNode != nil {
		log.Debug("Using dagor interceptor for posts client")
		options = append(options, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}
	conn = social.GetConn(utils.GetEnvVar(postsEnv, true), options...)
	s.postClient = pb.NewPostStorageClient(conn)

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("HomePort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func main() {
	/* var ok error
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
	} */
	/* log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("GeoMongoAddress", true))
	defer mongoClose() */

	srv := &HomeTimelineServer{
		//MongoClient: mongoClient,
		//mutex: sync.Mutex{},
	}

	populate(utils.StrToInt(utils.GetEnvVar("num_of_posts", true)))

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func populate(numberOfposts int) {
	ctx := context.Background()
	var PostsId = make([]string, numberOfposts)
	for i := 0; i < numberOfposts; i++ {
		PostsId[i] = strconv.Itoa(i)
	}
	for _, follower := range []string{"user1"} {
		postIds, err := utils.GetState[[]string](ctx, follower+"-"+serviceName)
		if err != nil {
			postIds = []string{}
		}
		if len(postIds) >= 10 {
			postIds = postIds[1:]
		}
		postIds = append(postIds, PostsId...)
		err = utils.SetState(ctx, follower+"-"+serviceName, postIds)
		if err != nil {
			log.Error("Failed to set state for follower", "follower", follower, "error", err)
			panic(err)
		}
	}
}

func (s *HomeTimelineServer) ReadHomeTimeline(ctx context.Context, req *pb.ReadHomeTimelineRequest) (*pb.ReadHomeTimelineResponse, error) {
	//ctx = utils.PropagateMetadata(ctx, "hometimeline")
	postIds, err := utils.GetState[[]string](ctx, req.UserId+"-"+serviceName)
	if err != nil {
		log.Error("[ReadHomeTimeline] Error getting state", "userId", req.UserId, "error", err)
		return &pb.ReadHomeTimelineResponse{Posts: []*pb.Post{}}, nil
	}

	postsReq := &pb.ReadPostsRequest{PostIds: postIds}
	//postsResp, err := invoke.Invoke[*pb.ReadPostsResponse](ctx, "poststorage", "readposts", postsReq)
	postsResp, err := s.postClient.ReadPosts(ctx, postsReq)
	if err != nil {
		log.Error("[ReadHomeTimeline] Error reading posts", "userId", req.UserId, "error", err)
		return nil, err
	}

	return &pb.ReadHomeTimelineResponse{Posts: postsResp.Posts}, nil
}

func (s *HomeTimelineServer) WriteHomeTimeline(ctx context.Context, req *pb.WriteHomeTimelineRequest) (*pb.WriteHomeTimelineResponse, error) {
	//ctx = utils.PropagateMetadata(ctx, "hometimeline")
	followersReq := &pb.GetFollowersRequest{UserId: req.UserId}
	//followersResp, err := invoke.Invoke[*pb.GetFollowersResponse](ctx, "socialgraph", "getfollowers", followersReq)
	followersResp, err := s.graphClient.GetFollowers(ctx, followersReq)
	if err != nil {
		log.Error("Failed to get followers", "error", err)
		return nil, err
	}

	for _, follower := range followersResp.Followers {
		postIds, err := utils.GetState[[]string](ctx, follower+"-"+serviceName)
		if err != nil {
			postIds = []string{}
		}
		if len(postIds) >= 10 {
			postIds = postIds[1:]
		}
		postIds = append(postIds, req.PostIds...)
		err = utils.SetState(ctx, follower+"-"+serviceName, postIds)
		if err != nil {
			log.Error("Failed to set state for follower", "follower", follower, "error", err)
			return nil, err
		}
	}

	return &pb.WriteHomeTimelineResponse{Success: true}, nil
}
