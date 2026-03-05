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
	pb.UnimplementedMS_51787Server
	MS_25806Client pb.MS_25806Client
	MS_44246Client pb.MS_44246Client
	MS_56113Client pb.MS_56113Client

}

const serviceName = "MS_51787"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	srv := grpc.NewServer(opts...)
	pb.RegisterMS_51787Server(srv, s)
	var conn *grpc.ClientConn
	conn = pkg.GetConn(utils.GetEnvVar("MS_25806_ADDR", true))
	s.MS_25806Client = pb.NewMS_25806Client(conn)
	conn = pkg.GetConn(utils.GetEnvVar("MS_44246_ADDR", true))
	s.MS_44246Client = pb.NewMS_44246Client(conn)
	conn = pkg.GetConn(utils.GetEnvVar("MS_56113_ADDR", true))
	s.MS_56113Client = pb.NewMS_56113Client(conn)

	reflection.Register(srv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 2000))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) RypaFB4PfJ(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(358)
	var err error
	_, err = s.MS_25806Client.M0PIREyu4Tb(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
	_, err = s.MS_44246Client.NRLDYEHBqx(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
	_, err = s.MS_56113Client.M0PIREyu4Tb(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
	_, err = s.MS_56113Client.KuU4P3BCru(ctx, req)
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
