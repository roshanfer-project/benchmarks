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
	pb.UnimplementedMS_62039Server
	MS_18750Client pb.MS_18750Client

}

const serviceName = "MS_62039"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	srv := grpc.NewServer(opts...)
	pb.RegisterMS_62039Server(srv, s)
	var conn *grpc.ClientConn
	conn = pkg.GetConn(utils.GetEnvVar("MS_18750_ADDR", true))
	s.MS_18750Client = pb.NewMS_18750Client(conn)

	reflection.Register(srv)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 2000))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) NK4Gw2Phix(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(934)
	var err error
	_, err = s.MS_18750Client.Vs2NQhH1Hq(ctx, req)
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
