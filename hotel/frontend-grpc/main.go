package main

import (
	"context"
	"fmt"
	"hotel"
	dagorinit "hotel/dagor_init"
	oteltool "hotel/otel_tool"
	pb "hotel/protobuf"
	rajomoninit "hotel/rajomon_init"
	reservation "hotel/reservation/proto"
	search "hotel/search/proto"
	user "hotel/user/proto"
	"hotel/utils"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"hotel/dagor"

	"github.com/pennsail/rajomon"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "frontend-grpc"

var log = utils.GetLogger(serviceName)

type Server struct {
	pb.UnimplementedFrontendServiceServer

	searchClient      search.SearchClient
	profileClient     pb.ProfileClient
	userClient        user.UserClient
	reservationClient reservation.ReservationClient
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

type CounterState struct {
	failedRPCCounter   atomic.Int64
	acceptedRPCCounter atomic.Int64
	inReq              sync.Map
	outReq             sync.Map
	maxQueue           sync.Map
}

func (s *CounterState) IncrementInReq(method string) {
	count, _ := s.inReq.LoadOrStore(method, int64(0))
	s.inReq.Store(method, count.(int64)+1)
}

func (s *CounterState) IncrementOutReq(method string) {
	count, _ := s.outReq.LoadOrStore(method, int64(0))
	s.outReq.Store(method, count.(int64)+1)
}

func (s *CounterState) IncrementMaxQueue(method string, value int64) {
	count, _ := s.maxQueue.LoadOrStore(method, int64(0))
	if value > count.(int64) {
		s.maxQueue.Store(method, value)
	}
}

func (s *CounterState) GetMaxQueue(method string) int64 {
	count, ok := s.maxQueue.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

func (s *CounterState) GetFailedRPCCounter() int64 {
	return s.failedRPCCounter.Load()
}

func (s *CounterState) IncrementFailedRPCCounter() {
	s.failedRPCCounter.Add(1)
}

func (s *CounterState) GetInReq(method string) int64 {
	count, ok := s.inReq.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

func (s *CounterState) GetOutReq(method string) int64 {
	count, ok := s.outReq.Load(method)
	if !ok {
		return 0
	}
	return count.(int64)
}

var maxQueueGuage metric.Int64Gauge
var failedRPCCounterGauge metric.Int64Counter
var acceptedRPCCounterGauge metric.Int64Counter
var counters = &CounterState{}

func AcceptedRPCInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// get metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Error("metadata not found in context")
			return nil, fmt.Errorf("metadata not found in context")
		}
		method := md.Get("method")
		if len(method) == 0 || len(method) > 1 {
			log.Error("method not found in metadata", "metadata", md)
			return nil, fmt.Errorf("method not found in metadata")
		}

		counters.acceptedRPCCounter.Add(1)
		acceptedRPCCounterGauge.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", method[0]),
		))
		return handler(ctx, req)
	}
}

func CountersInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// get metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Error("metadata not found in context")
			return nil, fmt.Errorf("metadata not found in context")
		}
		method := md.Get("method")
		if len(method) == 0 || len(method) > 1 {
			log.Error("method not found in metadata", "metadata", md)
			return nil, fmt.Errorf("method not found in metadata")
		}

		counters.IncrementInReq(method[0])
		queueSize := counters.GetInReq(method[0]) - counters.GetOutReq(method[0])
		if queueSize > counters.GetMaxQueue(method[0]) {
			counters.IncrementMaxQueue(method[0], queueSize)
			maxQueueGuage.Record(ctx, queueSize, metric.WithAttributes(
				attribute.String("method", method[0]),
			))
		}
		resp, err := handler(ctx, req)
		if err != nil {
			counters.IncrementFailedRPCCounter()
			failedRPCCounterGauge.Add(ctx, 1, metric.WithAttributes(
				attribute.String("method", method[0]),
			))
		}
		counters.IncrementOutReq(method[0])
		return resp, err
	}
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

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
			priceTable.UnaryInterceptor,
			AcceptedRPCInterceptor()))
	}

	var dagorNode *dagor.Dagor
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			ContextPropagationInterceptor(),
			dagorNode.UnaryInterceptorServer))
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
	pb.RegisterFrontendServiceServer(srv, s)

	log.Info("Initializing gRPC clients...")
	var searchEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		searchEnv = "FrontendEgress"
	} else {
		searchEnv = "SearchAddr"
	}
	options := []grpc.DialOption{}
	if priceTable != nil {
		log.Debug("Using rajomon interceptor for search client")
		options = append(options, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorClient))
	} else if dagorNode != nil {
		log.Debug("Using dagor interceptor for search client")
		options = append(options, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	}
	conn := hotel.GetConn(utils.GetEnvVar(searchEnv, true), options...)
	s.searchClient = search.NewSearchClient(conn)

	var profileEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		profileEnv = "FrontendEgress"
	} else {
		profileEnv = "ProfileAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(profileEnv, true), options...)
	s.profileClient = pb.NewProfileClient(conn)

	/* if err := s.initRecommendationClient("srv-recommendation"); err != nil {
		return err
	} */
	var userEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		userEnv = "FrontendEgress"
	} else {
		userEnv = "UserAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(userEnv, true), options...)
	s.userClient = user.NewUserClient(conn)

	var reservationEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		reservationEnv = "FrontendEgress"
	} else {
		reservationEnv = "ReservationAddr"
	}
	conn = hotel.GetConn(utils.GetEnvVar(reservationEnv, true), options...)
	s.reservationClient = reservation.NewReservationClient(conn)

	log.Info("Successful")

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("FrontendGRPCPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func (s *Server) SearchHotels(ctx context.Context, req *pb.SearchHotelsRequest) (*pb.SearchHotelsResponse, error) {
	log.Debug("starts searchHandler")

	if req.InDate == "" || req.OutDate == "" {
		log.Error("inDate or outDate is empty")
		return nil, fmt.Errorf("inDate or outDate is empty")
	}

	log.Debug("starts searchHandler querying downstream")

	/* if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Debug("SearchHotels handler", "metadata", md)
	} else {
		log.Debug("SearchHotels handler", "metadata", "no metadata found")
	} */

	log.Debug(fmt.Sprintf("SEARCH [lat: %v, lon: %v, inDate: %v, outDate: %v", req.Lat, req.Lon, req.InDate, req.OutDate))
	searchResp, err := s.searchClient.Nearby(ctx, &search.NearbyRequest{
		Lat:     req.Lat,
		Lon:     req.Lon,
		InDate:  req.InDate,
		OutDate: req.OutDate,
	})
	if err != nil {
		log.Error(fmt.Sprintf("failed to search nearby hotels: %v", err))
		return nil, fmt.Errorf("failed to search nearby hotels: %w", err)
	}

	log.Debug("SearchHandler gets searchResp")

	locale := "en"

	reservationResp, err := s.reservationClient.CheckAvailability(ctx, &reservation.Request{
		CustomerName: "",
		HotelId:      searchResp.HotelIds,
		InDate:       req.InDate,
		OutDate:      req.OutDate,
		RoomNumber:   1,
	})
	if err != nil {
		log.Error("SearchHandler CheckAvailability failed " + err.Error())
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}
	log.Debug("SearchHandler CheckAvailability successful")

	log.Debug(fmt.Sprintf("searchHandler gets reserveResp.HotelId = %s", reservationResp.HotelId))

	profileResp, err := s.profileClient.GetProfiles(ctx, &pb.Request{
		HotelIds: reservationResp.HotelId,
		Locale:   locale,
	})
	if err != nil {
		log.Error("SearchHandler GetProfiles failed " + err.Error())
		return nil, fmt.Errorf("failed to get hotel profiles: %w", err)
	}
	log.Debug("SearchHandler GetProfiles successful")

	geoJSONResponse(profileResp.Hotels)

	return &pb.SearchHotelsResponse{
		Profiles: profileResp.Hotels}, nil
}

func (s *Server) FrontendReservation(ctx context.Context, req *pb.FrontendReservationRequest) (*pb.FrontendReservationResponse, error) {
	if req.InDate == "" || req.OutDate == "" {
		log.Error("inDate or outDate is empty")
		return nil, fmt.Errorf("inDate or outDate is empty")
	}
	log.Debug("starts FrontendReservationHandler querying downstream")

	// Check username and password
	recResp, err := s.userClient.CheckUser(ctx, &user.Request{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		log.Error("FrontendReservationHandler CheckUser failed " + err.Error())
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	log.Debug("FrontendReservationHandler CheckUser successful")

	resp := &pb.FrontendReservationResponse{
		Success: recResp.Correct,
	}

	// Make reservation
	resResp, err := s.reservationClient.MakeReservation(ctx, &reservation.Request{
		CustomerName: req.CustomerName,
		HotelId:      []string{req.HotelId},
		InDate:       req.InDate,
		OutDate:      req.OutDate,
		RoomNumber:   int32(req.Number),
	})
	if err != nil {
		log.Error("FrontendReservationHandler MakeReservation failed " + err.Error())
		return nil, fmt.Errorf("failed to make reservation: %w", err)
	}
	log.Debug("FrontendReservationHandler MakeReservation successful")

	if len(resResp.HotelId) == 0 {
		resp.Success = false
	}

	return resp, nil
}

// return a geoJSON response that allows google map to plot points directly on map
// https://developers.google.com/maps/documentation/javascript/datalayer#sample_geojson
func geoJSONResponse(hs []*pb.Hotel) map[string]interface{} {
	fs := []interface{}{}

	for _, h := range hs {
		fs = append(fs, map[string]interface{}{
			"type": "Feature",
			"id":   h.Id,
			/* "properties": map[string]string{
				"name":         h.Name,
				"phone_number": h.PhoneNumber,
			},
			"geometry": map[string]interface{}{
				"type": "Point",
				"coordinates": []float32{
					h.Address.Lon,
					h.Address.Lat,
				},
			}, */
		})
	}

	return map[string]interface{}{
		"type":     "FeatureCollection",
		"features": fs,
	}
}

func main() {
	var ok error
	maxQueueGuage, ok = otel.GetMeterProvider().Meter(serviceName).Int64Gauge("max_queue",
		metric.WithDescription("Maximum queue length for each RPC method"))
	if ok != nil {
		log.Error("Failed to create max_queue gauge")
		panic("Failed to create max_queue gauge")
	}
	failedRPCCounterGauge, ok = otel.GetMeterProvider().Meter(serviceName).Int64Counter("failed_rpc",
		metric.WithDescription("Total number of failed RPC calls"))
	if ok != nil {
		log.Error("Failed to create failed_rpc counter")
		panic("Failed to create failed_rpc counter")
	}
	acceptedRPCCounterGauge, ok = otel.GetMeterProvider().Meter(serviceName).Int64Counter("accepted_rpc",
		metric.WithDescription("Total number of accepted RPC calls"))
	if ok != nil {
		log.Error("Failed to create accepted_rpc counter")
		panic("Failed to create accepted_rpc counter")
	}
	src := &Server{}
	if err := src.Run(); err != nil {
		log.Error("Failed to run server: " + err.Error())
		panic(err)
	}
}
