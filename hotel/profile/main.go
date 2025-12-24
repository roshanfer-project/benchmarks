package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hotel"
	breakwaterinit "hotel/breakwater-init"
	dagorinit "hotel/dagor_init"
	oteltool "hotel/otel_tool"
	pb "hotel/protobuf"
	rajomoninit "hotel/rajomon_init"
	"hotel/utils"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	bw "hotel/breakwater"
	"hotel/dagor"

	"github.com/google/uuid"
	"github.com/pennsail/rajomon"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "profile"

var log = utils.GetLogger(serviceName)
var tracer trace.Tracer

type Server struct {
	pb.UnimplementedProfileServer

	uuid string

	// In-memory data stores
	hotels   map[string]*pb.Hotel // key: hotelId
	memcache map[string][]byte    // key: hotelId, value: JSON-encoded Hotel
	mu       sync.RWMutex         // protects all in-memory stores
}

func configOTL(ctx context.Context, serviceName string, frontend bool) (grpc.ServerOption, []func(context.Context) error, bool) {
	if shutdownList, ok := oteltool.InitializeOTel(ctx, serviceName, frontend); ok {
		tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")
		//meter = otel.GetMeterProvider().Meter(serviceName + "-meter")
		return opentelemetry.ServerOption(opentelemetry.Options{
			MetricsOptions: opentelemetry.MetricsOptions{MeterProvider: otel.GetMeterProvider()}}), shutdownList, true
	} else {
		return nil, nil, false
	}

}

type CounterState struct {
	failedRPCCounter   atomic.Int64
	acceptedRPCCounter atomic.Int64
	inReq              sync.Map
	outReq             sync.Map
	maxQueue           sync.Map
	lock               sync.Mutex
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

		counters.lock.Lock()
		counters.acceptedRPCCounter.Add(1)
		acceptedRPCCounterGauge.Add(ctx, 1, metric.WithAttributes(
			attribute.String("api", method[0]),
		))
		counters.lock.Unlock()
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

		counters.lock.Lock()
		counters.IncrementInReq(method[0])
		queueSize := counters.GetInReq(method[0]) - counters.GetOutReq(method[0])
		if queueSize > counters.GetMaxQueue(method[0]) {
			counters.IncrementMaxQueue(method[0], queueSize)
			maxQueueGuage.Record(ctx, queueSize, metric.WithAttributes(
				attribute.String("api", method[0]),
			))
		}
		counters.lock.Unlock()
		resp, err := handler(ctx, req)
		counters.lock.Lock()
		if err != nil {
			counters.IncrementFailedRPCCounter()
			failedRPCCounterGauge.Add(ctx, 1, metric.WithAttributes(
				attribute.String("api", method[0]),
			))
		}
		counters.IncrementOutReq(method[0])
		counters.lock.Unlock()
		return resp, err
	}
}

func (s *Server) Run() error {

	s.uuid = uuid.New().String()

	log.Info(fmt.Sprintf("in run s.IpAddr = %s, port = %d", "localhost", utils.StrToInt(utils.GetEnvVar("ProfilePort", true))))

	opts := hotel.DefaultServerOptions()

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		//opts = append(opts, grpc.UnaryInterceptor(priceTable.UnaryInterceptor))
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			priceTable.UnaryInterceptor,
			AcceptedRPCInterceptor()))
	}

	var dagorNode *dagor.Dagor = nil
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
		//opts = append(opts, grpc.UnaryInterceptor(dagorNode.UnaryInterceptorServer))
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			dagorNode.UnaryInterceptorServer,
			AcceptedRPCInterceptor()))
	}

	var breakwaterd *bw.Breakwater
	if utils.GetEnvVar("breakwaterd", false) == "true" {
		log.Info("breakwaterd is enabled, configuring breakwaterd interceptor")
		breakwaterd = breakwaterinit.GetBreakwater(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			breakwaterd.UnaryInterceptor))
	}

	/* if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		opts = append(opts, grpc.UnaryInterceptor(CountersInterceptor()))
	}

	if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			AcceptedRPCInterceptor()))
	} */

	/* ctx := context.Background()
	if _, shutdownList, ok := configOTL(ctx, serviceName, false); ok {
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

	pb.RegisterProfileServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("ProfilePort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to configure listener: %v", err))
	}

	return srv.Serve(lis)
}

func (s *Server) GetProfiles(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	log.Debug("In GetProfiles")

	var wg sync.WaitGroup
	var mutex sync.Mutex

	// one hotel should only have one profile
	hotelIds := make([]string, 0)
	profileMap := make(map[string]struct{})
	for _, hotelId := range req.HotelIds {
		hotelIds = append(hotelIds, hotelId)
		profileMap[hotelId] = struct{}{}
	}

	//memSpan, _ := opentracing.StartSpanFromContext(ctx, "memcached_get_profile")
	_, memSpan := tracer.Start(ctx, "memcached_get_profile")
	// memSpan.SetAttributes(attribute.String("span.kind", "client"))
	memSpan.SetAttributes(attribute.String("span.kind", "client"))

	// Get from memcache
	s.mu.RLock()
	resMap := make(map[string][]byte)
	for _, hotelId := range hotelIds {
		if val, ok := s.memcache[hotelId]; ok {
			resMap[hotelId] = val
		}
	}
	s.mu.RUnlock()

	//memSpan.Finish()
	memSpan.End()

	res := new(pb.Result)
	hotels := make([]*pb.Hotel, 0)

	for hotelId, item := range resMap {
		profileStr := string(item)
		log.Debug(fmt.Sprintf("memc hit with %v", profileStr))

		hotelProf := new(pb.Hotel)
		json.Unmarshal(item, hotelProf)
		hotels = append(hotels, hotelProf)
		delete(profileMap, hotelId)
	}

	wg.Add(len(profileMap))
	for hotelId := range profileMap {
		go func(hotelId string) {
			_, mongoSpan := tracer.Start(ctx, "mongo_profile")
			mongoSpan.SetAttributes(attribute.String("span.kind", "client"))

			s.mu.RLock()
			hotelProf := s.hotels[hotelId]
			s.mu.RUnlock()

			mongoSpan.End()

			if hotelProf == nil {
				log.Error(fmt.Sprintf("Failed get hotels data: hotelId %v not found", hotelId))
			}

			mutex.Lock()
			hotels = append(hotels, hotelProf)
			mutex.Unlock()

			if hotelProf != nil {
				profJson, err := json.Marshal(hotelProf)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to marshal hotel [id: %v] with err: %v", hotelProf.Id, err))
				} else {
					// write to memcached
					s.mu.Lock()
					s.memcache[hotelId] = profJson
					s.mu.Unlock()
				}
			}
			defer wg.Done()
		}(hotelId)
	}
	wg.Wait()

	res.Hotels = hotels
	log.Debug("In GetProfiles after getting resp")
	return res, nil
}

func main() {
	log.Info("Reading config...")
	tracer = otel.GetTracerProvider().Tracer(serviceName)
	/* var ok error
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
	} */
	log.Info("Initializing in-memory data stores...")
	hotels := initializeDatabase()

	srv := &Server{
		hotels:   hotels,
		memcache: make(map[string][]byte),
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

type Hotel struct {
	Id string `bson:"id"`
}

type Address struct {
	StreetNumber string  `bson:"streetNumber"`
	StreetName   string  `bson:"streetName"`
	City         string  `bson:"city"`
	State        string  `bson:"state"`
	Country      string  `bson:"country"`
	PostalCode   string  `bson:"postalCode"`
	Lat          float32 `bson:"lat"`
	Lon          float32 `bson:"lon"`
}

func initializeDatabase() map[string]*pb.Hotel {
	log.Info("Generating test data...")

	hotels := make(map[string]*pb.Hotel)

	hotelIds := []string{"1", "2", "3", "4", "5", "6"}
	for i := 7; i <= 50; i++ {
		hotelIds = append(hotelIds, strconv.Itoa(i))
	}

	for _, hotelID := range hotelIds {
		hotels[hotelID] = &pb.Hotel{
			Id: hotelID,
		}
	}

	log.Info("Successfully initialized in-memory profile data stores")
	return hotels
}
