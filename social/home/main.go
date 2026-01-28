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

type HomeTimelineServer struct {
	pb.UnimplementedHomeTimelineServer

	postClient  pb.PostStorageClient
	graphClient pb.SocialGraphClient
}

const serviceName = "home"

var log = utils.GetLogger(serviceName)

func (s *HomeTimelineServer) Run() error {
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
	srv := &HomeTimelineServer{}

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
