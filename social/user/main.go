package main

import (
	"context"
	"fmt"
	"net"
	"social"
	"strconv"

	"social/dagor"
	dagorinit "social/dagor_init"
	pb "social/protobuf"
	rajomoninit "social/rajomon_init"
	"social/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type UserTimelineServer struct {
	pb.UnimplementedUserTimelineServer

	postClient pb.PostStorageClient
}

const serviceName = "user"

var log = utils.GetLogger(serviceName)

func (s *UserTimelineServer) Run() error {
	log.Info("Initializing gRPC server...")

	opts := social.GetServerOptions()
	counter := utils.NewCounterState(serviceName)

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	}

	if utils.GetEnvVar("sidecar", false) == "true" {
		if utils.GetEnvVar("queuing_export", false) == "true" {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				counter.GetInterceptor()))
		} else {
			opts = append(opts, grpc.UnaryInterceptor(utils.ContextPropagationInterceptor()))
		}
	} else if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor()))
	}

	srv := grpc.NewServer(opts...)
	pb.RegisterUserTimelineServer(srv, s)

	log.Info("Initializing gRPC clients...")
	var postsEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		postsEnv = "UserEgress"
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

	log.Info("Successful")

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("UserPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func main() {
	srv := &UserTimelineServer{}

	populate(srv, utils.StrToInt(utils.GetEnvVar("num_of_users", true)), utils.StrToInt(utils.GetEnvVar("num_of_posts", true)))

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func populate(s *UserTimelineServer, numberOfUsers, numberOfposts int) {
	var postsId = make([]string, numberOfposts)
	for i := 0; i < numberOfposts; i++ {
		postsId[i] = strconv.Itoa(i)
	}
	for u := 0; u < numberOfUsers; u++ {
		userId := "user" + strconv.Itoa(u)
		req := &pb.WriteUserTimelineRequest{
			UserId:  userId,
			PostIds: postsId,
		}
		_, err := s.WriteUserTimeline(context.Background(), req)
		if err != nil {
			log.Error("[populate] Error writing user timeline", "userId", userId, "error", err)
			panic(err)
		}

	}
}

func (s *UserTimelineServer) ReadUserTimeline(ctx context.Context, req *pb.ReadUserTimelineRequest) (*pb.ReadUserTimelineResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "usertimeline")
	postIds, err := utils.GetState[[]string](ctx, req.UserId+"-"+serviceName)
	if err != nil {
		log.Error("[ReadUserTimeline] Error getting state", "userId", req.UserId, "error", err)
		return &pb.ReadUserTimelineResponse{Posts: []*pb.Post{}}, nil
	}

	postsReq := &pb.ReadPostsRequest{PostIds: postIds}
	//postsResp, err := invoke.Invoke[*pb.ReadPostsResponse](ctx, "poststorage", "readposts", postsReq)
	postsResp, err := s.postClient.ReadPosts(ctx, postsReq)
	if err != nil {
		log.Error("[ReadUserTimeline] Error reading posts", "userId", req.UserId, "error", err)
		return nil, err
	}

	return &pb.ReadUserTimelineResponse{Posts: postsResp.Posts}, nil
}

func (s *UserTimelineServer) WriteUserTimeline(ctx context.Context, req *pb.WriteUserTimelineRequest) (*pb.WriteUserTimelineResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "usertimeline")
	postIds, err := utils.GetState[[]string](ctx, req.UserId+"-"+serviceName)
	if err != nil {
		postIds = []string{}
	}
	if len(postIds) >= 10 {
		postIds = postIds[1:]
	}
	postIds = append(postIds, req.PostIds...)
	err = utils.SetState(ctx, req.UserId+"-"+serviceName, postIds)
	if err != nil {
		return nil, err
	}
	return &pb.WriteUserTimelineResponse{Success: true}, nil
}
