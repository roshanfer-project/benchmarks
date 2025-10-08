package main

import (
	"context"
	"fmt"
	"net"
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
	"google.golang.org/grpc/stats/opentelemetry"
)

type GraphServer struct {
	pb.UnimplementedSocialGraphServer
}

const serviceName = "graph"

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

func (s *GraphServer) Run() error {
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
			dagorNode.UnaryInterceptorServer))
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
	pb.RegisterSocialGraphServer(srv, s)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("GraphPort", true))))
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

	srv := &GraphServer{
		//MongoClient: mongoClient,
		//mutex: sync.Mutex{},
	}

	// initialize Redis
	populateUsersAndFollows(
		srv,
		utils.StrToInt(utils.GetEnvVar("num_of_users", true)),
		utils.StrToInt(utils.GetEnvVar("num_of_followers", true)),
	)

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func populateUsersAndFollows(s *GraphServer, numOfUsers, numOfFollowers int) {

	ctx := context.Background()

	for i := 0; i < numOfUsers; i++ {
		userId := fmt.Sprintf("user%d", i)

		// Insert user
		_, err := s.InsertUser(ctx, &pb.InsertUserRequest{UserId: userId})
		if err != nil {
			//log.Printf("Failed to insert user %s: %v", userId, err)
			log.Error("Failed to insert user", "userId", userId, "error", err)
		} else {
			log.Debug("Inserted user", "userId", userId)
		}
	}

	for i := 0; i < numOfUsers; i++ {
		// Follow other users
		userId := fmt.Sprintf("user%d", i)
		// if i+1 < numOfUsers {
		// each user follows the last numOfFollowers users
		if i > numOfFollowers {
			for j := 0; j < numOfFollowers; j++ {
				followeeId := fmt.Sprintf("user%d", i-j)
				_, err := s.Follow(ctx, &pb.FollowRequest{
					FollowerId: userId,
					FolloweeId: followeeId,
				})
				if err != nil {
					log.Error("Failed to follow user", "followerId", userId, "followeeId", followeeId, "error", err)
				} else {
					log.Debug("User followed user", "followerId", userId, "followeeId", followeeId)
				}
			}
		} else {
			for j := 0; j < i; j++ {
				followeeId := fmt.Sprintf("user%d", i-j)
				_, err := s.Follow(ctx, &pb.FollowRequest{
					FollowerId: userId,
					FolloweeId: followeeId,
				})
				if err != nil {
					log.Error("Failed to follow user", "followerId", userId, "followeeId", followeeId, "error", err)
				} else {
					log.Debug("User followed user", "followerId", userId, "followeeId", followeeId)
				}
			}
		}
	}
}

// InsertUser inserts a user with an empty social graph
func (s *GraphServer) InsertUser(ctx context.Context, req *pb.InsertUserRequest) (*pb.InsertUserResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	sg := utils.SGVertex{
		UserId:    req.UserId,
		Followers: []string{},
		Followees: []string{},
	}
	err := utils.SetState(ctx, req.UserId, sg)
	if err != nil {
		//config.DebugLog("Error inserting user %s: %v", req.UserId, err)
		log.Debug("Error inserting user", "userId", req.UserId, "error", err)
		return nil, err
	}
	log.Debug("Inserted user", "userId", req.UserId)
	return &pb.InsertUserResponse{}, nil
}

// GetFollowers retrieves the list of followers for a given user
func (s *GraphServer) GetFollowers(ctx context.Context, req *pb.GetFollowersRequest) (*pb.GetFollowersResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	sg, err := utils.GetState[utils.SGVertex](ctx, req.UserId)
	if err != nil {
		log.Error("Error getting followers for user", "userId", req.UserId, "error", err)
		return nil, err
	}
	log.Debug("Retrieved followers for user", "userId", req.UserId, "followers", sg.Followers)
	return &pb.GetFollowersResponse{Followers: sg.Followers}, nil
}

// GetFollowees retrieves the list of followees for a given user
func (s *GraphServer) GetFollowees(ctx context.Context, req *pb.GetFolloweesRequest) (*pb.GetFolloweesResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	sg, err := utils.GetState[utils.SGVertex](ctx, req.UserId)
	if err != nil {
		log.Error("Error getting followees for user", "userId", req.UserId, "error", err)
		return nil, err
	}
	log.Debug("Retrieved followees for user", "userId", req.UserId, "followees", sg.Followees)
	return &pb.GetFolloweesResponse{Followees: sg.Followees}, nil
}

// Follow allows a user to follow another user
func (s *GraphServer) Follow(ctx context.Context, req *pb.FollowRequest) (*pb.FollowResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	err := s.follow(ctx, req.FollowerId, req.FolloweeId)
	if err != nil {
		log.Error("Error following user", "followerId", req.FollowerId, "followeeId", req.FolloweeId, "error", err)
		return nil, err
	}
	log.Debug("User followed user", "followerId", req.FollowerId, "followeeId", req.FolloweeId)
	return &pb.FollowResponse{}, nil
}

// follow is a helper function to handle the following logic
func (s *GraphServer) follow(ctx context.Context, followerId string, followeeId string) error {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	// Retrieve the follower's state
	sgFollower, err := utils.GetState[utils.SGVertex](ctx, followerId)
	if err != nil {
		log.Error("Error getting state for follower", "followerId", followerId, "error", err)
		sgFollower = utils.SGVertex{
			UserId:    followerId,
			Followers: []string{},
			Followees: []string{},
		}
	}
	log.Debug("Before following: follower", "followerId", followerId, "followees", sgFollower.Followees)
	// Add the followee to the follower's followees
	sgFollower.Followees = append(sgFollower.Followees, followeeId)
	err = utils.SetState(ctx, followerId, sgFollower)
	if err != nil {
		log.Error("Error setting state for follower", "followerId", followerId, "error", err)
		return err
	}
	log.Debug("After following: follower", "followerId", followerId, "followees", sgFollower.Followees)

	// Retrieve the followee's state
	sgFollowee, err := utils.GetState[utils.SGVertex](ctx, followeeId)
	if err != nil {
		log.Error("Error getting state for followee", "followeeId", followeeId, "error", err)
		sgFollowee = utils.SGVertex{
			UserId:    followeeId,
			Followers: []string{},
			Followees: []string{},
		}
	}
	log.Debug("Before following: followee", "followeeId", followeeId, "followers", sgFollowee.Followers)
	// Add the follower to the followee's followers
	sgFollowee.Followers = append(sgFollowee.Followers, followerId)
	err = utils.SetState(ctx, followeeId, sgFollowee)
	if err != nil {
		log.Error("Error setting state for followee", "followeeId", followeeId, "error", err)
		return err
	}
	log.Debug("After following: followee", "followeeId", followeeId, "followers", sgFollowee.Followers)
	return nil
}
