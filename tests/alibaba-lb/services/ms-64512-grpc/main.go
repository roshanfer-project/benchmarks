package main

import (
	"context"
	"fmt"
	"net"
	"alibabalb/pkg"
	pb "alibabalb/protobuf"
	dagor "alibabalb/dagor"
	dagorinit "alibabalb/dagor_init"
	rajomoninit "alibabalb/rajomon_init"
	"alibabalb/utils"

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

func benchExpBusyLoop(mean float64) {
	benchRng.mu.Lock()
	rt := benchRng.r.ExpFloat64() * mean
	benchRng.mu.Unlock()
	repeats := int(rt * 320)
	if repeats < 1 {
		repeats = 1
	}
	utils.BusyLoop(repeats)
}


type Server struct {
	pb.UnimplementedMS_64512Server
	MS_2687Client pb.MS_2687Client
	MS_40087Client pb.MS_40087Client
	MS_51787Client pb.MS_51787Client
	MS_70124Client pb.MS_70124Client
}

const serviceName = "ms-64512-grpc"
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
	pb.RegisterMS_64512Server(srv, s)
	{
		addr := utils.GetEnvVar("MS_2687_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
		s.MS_2687Client = pb.NewMS_2687Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_40087_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
		s.MS_40087Client = pb.NewMS_40087Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_51787_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
		s.MS_51787Client = pb.NewMS_51787Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_70124_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
		s.MS_70124Client = pb.NewMS_70124Client(conn)
	}

	reflection.Register(srv)
	listenPort := utils.StrToInt(utils.GetEnvVar("EntryGRPCPort", true))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) Z8TrRkp4Mp(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	benchExpBusyLoop(0.5)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "Z8trRkp4mp":
		var err error
		_, err = s.MS_2687Client.VdboDuPbKj(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_40087Client.M2QxmWDHq1O(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_40087Client.M5ISZV1SCx(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_51787Client.RypaFB4PfJ(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_70124Client.V0Gqd6H7Nw(ctx, req)
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
