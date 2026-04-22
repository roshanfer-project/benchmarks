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
	"sync"
)



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"f1",
	})
}

type Server struct {
	pb.UnimplementedFrontendServer
	Backend1Client pb.Backend1Client
	Backend2Client pb.Backend2Client
	Backend3Client pb.Backend3Client
	Backend4Client pb.Backend4Client
}

const serviceName = "frontend-grpc"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	utils.StartFailslowAdmin()
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	if useRajomon == useDagor {
		panic("entry-grpc requires exactly one of rajomon=true or dagor=true")
	}
	opts := pkg.GetServerOptions()
	var pt *rajomon.PriceTable
	var dn *dagor.Dagor
	if useRajomon {
		pt = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			pt.UnaryInterceptor,
			utils.FailslowUnaryServerInterceptor()))
	} else {
		dn = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dn.UnaryInterceptorServer,
			utils.FailslowUnaryServerInterceptor()))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterFrontendServer(srv, s)
	{
		addr := utils.GetEnvVar("backend1_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.Backend1Client = pb.NewBackend1Client(conn)
	}
	{
		addr := utils.GetEnvVar("backend2_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.Backend2Client = pb.NewBackend2Client(conn)
	}
	{
		addr := utils.GetEnvVar("backend3_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.Backend3Client = pb.NewBackend3Client(conn)
	}
	{
		addr := utils.GetEnvVar("backend4_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.Backend4Client = pb.NewBackend4Client(conn)
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
	utils.BusyLoop(64)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		var wg sync.WaitGroup
		var errMu sync.Mutex
		var firstErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Backend1Client.F2(ctx, req)
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
			_, e := s.Backend2Client.F3(ctx, req)
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
			_, e := s.Backend3Client.F4(ctx, req)
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
			_, e := s.Backend4Client.F5(ctx, req)
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
