package main

import (
	"context"
	"fmt"
	"net"
	"probroutefanout/pkg"
	pb "probroutefanout/protobuf"
	rajomoninit "probroutefanout/rajomon_init"
	"probroutefanout/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)


type Server struct {
	pb.UnimplementedFrontendServer
	Backend1Client pb.Backend1Client
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

	reflection.Register(srv)
	listenPort := utils.StrToInt(utils.GetEnvVar("EntryGRPCPort", true))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) F1(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(80)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "f1":
		var err error
		_, err = s.Backend1Client.F2(ctx, req)
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
