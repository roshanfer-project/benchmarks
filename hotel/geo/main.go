package main

import (
	"context"
	"fmt"
	breakwaterinit "hotel/breakwater-init"
	dagorinit "hotel/dagor_init"
	oteltool "hotel/otel_tool"
	rajomoninit "hotel/rajomon_init"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	bw "hotel/breakwater"
	pb "hotel/geo/proto"
	"hotel/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats/opentelemetry"

	"hotel/dagor"

	"github.com/google/uuid"
	"github.com/hailocab/go-geoindex"
	"github.com/pennsail/rajomon"
)

const serviceName = "geo"

var log = utils.GetLogger(serviceName)

//var tracer trace.Tracer

const (
	name             = "srv-geo"
	maxSearchRadius  = 10
	maxSearchResults = 5
)

type Server struct {
	pb.UnimplementedGeoServer

	index *geoindex.ClusteringIndex
	uuid  string
	mutex sync.Mutex // Add mutex for thread-safe access to the index

	MongoClient *mongo.Client
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

	if s.index == nil {
		s.index = newGeoIndex(s.MongoClient)
	}

	s.uuid = uuid.New().String()

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
	if _, shutdownList, ok := configOTL(ctx, "geo"); ok {
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

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			priceTable.UnaryInterceptor,
			AcceptedRPCInterceptor(),
		))
		//opts = append(opts, grpc.UnaryInterceptor(priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor = nil
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			dagorNode.UnaryInterceptorServer,
			AcceptedRPCInterceptor()))
		//opts = append(opts, grpc.UnaryInterceptor(dagorNode.UnaryInterceptorServer))
	}

	var breakwaterd *bw.Breakwater
	if utils.GetEnvVar("breakwaterd", false) == "true" {
		log.Info("breakwaterd is enabled, configuring breakwaterd interceptor")
		breakwaterd = breakwaterinit.GetBreakwater(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			breakwaterd.UnaryInterceptor))
	}

	if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		opts = append(opts, grpc.UnaryInterceptor(CountersInterceptor()))
	}

	if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			AcceptedRPCInterceptor()))
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterGeoServer(srv, s)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("GeoPort", true))))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	return srv.Serve(lis)
}

func (s *Server) Nearby(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	log.Debug("In geo Nearby")

	var (
		points = s.getNearbyPoints(ctx, float64(req.Lat), float64(req.Lon))
		res    = &pb.Result{}
	)

	log.Debug(fmt.Sprintf("geo after getNearbyPoints, len = %d", len(points)))

	for _, p := range points {
		log.Debug(fmt.Sprintf("In geo Nearby return hotelId = %s", p.Id()))
		res.HotelIds = append(res.HotelIds, p.Id())
	}

	return res, nil
}

func (s *Server) getNearbyPoints(_ context.Context, lat, lon float64) []geoindex.Point {
	log.Debug(fmt.Sprintf("In geo getNearbyPoints, lat = %f, lon = %f", lat, lon))

	//return []geoindex.Point{&point{Pid: "1", Plat: 37.7867, Plon: -122.4112}}
	// For the specific request coordinates, return hotel "1" which has exact match
	/* if lat == 37.7867 && lon == -122.4112 {
		return []geoindex.Point{&point{Pid: "1", Plat: 37.7867, Plon: -122.4112}}
	} */

	center := &geoindex.GeoPoint{
		Pid:  "",
		Plat: lat,
		Plon: lon,
	}

	// Use read lock for thread-safe access to the geo index
	s.mutex.Lock()
	points := s.index.KNearest(
		center,
		maxSearchResults,
		geoindex.Km(maxSearchRadius), func(p geoindex.Point) bool {
			return true
		},
	)
	s.mutex.Unlock()

	return points

	/* if len(points) > 0 {
		return points[0]
	}
	return nil */
}

// newGeoIndex returns a geo index with points loaded
func newGeoIndex(client *mongo.Client) *geoindex.ClusteringIndex {
	log.Debug("new geo newGeoIndex")

	collection := client.Database("geo-db").Collection("geo")
	curr, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Error(fmt.Sprintf("Failed get geo data: %v", err))
	}

	var points []*point
	curr.All(context.TODO(), &points)
	if err != nil {
		log.Error(fmt.Sprintf("Failed get geo data: %v", err))
	}

	// add points to index
	index := geoindex.NewClusteringIndex()
	for _, point := range points {
		index.Add(point)
	}

	return index
}

type point struct {
	Pid  string  `bson:"hotelId"`
	Plat float64 `bson:"lat"`
	Plon float64 `bson:"lon"`
}

// Implement Point interface
func (p *point) Lat() float64 { return p.Plat }
func (p *point) Lon() float64 { return p.Plon }
func (p *point) Id() string   { return p.Pid }

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
	log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("GeoMongoAddress", true))
	defer mongoClose()

	srv := &Server{
		MongoClient: mongoClient,
		mutex:       sync.Mutex{},
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func initializeDatabase(url string) (*mongo.Client, func()) {
	log.Info("Generating test data...")

	newPoints := []interface{}{
		point{"1", 37.7867, -122.4112},
		point{"2", 37.7854, -122.4005},
		point{"3", 37.7854, -122.4071},
		point{"4", 37.7936, -122.3930},
		point{"5", 37.7831, -122.4181},
		point{"6", 37.7863, -122.4015},
	}

	for i := 7; i <= 80; i++ {
		hotelID := strconv.Itoa(i)
		lat := 37.7835 + float64(i)/500.0*3
		lon := -122.41 + float64(i)/500.0*4

		newPoints = append(newPoints, point{hotelID, lat, lon})
	}

	uri := fmt.Sprintf("mongodb://%s", url)
	log.Info(fmt.Sprintf("Attempting connection to %v", uri))

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully connected to MongoDB")

	collection := client.Database("geo-db").Collection("geo")
	_, err = collection.InsertMany(context.TODO(), newPoints)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully inserted test data into geo DB")

	return client, func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			log.Error(err.Error())
		}
	}
}
