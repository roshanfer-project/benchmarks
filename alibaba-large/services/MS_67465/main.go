package main

import (
	"context"
	"fmt"
	"net"
	"alibabalarge/pkg"
	pb "alibabalarge/protobuf"
	"alibabalarge/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	pb.UnimplementedMS_67465Server
	MS_19439Client pb.MS_19439Client

}

const serviceName = "MS_67465"
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
	pb.RegisterMS_67465Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_67465_EGRESS", true))
	}
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_19439_ADDR", true))
	}
	s.MS_19439Client = pb.NewMS_19439Client(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("MS_67465_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) WIe9Cm5AqE(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(592)
	var err error
	_, err = s.MS_19439Client.UfXkUzqEz3(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
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
