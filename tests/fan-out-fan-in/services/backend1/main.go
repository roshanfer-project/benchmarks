package main

import (
	"context"
	"fmt"
	"net"
	"fanoutfanin/pkg"
	pb "fanoutfanin/protobuf"
	dagor "fanoutfanin/dagor"
	dagorinit "fanoutfanin/dagor_init"
	rajomoninit "fanoutfanin/rajomon_init"
	"fanoutfanin/pkg/rpcpolicy"
	"fanoutfanin/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"f1",		"g1",
	})
}

type Server struct {
	pb.UnimplementedBackend1Server
	SharedClient pb.SharedClient
}

const serviceName = "backend1"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	utils.StartFailslowAdmin()
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
				utils.NewCounterState(serviceName).GetInterceptor(),
				utils.FailslowUnaryServerInterceptor()))
		} else {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				utils.ContextPropagationInterceptor(),
				utils.FailslowUnaryServerInterceptor()))
		}
	} else if useRajomon {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			priceTable.UnaryInterceptor,
			utils.FailslowUnaryServerInterceptor()))
	} else if useDagor {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dagorNode.UnaryInterceptorServer,
			utils.FailslowUnaryServerInterceptor()))
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			utils.FailslowUnaryServerInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterBackend1Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("backend1_EGRESS", true), sidecar)
	}
	if !sidecar {
		addr := utils.GetEnvVar("shared_ADDR", true)
		if useRajomon {
			conn = pkg.DialClient(addr, sidecar, priceTable.UnaryInterceptorClient)
		} else if useDagor {
			conn = pkg.DialClient(addr, sidecar, dagorNode.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, sidecar)
		}
	}
	s.SharedClient = pb.NewSharedClient(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("backend1_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F2(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(128)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		var err error
		_, err = s.SharedClient.F4(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}


	case "g1":

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
