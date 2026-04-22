package main

import (
	"context"
	"fmt"
	"net"
	"pfanout4/pkg"
	pb "pfanout4/protobuf"
	dagor "pfanout4/dagor"
	dagorinit "pfanout4/dagor_init"
	rajomoninit "pfanout4/rajomon_init"
	"pfanout4/pkg/rpcpolicy"
	"pfanout4/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"f1",
	})
}

type Server struct {
	pb.UnimplementedBackend3Server
}

const serviceName = "backend3"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if !sidecar && useRajomon && useDagor {
		panic("rajomon and dagor cannot both be enabled")
	}
	var priceTable *rajomon.PriceTable
	var dagorNode *dagor.Dagor
	if useRajomon && !sidecar {
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
	}
	if useDagor && !sidecar {
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
	}
	if sidecar {
		if queuingExport {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				utils.NewCounterState(serviceName).GetInterceptor()))
		} else {
			opts = append(opts, grpc.UnaryInterceptor(utils.ContextPropagationInterceptor()))
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
	pb.RegisterBackend3Server(srv, s)

	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("backend3_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F4(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(288)

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
