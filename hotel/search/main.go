package main

import (
	"context"
	"fmt"
	"hotel"
	dagorinit "hotel/dagor_init"
	geo "hotel/geo/proto"
	oteltool "hotel/otel_tool"
	rajomoninit "hotel/rajomon_init"
	rate "hotel/rate/proto"
	pb "hotel/search/proto"
	"hotel/utils"
	"net"

	"hotel/dagor"

	"github.com/google/uuid"
	"github.com/pennsail/rajomon"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "search"

var log = utils.GetLogger(serviceName)

//var tracer trace.Tracer

type Server struct {
	pb.UnimplementedSearchServer

	geoClient  geo.GeoClient
	rateClient rate.RateClient
	uuid       string
}

func ContextPropagationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		if in, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = metadata.NewOutgoingContext(ctx, in)
		}
		return handler(ctx, req)
	}
}

/* func tracingInterceptor(ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

	_, span := tracer.Start(ctx, info.FullMethod)
	defer span.End()
	return handler(ctx, req)

} */

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
	s.uuid = uuid.New().String()

	opts := hotel.DefaultServerOptions()
	counter := utils.NewCounterState(serviceName)

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor = nil
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	}

	/* if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
		))
	} else if utils.GetEnvVar("sidecar", false) == "true" {
		opts = append(opts, grpc.UnaryInterceptor(ContextPropagationInterceptor()))
	} */

	/* if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
			AcceptedRPCInterceptor()))
	} */

	if utils.GetEnvVar("sidecar", false) == "true" {
		if utils.GetEnvVar("queuing_export", false) == "true" {
			opts = append(opts, grpc.ChainUnaryInterceptor(
				ContextPropagationInterceptor(),
				counter.GetInterceptor()))
		} else {
			opts = append(opts, grpc.UnaryInterceptor(ContextPropagationInterceptor()))
		}
	} else if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			ContextPropagationInterceptor(),
			counter.GetInterceptor()))
	}

	/* ctx := context.Background()
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
	} */

	srv := grpc.NewServer(opts...)
	pb.RegisterSearchServer(srv, s)

	// init grpc clients
	/* if err := s.initGeoClient("srv-geo"); err != nil {
		return err
	}
	if err := s.initRateClient("srv-rate"); err != nil {
		return err
	} */

	var geoEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		geoEnv = "SearchEgress"
	} else {
		geoEnv = "GeoAddr"
	}
	options := []grpc.DialOption{}
	if priceTable != nil {
		options = append(options, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
	} else if dagorNode != nil {
		options = append(options, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}

	conn := hotel.GetConn(utils.GetEnvVar(geoEnv, true), options...)
	s.geoClient = geo.NewGeoClient(conn)

	var rateEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		rateEnv = "SearchEgress"
	} else {
		rateEnv = "RateAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(rateEnv, true), options...)
	s.rateClient = rate.NewRateClient(conn)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("SearchPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

// Nearby returns ids of nearby hotels ordered by ranking algo
func (s *Server) Nearby(ctx context.Context, req *pb.NearbyRequest) (*pb.SearchResult, error) {
	// find nearby hotels
	log.Debug("in Search Nearby")

	log.Debug(fmt.Sprintf("nearby lat = %f", req.Lat))
	log.Debug(fmt.Sprintf("nearby lon = %f", req.Lon))

	nearby, err := s.geoClient.Nearby(ctx, &geo.Request{
		Lat: req.Lat,
		Lon: req.Lon,
	})
	if err != nil {
		return nil, err
	}

	for _, hid := range nearby.HotelIds {
		log.Debug(fmt.Sprintf("get Nearby hotelId = %s", hid))
	}

	// find rates for hotels
	rates, err := s.rateClient.GetRates(ctx, &rate.Request{
		HotelIds: nearby.HotelIds,
		InDate:   req.InDate,
		OutDate:  req.OutDate,
	})
	if err != nil {
		return nil, err
	}

	// TODO(hw): add simple ranking algo to order hotel ids:
	// * geo distance
	// * price (best discount?)
	// * reviews

	// build the response
	res := new(pb.SearchResult)
	for _, ratePlan := range rates.RatePlans {
		log.Debug(fmt.Sprintf("get RatePlan HotelId = %s", ratePlan.HotelId))
		res.HotelIds = append(res.HotelIds, ratePlan.HotelId)
	}
	return res, nil
}

func main() {
	srv := &Server{}
	if err := srv.Run(); err != nil {
		log.Error(fmt.Sprintf("failed to run: %v", err))
	}
}
