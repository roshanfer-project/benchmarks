package main

import (
	"context"
	"fmt"
	"net"
	"fanoutfaninheavy/pkg"
	pb "fanoutfaninheavy/protobuf"
	"fanoutfaninheavy/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	pb.UnimplementedSharedServer

}

const serviceName = "shared"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if sidecar {
		if queuingExport {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				utils.NewCounterState(serviceName).GetInterceptor()))
		} else {
			opts = append(opts, grpc.UnaryInterceptor(utils.ContextPropagationInterceptor()))
		}
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterSharedServer(srv, s)

	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("shared_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F4(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(640)
	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":

	default:
	}
	return &pb.Response{}, nil
}


func main() {
	s := &Server{}
	log.Info("Starting server...")
	if err := s.Run(); err != nil {
		log.Error("Server failed", "error", err)
	}
}
