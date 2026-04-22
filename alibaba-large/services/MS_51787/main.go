package main

import (
	"context"
	"fmt"
	"net"
	"alibabalarge/pkg"
	pb "alibabalarge/protobuf"
	dagor "alibabalarge/dagor"
	dagorinit "alibabalarge/dagor_init"
	rajomoninit "alibabalarge/rajomon_init"
	"alibabalarge/pkg/rpcpolicy"
	"alibabalarge/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"Z8trRkp4mp",
	})
}

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
	pb.RegisterMS_51787Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("MS_51787_EGRESS", true), sidecar)
	}
	if !sidecar {
		addr := utils.GetEnvVar("MS_25806_ADDR", true)
		if useRajomon {
			conn = pkg.DialClient(addr, sidecar, priceTable.UnaryInterceptorClient)
		} else if useDagor {
			conn = pkg.DialClient(addr, sidecar, dagorNode.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, sidecar)
		}
	}
	s.MS_25806Client = pb.NewMS_25806Client(conn)
	if !sidecar {
		addr := utils.GetEnvVar("MS_44246_ADDR", true)
		if useRajomon {
			conn = pkg.DialClient(addr, sidecar, priceTable.UnaryInterceptorClient)
		} else if useDagor {
			conn = pkg.DialClient(addr, sidecar, dagorNode.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, sidecar)
		}
	}
	s.MS_44246Client = pb.NewMS_44246Client(conn)
	if !sidecar {
		addr := utils.GetEnvVar("MS_56113_ADDR", true)
		if useRajomon {
			conn = pkg.DialClient(addr, sidecar, priceTable.UnaryInterceptorClient)
		} else if useDagor {
			conn = pkg.DialClient(addr, sidecar, dagorNode.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, sidecar)
		}
	}
	s.MS_56113Client = pb.NewMS_56113Client(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("MS_51787_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) RypaFB4PfJ(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(358)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "Z8trRkp4mp":
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
