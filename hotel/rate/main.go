package main

import (
	"context"
	"encoding/json"
	"fmt"
	breakwaterinit "hotel/breakwater-init"
	dagorinit "hotel/dagor_init"
	oteltool "hotel/otel_tool"
	rajomoninit "hotel/rajomon_init"
	pb "hotel/rate/proto"
	"hotel/utils"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bw "hotel/breakwater"
	"hotel/dagor"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"github.com/pennsail/rajomon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "rate"

var log = utils.GetLogger(serviceName)
var tracer trace.Tracer

type Server struct {
	pb.UnimplementedRateServer

	uuid string

	MongoClient *mongo.Client
	MemcClient  *memcache.Client
}

func configOTL(ctx context.Context, serviceName string) (grpc.ServerOption, []func(context.Context) error, bool) {
	if shutdownList, ok := oteltool.InitializeOTel(ctx, serviceName, false); ok {
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
			priceTable.UnaryInterceptor,
			AcceptedRPCInterceptor()))
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

	pb.RegisterRateServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("RatePort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func (s *Server) GetRates(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	res := new(pb.Result)

	ratePlans := make(RatePlans, 0)

	hotelIds := []string{}
	rateMap := make(map[string]struct{})
	for _, hotelID := range req.HotelIds {
		hotelIds = append(hotelIds, hotelID)
		rateMap[hotelID] = struct{}{}
	}
	// first check memcached(get-multi)
	//memSpan, _ := opentracing.StartSpanFromContext(ctx, "memcached_get_multi_rate")
	_, memSpan := tracer.Start(ctx, "memcached_get_multi_rate")
	//memSpan.SetTag("span.kind", "client")
	memSpan.SetAttributes(attribute.String("span.kind", "client"))

	resMap, err := s.MemcClient.GetMulti(hotelIds)
	//memSpan.Finish()
	memSpan.End()

	var wg sync.WaitGroup
	var mutex sync.Mutex
	if err != nil && err != memcache.ErrCacheMiss {
		log.Error(fmt.Sprintf("Memmcached error while trying to get hotel [id: %v]= %s", hotelIds, err.Error()))
	} else {
		for hotelId, item := range resMap {
			rateStrs := strings.Split(string(item.Value), "\n")
			log.Debug("memc hit, hotelId = %s,rate strings: %v", hotelId, rateStrs)

			for _, rateStr := range rateStrs {
				if len(rateStr) != 0 {
					rateP := new(pb.RatePlan)
					json.Unmarshal([]byte(rateStr), rateP)
					ratePlans = append(ratePlans, rateP)
				}
			}

			delete(rateMap, hotelId)
		}

		wg.Add(len(rateMap))
		for hotelId := range rateMap {
			go func(id string) {
				log.Debug(fmt.Sprintf("memc miss, hotelId = %s", id))
				log.Debug("memcached miss, set up mongo connection")

				//mongoSpan, _ := opentracing.StartSpanFromContext(ctx, "mongo_rate")
				_, mongoSpan := tracer.Start(ctx, "mongo_rate")
				//mongoSpan.SetTag("span.kind", "client")
				mongoSpan.SetAttributes(attribute.String("span.kind", "client"))

				// memcached miss, set up mongo connection
				collection := s.MongoClient.Database("rate-db").Collection("inventory")
				curr, err := collection.Find(context.TODO(), bson.D{})
				if err != nil {
					log.Error(fmt.Sprintf("Failed get rate data: %s", err.Error()))
				}

				tmpRatePlans := make(RatePlans, 0)
				curr.All(context.TODO(), &tmpRatePlans)
				if err != nil {
					log.Error(fmt.Sprintf("Failed get rate data: %s", err.Error()))
				}

				//mongoSpan.Finish()
				mongoSpan.End()

				memcStr := ""
				if err != nil {
					log.Error("Tried to find hotelId [%v], but got error", id, err.Error())
				} else {
					for _, r := range tmpRatePlans {
						mutex.Lock()
						ratePlans = append(ratePlans, r)
						mutex.Unlock()
						rateJson, err := json.Marshal(r)
						if err != nil {
							log.Error("Failed to marshal plan with error", "error", err)
						}
						memcStr = memcStr + string(rateJson) + "\n"
					}
				}
				go s.MemcClient.Set(&memcache.Item{Key: id, Value: []byte(memcStr)})

				defer wg.Done()
			}(hotelId)
		}
	}
	wg.Wait()

	sort.Sort(ratePlans)
	res.RatePlans = ratePlans

	return res, nil
}

type RatePlans []*pb.RatePlan

func (r RatePlans) Len() int {
	return len(r)
}

func (r RatePlans) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RatePlans) Less(i, j int) bool {
	return r[i].RoomType.TotalRate > r[j].RoomType.TotalRate
}

func main() {
	log.Info("Reading config...")
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
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("RateMongoAddress", true))
	defer mongoClose()

	log.Info(fmt.Sprintf("Read profile memcashed address: %v", utils.GetEnvVar("RateMemcAddress", true)))
	log.Info("Initializing Memcashed client...")
	memcClient := utils.NewMemCClient2(utils.GetEnvVar("RateMemcAddress", true))
	log.Info("Success")

	srv := &Server{
		MongoClient: mongoClient,
		MemcClient:  memcClient,
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

type RoomType struct {
	BookableRate       float64 `bson:"bookableRate"`
	Code               string  `bson:"code"`
	RoomDescription    string  `bson:"roomDescription"`
	TotalRate          float64 `bson:"totalRate"`
	TotalRateInclusive float64 `bson:"totalRateInclusive"`
}

type RatePlan struct {
	HotelId  string    `bson:"hotelId"`
	Code     string    `bson:"code"`
	InDate   string    `bson:"inDate"`
	OutDate  string    `bson:"outDate"`
	RoomType *RoomType `bson:"roomType"`
}

func initializeDatabase(url string) (*mongo.Client, func()) {
	log.Info("Generating test data...")

	newRatePlans := []interface{}{
		RatePlan{
			"1",
			"RACK",
			"2015-04-09",
			"2015-04-10",
			&RoomType{
				109.00,
				"KNG",
				"King sized bed",
				109.00,
				123.17,
			},
		},
		RatePlan{
			"2",
			"RACK",
			"2015-04-09",
			"2015-04-10",
			&RoomType{
				139.00,
				"QN",
				"Queen sized bed",
				139.00,
				153.09,
			},
		},
		RatePlan{
			"3",
			"RACK",
			"2015-04-09",
			"2015-04-10",
			&RoomType{
				109.00,
				"KNG",
				"King sized bed",
				109.00,
				123.17,
			},
		},
	}

	for i := 7; i <= 50; i++ {
		if i%3 != 0 {
			continue
		}

		hotelID := strconv.Itoa(i)

		endDate := "2015-04-"
		if i%2 == 0 {
			endDate = fmt.Sprintf("%s17", endDate)
		} else {
			endDate = fmt.Sprintf("%s24", endDate)
		}

		rate := 109.00
		rateInc := 123.17
		if i%5 == 1 {
			rate = 120.00
			rateInc = 140.00
		} else if i%5 == 2 {
			rate = 124.00
			rateInc = 144.00
		} else if i%5 == 3 {
			rate = 132.00
			rateInc = 158.00
		} else if i%5 == 4 {
			rate = 232.00
			rateInc = 258.00
		}

		newRatePlans = append(
			newRatePlans,
			RatePlan{
				hotelID,
				"RACK",
				"2015-04-09",
				endDate,
				&RoomType{
					rate,
					"KNG",
					"King sized bed",
					rate,
					rateInc,
				},
			},
		)
	}

	uri := fmt.Sprintf("mongodb://%s", url)
	log.Info(fmt.Sprintf("Attempting connection to %v", uri))

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully connected to MongoDB")

	collection := client.Database("rate-db").Collection("inventory")
	_, err = collection.InsertMany(context.TODO(), newRatePlans)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully inserted test data into rate DB")

	return client, func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			log.Error(err.Error())
		}
	}
}
