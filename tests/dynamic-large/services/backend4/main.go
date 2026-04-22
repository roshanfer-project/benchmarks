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
	"dynamiclarge/pkg/rpcpolicy"
	"dynamiclarge/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var benchRng struct {
	mu sync.Mutex
	r  *rand.Rand
}

func init() {
	seed := time.Now().UnixNano()
	if s := os.Getenv("ROUTING_SEED"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			seed = v
		}
	}
	benchRng.r = rand.New(rand.NewSource(seed))
}

func benchFloat() float64 {
	benchRng.mu.Lock()
	defer benchRng.mu.Unlock()
	return benchRng.r.Float64()
}



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"f1",
	})
}

type Server struct {
	pb.UnimplementedBackend4Server
	Backend5Client pb.Backend5Client
	Backend6Client pb.Backend6Client
}

const serviceName = "backend4"
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
	pb.RegisterBackend4Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("backend4_EGRESS", true), sidecar)
	}
	if !sidecar {
		addr := utils.GetEnvVar("backend5_ADDR", true)
		if useRajomon {
			conn = pkg.DialClient(addr, sidecar, priceTable.UnaryInterceptorClient)
		} else if useDagor {
			conn = pkg.DialClient(addr, sidecar, dagorNode.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, sidecar)
		}
	}
	s.Backend5Client = pb.NewBackend5Client(conn)
	if !sidecar {
		addr := utils.GetEnvVar("backend6_ADDR", true)
		if useRajomon {
			conn = pkg.DialClient(addr, sidecar, priceTable.UnaryInterceptorClient)
		} else if useDagor {
			conn = pkg.DialClient(addr, sidecar, dagorNode.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, sidecar)
		}
	}
	s.Backend6Client = pb.NewBackend6Client(conn)


	reflection.Register(srv)
	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("backend4_PORT", true))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F5(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(128)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		u := benchFloat()
		var err error
		if u < 0.1 {
			_, err = s.Backend5Client.F6(ctx, req)
		} else {
			_, err = s.Backend6Client.F7(ctx, req)
		}
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
