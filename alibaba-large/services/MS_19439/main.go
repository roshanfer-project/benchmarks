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
	pb.UnimplementedMS_19439Server
	MS_12657Client pb.MS_12657Client
	MS_45067Client pb.MS_45067Client
	MS_7103Client pb.MS_7103Client

}

const serviceName = "MS_19439"
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
	pb.RegisterMS_19439Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_19439_EGRESS", true))
	}
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_12657_ADDR", true))
	}
	s.MS_12657Client = pb.NewMS_12657Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_45067_ADDR", true))
	}
	s.MS_45067Client = pb.NewMS_45067Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_7103_ADDR", true))
	}
	s.MS_7103Client = pb.NewMS_7103Client(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("MS_19439_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) KvuxGZYcwm(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(627)
	var err error
	_, err = s.MS_12657Client.KiMcs4YawB(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
	_, err = s.MS_45067Client.WU2EZy8FzO(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}
	_, err = s.MS_7103Client.HnZO60J2RH(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		return nil, err
	}

	return &pb.Response{}, nil
}

func (s *Server) UfXkUzqEz3(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(396)

	return &pb.Response{}, nil
}


func main() {
	s := &Server{}
	log.Info("Starting server...")
	if err := s.Run(); err != nil {
		log.Error("Server failed", "error", err)
	}
}
