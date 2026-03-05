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
	pb.UnimplementedMS_21298Server
	MS_25806Client pb.MS_25806Client

}

const serviceName = "MS_21298"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	srv := grpc.NewServer(opts...)
	pb.RegisterMS_21298Server(srv, s)
	var conn *grpc.ClientConn
	conn = pkg.GetConn(utils.GetEnvVar("MS_25806_ADDR", true))
	s.MS_25806Client = pb.NewMS_25806Client(conn)

	reflection.Register(srv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 2000))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) Te9DKpWLH7(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(553)

	return &pb.Response{}, nil
}

func (s *Server) QRB35KFger(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(480)
	var err error
	_, err = s.MS_25806Client.QQqbn5HPP(ctx, req)
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
