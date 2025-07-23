package main

import (
	"context"
	"fmt"
	"hotel"
	dagorinit "hotel/dagor_init"
	rajomoninit "hotel/rajomon_init"
	"hotel/utils"
	"net"
	"time"

	oteltool "hotel/otel_tool"
	pb "hotel/protobuf"

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

	frontendClient pb.FrontendServiceClient
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

	srv := grpc.NewServer(opts...)
	pb.RegisterRajomonClientServer(srv, s)

	// enable reflection
	reflection.Register(srv)
	log.Info("gRPC server initialized")

	log.Info("Initializing gRPC clients...")

	frontendEnv := utils.GetEnvVar("FrontendGRPCAddr", true)
	var conn *grpc.ClientConn

	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("Rajomon is enabled, initializing Rajomon client...")
		priceTable := rajomoninit.GetPriceTable(serviceName, true)
		conn = hotel.GetRajomonClient(frontendEnv, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorEnduser))
	} else if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("Dagor is enabled, initializing Dagor client...")
		dagorNode := dagorinit.GetDagorNode(serviceName, false, true)
		conn = hotel.GetConn(frontendEnv, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	} else {
		panic("Either Rajomon or Dagor must be enabled")
	}
	s.frontendClient = pb.NewFrontendServiceClient(conn)

	log.Info("Successful")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("RajomonClientPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func main() {
	src := &Server{}
	if err := src.Run(); err != nil {
		log.Error("Failed to run server: " + err.Error())
		panic(err)
	}
}

func (s *Server) FrontendReservation(ctx context.Context, req *pb.FrontendReservationRequest) (*pb.FrontendReservationResponse, error) {
	log.Debug("FrontendReservation RPC called")
	ctx = metadata.AppendToOutgoingContext(ctx, "method", "reserve-hotel")
	return s.frontendClient.FrontendReservation(ctx, req)
}

func (s *Server) SearchHotels(ctx context.Context, req *pb.SearchHotelsRequest) (*pb.SearchHotelsResponse, error) {
	log.Debug("SearchHotels RPC called")
	ctx = metadata.AppendToOutgoingContext(ctx, "method", "search-hotel")
	return s.frontendClient.SearchHotels(ctx, req)
}
