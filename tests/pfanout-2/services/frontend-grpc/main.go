package main

import (
	"context"
	"fmt"
	"net"
	"pfanout2/pkg"
	pb "pfanout2/protobuf"
	dagor "pfanout2/dagor"
	dagorinit "pfanout2/dagor_init"
	rajomoninit "pfanout2/rajomon_init"
	"pfanout2/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"sync"
)


type Server struct {
	pb.UnimplementedFrontendServer
	Backend1Client pb.Backend1Client
	Backend2Client pb.Backend2Client
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
		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			pt.UnaryInterceptor))
	} else {
		dn = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.InTapHandle(utils.TapHandler(serviceName)))
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dn.UnaryInterceptorServer))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterFrontendServer(srv, s)
	{
		addr := utils.GetEnvVar("backend1_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
		s.Backend1Client = pb.NewBackend1Client(conn)
	}
	{
		addr := utils.GetEnvVar("backend2_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorClient))
		} else {
			conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
		}
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


func (s *Server) Api(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(96)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "api":
		var wg sync.WaitGroup
		var errMu sync.Mutex
		var firstErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Backend1Client.Svc(ctx, req)
			if e != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				errMu.Unlock()
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Backend2Client.Svc(ctx, req)
			if e != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				errMu.Unlock()
			}
		}()
		wg.Wait()
		if firstErr != nil {
			log.Error("downstream call failed", "error", firstErr)
			return nil, firstErr
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
