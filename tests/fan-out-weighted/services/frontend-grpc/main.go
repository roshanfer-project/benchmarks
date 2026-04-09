package main

import (
	"context"
	"fmt"
	"net"
	"fanoutweighted/pkg"
	pb "fanoutweighted/protobuf"
	rajomoninit "fanoutweighted/rajomon_init"
	"fanoutweighted/utils"

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
	pb.UnimplementedFrontendServer
	Backend1Client pb.Backend1Client
	Backend2Client pb.Backend2Client
}

const serviceName = "frontend-grpc"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	if utils.GetEnvVar("rajomon", false) != "true" {
		panic("entry-grpc requires rajomon=true")
	}
	opts := pkg.GetServerOptions()
	pt := rajomoninit.GetPriceTable(serviceName, false)
	opts = append(opts, grpc.ChainUnaryInterceptor(
		utils.ContextPropagationInterceptor(),
		utils.NewCounterState(serviceName).GetInterceptor(),
		pt.UnaryInterceptor))
	srv := grpc.NewServer(opts...)
	pb.RegisterFrontendServer(srv, s)
	{
		addr := utils.GetEnvVar("backend1_ADDR", true)
		conn := pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		s.Backend1Client = pb.NewBackend1Client(conn)
	}
	{
		addr := utils.GetEnvVar("backend2_ADDR", true)
		conn := pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		s.Backend2Client = pb.NewBackend2Client(conn)
	}

	reflection.Register(srv)
	listenPort := utils.StrToInt(utils.GetEnvVar("EntryGRPCPort", true))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F1(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(96)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		u := benchFloat()
		var err error
		if u < 0.7 {
			_, err = s.Backend1Client.F2(ctx, req)
		} else {
			_, err = s.Backend2Client.F3(ctx, req)
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
