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
	pb.UnimplementedMS_70124Server
	MS_56113Client pb.MS_56113Client

}

const serviceName = "MS_70124"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	srv := grpc.NewServer(opts...)
	pb.RegisterMS_70124Server(srv, s)
	var conn *grpc.ClientConn
	conn = pkg.GetConn(utils.GetEnvVar("MS_56113_ADDR", true))
	s.MS_56113Client = pb.NewMS_56113Client(conn)

	reflection.Register(srv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 2000))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) V0Gqd6H7Nw(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(1164)
	var err error
	_, err = s.MS_56113Client.M0PIREyu4Tb(ctx, req)
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
