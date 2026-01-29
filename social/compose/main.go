package main

import (
	"context"
	"fmt"
	"net"
	"social"

	"social/dagor"
	dagorinit "social/dagor_init"
	pb "social/protobuf"
	rajomoninit "social/rajomon_init"
	"social/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ComposePostServer struct {
	pb.UnimplementedComposePostServer

	postClient pb.PostStorageClient
	homeClient pb.HomeTimelineClient
	userClient pb.UserTimelineClient
}

const serviceName = "compose"

var log = utils.GetLogger(serviceName)

func (s *ComposePostServer) Run() error {
	log.Info("Initializing gRPC server...")

	opts := social.GetServerOptions()
	counter := utils.NewCounterState(serviceName)

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
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
	srv := &ComposePostServer{}

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
