package main

import (
	"context"
	"fmt"
	"net"
	"dynamiclarge/pkg"
	pb "dynamiclarge/protobuf"
	rajomoninit "dynamiclarge/rajomon_init"
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


type Server struct {
	pb.UnimplementedBackend1Server
	Backend2Client pb.Backend2Client
	Backend3Client pb.Backend3Client
}

const serviceName = "backend1"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	opts := pkg.GetServerOptions()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	var priceTable *rajomon.PriceTable
	if useRajomon && !sidecar {
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
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
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterBackend1Server(srv, s)
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("backend1_EGRESS", true))
	}
	if !sidecar {
		addr := utils.GetEnvVar("backend2_ADDR", true)
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr)
		}
	}
	s.Backend2Client = pb.NewBackend2Client(conn)
	if !sidecar {
		addr := utils.GetEnvVar("backend3_ADDR", true)
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr)
		}
	}
	s.Backend3Client = pb.NewBackend3Client(conn)


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
	utils.BusyLoop(192)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		u := benchFloat()
		var err error
		if u < 0.6 {
			_, err = s.Backend2Client.F3(ctx, req)
		} else {
			_, err = s.Backend3Client.F4(ctx, req)
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
