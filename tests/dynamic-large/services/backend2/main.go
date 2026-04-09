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
	pb.UnimplementedBackend2Server
	Backend4Client pb.Backend4Client
}

const serviceName = "backend2"
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
	pb.RegisterBackend2Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("backend2_EGRESS", true))
	}
	if !sidecar {
		addr := utils.GetEnvVar("backend4_ADDR", true)
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
		} else if useDagor {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr)
		}
	}
	s.Backend4Client = pb.NewBackend4Client(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("backend2_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F3(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(128)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		var err error
		_, err = s.Backend4Client.F5(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}


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
