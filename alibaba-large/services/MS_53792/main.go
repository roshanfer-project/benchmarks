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
	pb.UnimplementedMS_53792Server
	MS_41667Client pb.MS_41667Client
	MS_5720Client pb.MS_5720Client

}

const serviceName = "MS_53792"
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
	pb.RegisterMS_53792Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_53792_EGRESS", true))
	}
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_41667_ADDR", true))
	}
	s.MS_41667Client = pb.NewMS_41667Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_5720_ADDR", true))
	}
	s.MS_5720Client = pb.NewMS_5720Client(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("MS_53792_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) M8JkkxghEWB(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(544)
	var err error
	_, err = s.MS_41667Client.Bk01RUGHnE(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
	_, err = s.MS_5720Client.M4Cgy9S6B5O(ctx, req)
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
