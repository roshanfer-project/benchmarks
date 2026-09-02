package main

import (
	"context"
	"fmt"
	"net"
	"leaf12p21/pkg"
	pb "leaf12p21/protobuf"
	dagor "leaf12p21/dagor"
	dagorinit "leaf12p21/dagor_init"
	rajomoninit "leaf12p21/rajomon_init"
	"leaf12p21/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)


type Server struct {
	pb.UnimplementedFrontendServer
}

const serviceName = "frontend-grpc"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	if useRajomon == useDagor {
		panic("entry-grpc requires exactly one of rajomon=true or dagor=true")
	}
	opts := pkg.GetServerOptions()
	var pt *rajomon.PriceTable
	var dn *dagor.Dagor
	if useRajomon {
		pt = rajomoninit.GetPriceTable(rajomoninit.InstanceName(serviceName), false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			pt.UnaryInterceptor))
	} else {
		dn = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dn.UnaryInterceptorServer))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterFrontendServer(srv, s)

	reflection.Register(srv)
	listenPort := utils.StrToInt(utils.GetEnvVar("EntryGRPCPort", true))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) Api1(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(3200)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "api1":

	default:
	}
	return &pb.Response{}, nil
}

func (s *Server) Api2(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(6400)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "api2":

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
