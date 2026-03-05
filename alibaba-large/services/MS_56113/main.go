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
	pb.UnimplementedMS_56113Server

}

const serviceName = "MS_56113"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	srv := grpc.NewServer(opts...)
	pb.RegisterMS_56113Server(srv, s)

	reflection.Register(srv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 2000))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) M0PIREyu4Tb(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(560)

	return &pb.Response{}, nil
}

func (s *Server) F0BDDol0SG(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(960)

	return &pb.Response{}, nil
}

func (s *Server) KuU4P3BCru(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(579)

	return &pb.Response{}, nil
}


func main() {
	s := &Server{}
	log.Info("Starting server...")
	if err := s.Run(); err != nil {
		log.Error("Server failed", "error", err)
	}
}
