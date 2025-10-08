package main

import (
	"context"
	"fmt"
	"net"
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

	"github.com/lithammer/shortuuid"
	"github.com/pennsail/rajomon"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "posts"

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

type PostStorageServer struct {
	pb.UnimplementedPostStorageServer
}

func (s *PostStorageServer) Run() error {
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		//grpc.UnaryInterceptor(tracingInterceptor),
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

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			priceTable.UnaryInterceptor,
			AcceptedRPCInterceptor(),
		))
		//opts = append(opts, grpc.UnaryInterceptor(priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor = nil
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			dagorNode.UnaryInterceptorServer,
			AcceptedRPCInterceptor()))
		//opts = append(opts, grpc.UnaryInterceptor(dagorNode.UnaryInterceptorServer))
	}

	var breakwaterd *bw.Breakwater
	if utils.GetEnvVar("breakwaterd", false) == "true" {
		log.Info("breakwaterd is enabled, configuring breakwaterd interceptor")
		breakwaterd = breakwaterinit.GetBreakwater(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			breakwaterd.UnaryInterceptor))
	}

	if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		opts = append(opts, grpc.UnaryInterceptor(CountersInterceptor()))
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterPostStorageServer(srv, s)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("PostsPort", true))))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
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

	srv := &PostStorageServer{
		//MongoClient: mongoClient,
		//mutex:       sync.Mutex{},
	}

	populatePosts(srv,
		utils.StrToInt(utils.GetEnvVar("num_of_posts", true)),
		utils.StrToInt(utils.GetEnvVar("num_of_users", true)))

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func populatePosts(s *PostStorageServer, numberOfPosts, numberOfUser int) {
	for ui := 0; ui < numberOfUser; ui++ {
		userId := "user" + strconv.Itoa(ui)

		posts := make(map[string]interface{}, numberOfPosts)
		postIds := make([]string, numberOfPosts)
		for i := 0; i < numberOfPosts; i++ {
			postId := strconv.Itoa(i)
			timestamp := time.Now().Unix()
			posts[postId] = pb.Post{
				PostId:    postId,
				CreatorId: userId,
				Text:      "This is a sample post",
				Timestamp: timestamp,
			}
			postIds[i] = postId
		}
		utils.SetBulkState(context.Background(), posts)
	}

}

func (s *PostStorageServer) StorePost(ctx context.Context, req *pb.StorePostRequest) (*pb.StorePostResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	postId := s.storePost(ctx, req.CreatorId, req.Text)
	return &pb.StorePostResponse{PostId: postId}, nil
}

func (s *PostStorageServer) StorePostMulti(ctx context.Context, req *pb.StorePostMultiRequest) (*pb.StorePostMultiResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	postIds := s.storePostMulti(ctx, req.CreatorId, req.Text, int(req.Number))
	return &pb.StorePostMultiResponse{PostIds: postIds}, nil
}

func (s *PostStorageServer) ReadPost(ctx context.Context, req *pb.ReadPostRequest) (*pb.ReadPostResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	post, err := utils.GetState[pb.Post](ctx, req.PostId)
	if err != nil {
		log.Error("[ReadPost] Error reading post", "postId", req.PostId, "error", err)
		return nil, err
	}
	return &pb.ReadPostResponse{Post: &post}, nil
}

func (s *PostStorageServer) ReadPostsHome(ctx context.Context, req *pb.ReadPostsRequest) (*pb.ReadPostsResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	retPosts, err := utils.GetBulkState[pb.Post](ctx, req.PostIds)
	if err != nil {
		log.Error("[ReadPosts] Error reading posts", "postIds", req.PostIds, "error", err)
		return nil, err
	}
	posts := make([]*pb.Post, len(retPosts))
	for i, post := range retPosts {
		posts[i] = &post
	}
	return &pb.ReadPostsResponse{Posts: posts}, nil
}

func (s *PostStorageServer) ReadPostsUser(ctx context.Context, req *pb.ReadPostsRequest) (*pb.ReadPostsResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	retPosts, err := utils.GetBulkState[pb.Post](ctx, req.PostIds)
	if err != nil {
		log.Error("[ReadPosts] Error reading posts", "postIds", req.PostIds, "error", err)
		return nil, err
	}
	posts := make([]*pb.Post, len(retPosts))
	for i, post := range retPosts {
		posts[i] = &post
	}
	return &pb.ReadPostsResponse{Posts: posts}, nil
}

func (s *PostStorageServer) storePost(ctx context.Context, creatorId string, text string) string {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	postIds := s.storePostMulti(ctx, creatorId, text, 1)
	return postIds[0]
}

func (s *PostStorageServer) storePostMulti(ctx context.Context, creatorId string, text string, number int) []string {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	posts := make(map[string]interface{}, number)
	postIds := make([]string, number)
	for i := 0; i < number; i++ {
		postId := shortuuid.New()
		timestamp := time.Now().Unix()
		posts[postId] = pb.Post{
			PostId:    postId,
			CreatorId: creatorId,
			Text:      text,
			Timestamp: timestamp,
		}
		postIds[i] = postId
	}
	utils.SetBulkState(ctx, posts)
	return postIds
}
