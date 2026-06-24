package main

import (
	"context"
	"fmt"
	"net"
	"dynamiclarge/pkg"
	pb "dynamiclarge/protobuf"
	dagor "dynamiclarge/dagor"
	dagorinit "dynamiclarge/dagor_init"
	rajomoninit "dynamiclarge/rajomon_init"
	"dynamiclarge/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)


type Server struct {
	pb.UnimplementedBackend7Server
}

const serviceName = "backend7"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	envoy := utils.GetEnvVar("envoy", false) == "true"
	if sidecar && envoy {
		panic("sidecar and envoy cannot both be enabled")
	}
	meshProxy := sidecar || envoy
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if !meshProxy && useRajomon && useDagor {
		panic("rajomon and dagor cannot both be enabled")
	}
	var priceTable *rajomon.PriceTable
	var dagorNode *dagor.Dagor
	if useRajomon && !meshProxy {
		priceTable = rajomoninit.GetPriceTable(rajomoninit.InstanceName(serviceName), false)
	}
	if useDagor && !meshProxy {
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
	}
	if meshProxy {
		if sidecar && queuingExport {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				utils.NewCounterState(serviceName).GetInterceptor()))
		} else {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor()))
		}
	} else if useRajomon {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			priceTable.UnaryInterceptor))
	} else if useDagor {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterBackend7Server(srv, s)

	reflection.Register(srv)
	port := 2000
	if meshProxy {
		port = utils.StrToInt(utils.GetEnvVar("backend7_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F8(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(320)

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
