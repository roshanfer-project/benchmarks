package main

import (
	"context"
	"fmt"
	"net"
	"social"
	breakwaterinit "social/breakwater-init"
	dagorinit "social/dagor_init"
	rajomoninit "social/rajomon_init"
	"social/utils"
	"time"

	oteltool "social/otel_tool"
	pb "social/protobuf"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "client"

var log = utils.GetLogger(serviceName)

type Server struct {
	pb.UnimplementedRajomonClientServer

	nginxClient pb.NginxServiceClient
}

func configOTL(ctx context.Context, serviceName string) (grpc.ServerOption, []func(context.Context) error, bool) {
	if shutdownList, ok := oteltool.InitializeOTel(ctx, serviceName, false); ok {
		//tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")
		//meter = otel.GetMeterProvider().Meter(serviceName + "-meter")
		return opentelemetry.ServerOption(opentelemetry.Options{
			MetricsOptions: opentelemetry.MetricsOptions{MeterProvider: otel.GetMeterProvider()}}), shutdownList, true
	} else {
		return nil, nil, false
	}

}

var failedRPCCounter int64

func FailRPCCounterInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			failedRPCCounter++
		}
		return resp, err
	}
}

func (s *Server) Run() error {

	log.Info("Initializing gRPC server...")

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		//grpc.UnaryInterceptor(tracingInterceptor),
	}

	ctx := context.Background()
	if _, shutdownList, ok := configOTL(ctx, serviceName); ok {
		opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))

		for _, f := range shutdownList {
			defer func() {
				if err := f(ctx); err != nil {
					log.Error("main", "failed to shutdown OpenTelemetry provider", err)
				}
			}()
		}
		log.Info("Successfully initialized OpenTelemetry")
	} else {
		log.Error("Failed to initialize OpenTelemetry")
	}

	opts = append(opts, grpc.UnaryInterceptor(FailRPCCounterInterceptor()))

	srv := grpc.NewServer(opts...)
	pb.RegisterRajomonClientServer(srv, s)

	// enable reflection
	reflection.Register(srv)
	log.Info("gRPC server initialized")

	log.Info("Initializing gRPC clients...")

	nginxEnv := utils.GetEnvVar("NginxGRPCAddr", true)
	var conn *grpc.ClientConn

	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("Rajomon is enabled, initializing Rajomon client...")
		priceTable := rajomoninit.GetPriceTable(serviceName, true)
		conn = social.GetRajomonClient(nginxEnv, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorEnduser))
	} else if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("Dagor is enabled, initializing Dagor client...")
		dagorNode := dagorinit.GetDagorNode(serviceName, false, true)
		conn = social.GetConn(nginxEnv, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	} else if utils.GetEnvVar("breakwater", false) == "true" {
		log.Info("Breakwater is enabled, initializing Breakwater client...")
		breakwater := breakwaterinit.GetBreakwater(serviceName, true)
		conn = social.GetConn(nginxEnv, grpc.WithUnaryInterceptor(breakwater.UnaryInterceptorClient))
	} else {
		panic("Either Rajomon or Dagor or Breakwater must be enabled")
	}
	s.nginxClient = pb.NewNginxServiceClient(conn)

	log.Info("Successful")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("RajomonClientPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func main() {
	failedRPCCounter = 0
	// start a go routine to print failed RPC count every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			log.Info(fmt.Sprintf("Failed RPC count: %d", failedRPCCounter))
		}
	}()
	src := &Server{}
	if err := src.Run(); err != nil {
		log.Error("Failed to run server: " + err.Error())
		panic(err)
	}
}

func (s *Server) ComposePost(ctx context.Context, req *pb.ComposePostRequest) (*pb.ComposePostResponse, error) {
	log.Debug("ComposePost RPC called")
	ctx = metadata.AppendToOutgoingContext(ctx, "method", "compose-post")
	return s.nginxClient.ComposePost(ctx, req)
}

func (s *Server) ReadHomeTimeline(ctx context.Context, req *pb.ReadHomeTimelineRequest) (*pb.ReadHomeTimelineResponse, error) {
	log.Debug("ReadHomeTimeline RPC called")
	ctx = metadata.AppendToOutgoingContext(ctx, "method", "read-home-timeline")
	return s.nginxClient.ReadHomeTimeline(ctx, req)
}

func (s *Server) ReadUserTimeline(ctx context.Context, req *pb.ReadUserTimelineRequest) (*pb.ReadUserTimelineResponse, error) {
	log.Debug("ReadUserTimeline RPC called")
	ctx = metadata.AppendToOutgoingContext(ctx, "method", "read-user-timeline")
	return s.nginxClient.ReadUserTimeline(ctx, req)
}
