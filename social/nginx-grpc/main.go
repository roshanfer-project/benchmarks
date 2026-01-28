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

type NginxGRPCServer struct {
	pb.UnimplementedNginxServiceServer

	composeClient pb.ComposePostClient
	homeClient    pb.HomeTimelineClient
	userClient    pb.UserTimelineClient
}

const serviceName = "nginx-grpc"

var log = utils.GetLogger(serviceName)

func (s *NginxGRPCServer) Run() error {
	log.Info("Initializing gRPC server...")

	opts := social.GetServerOptions()
	counter := utils.NewCounterState(serviceName)

	var priceTable *rajomon.PriceTable = nil
	var dagorNode *dagor.Dagor
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			priceTable.UnaryInterceptor))
	} else if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	} else {
		panic("One of rajomon or dagor must be enabled")
	}

	srv := grpc.NewServer(opts...)
	pb.RegisterNginxServiceServer(srv, s)

	log.Info("Initializing gRPC clients...")
	options := []grpc.DialOption{}
	if priceTable != nil {
		log.Debug("Using rajomon interceptor for compose client")
		options = append(options, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
	} else if dagorNode != nil {
		log.Debug("Using dagor interceptor for compose client")
		options = append(options, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}
	conn := social.GetConn(utils.GetEnvVar("ComposeAddr", true), options...)
	s.composeClient = pb.NewComposePostClient(conn)

	conn = social.GetConn(utils.GetEnvVar("HomeAddr", true), options...)
	s.homeClient = pb.NewHomeTimelineClient(conn)

	conn = social.GetConn(utils.GetEnvVar("UserAddr", true), options...)
	s.userClient = pb.NewUserTimelineClient(conn)

	log.Info("Successful")

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("NginxGRPCPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func main() {
	srv := &NginxGRPCServer{}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func (s *NginxGRPCServer) ComposePost(ctx context.Context, req *pb.ComposePostRequest) (*pb.ComposePostResponse, error) {
	// Generate the CreatorId based on connection ID
	username := "user1"
	req.CreatorId = username

	//ctx = config.PropagateMetadata(ctx, "nginx")
	resp, err := s.composeClient.ComposePost(ctx, req)
	if err != nil {
		log.Error("[ComposePost] Error forwarding compose post request.", "error", err)
		return nil, err
	}
	return resp, nil
}

func (s *NginxGRPCServer) ReadUserTimeline(ctx context.Context, req *pb.ReadUserTimelineRequest) (*pb.ReadUserTimelineResponse, error) {
	// Generate the UserId based on connection ID
	username := "user1"
	req.UserId = username

	//ctx = config.PropagateMetadata(ctx, "nginx")
	//resp, err := invoke.Invoke[*pb.ReadUserTimelineResponse](ctx, "usertimeline", "ReadUserTimeline", req)
	resp, err := s.userClient.ReadUserTimeline(ctx, req)
	if err != nil {
		log.Error("[ReadUserTimeline] Error forwarding read user timeline request.", "error", err)
		return nil, err
	}
	return resp, nil
}

func (s *NginxGRPCServer) ReadHomeTimeline(ctx context.Context, req *pb.ReadHomeTimelineRequest) (*pb.ReadHomeTimelineResponse, error) {
	// Generate the UserId based on connection ID
	username := "user1"
	req.UserId = username

	//ctx = config.PropagateMetadata(ctx, "nginx")
	//resp, err := invoke.Invoke[*pb.ReadHomeTimelineResponse](ctx, "hometimeline", "ReadHomeTimeline", req)
	resp, err := s.homeClient.ReadHomeTimeline(ctx, req)
	if err != nil {
		log.Error("[ReadHomeTimeline] Error forwarding read home timeline request.", "error", err)
		return nil, err
	}
	return resp, nil
}
