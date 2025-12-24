package main

import (
	"context"
	"fmt"
	"hotel"
	breakwaterinit "hotel/breakwater-init"
	dagorinit "hotel/dagor_init"
	oteltool "hotel/otel_tool"
	rajomoninit "hotel/rajomon_init"
	pb "hotel/reservation/proto"
	"hotel/utils"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

const serviceName = "reservation"

var log = utils.GetLogger(serviceName)
var tracer trace.Tracer

type Server struct {
	pb.UnimplementedReservationServer

	uuid string

	// In-memory data stores
	reservations map[string][]reservation // key: hotelId_inDate_outDate
	numbers      map[string]int           // key: hotelId, value: capacity
	memcache     map[string][]byte        // key: memcache key, value: cached value
	mu           sync.RWMutex             // protects all in-memory stores
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

// var queueHistogram metric.Int64Histogram
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

	opts := hotel.DefaultServerOptions()

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

	/* if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		opts = append(opts, grpc.UnaryInterceptor(CountersInterceptor()))
	}

	if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			CountersInterceptor(),
			AcceptedRPCInterceptor()))
	} */

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

	pb.RegisterReservationServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("ReservationPort", true))))
	if err != nil {
		log.Info(fmt.Sprintf("failed to listen: %v", err))
	}

	log.Info("In reservation s.IpAddr = %s, port = %d", "localhost", utils.GetEnvVar("ReservationPort", true))

	return srv.Serve(lis)
}

func (s *Server) MakeReservation(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	res := new(pb.Result)
	res.HotelId = make([]string, 0)

	inDate, _ := time.Parse(
		time.RFC3339,
		req.InDate+"T12:00:00+00:00")

	outDate, _ := time.Parse(
		time.RFC3339,
		req.OutDate+"T12:00:00+00:00")
	hotelId := req.HotelId[0]

	indate := inDate.String()[0:10]

	memc_date_num_map := make(map[string]int)

	for inDate.Before(outDate) {
		// check reservations
		count := 0
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]

		// first check memc
		memc_key := hotelId + "_" + inDate.String()[0:10] + "_" + outdate
		s.mu.RLock()
		cachedVal, memcHit := s.memcache[memc_key]
		s.mu.RUnlock()

		if memcHit {
			// memcached hit
			count, _ = strconv.Atoi(string(cachedVal))
			log.Debug("memcached hit %s = %d", memc_key, count)
			memc_date_num_map[memc_key] = count + int(req.RoomNumber)
		} else {
			// memcached miss - check reservations
			log.Debug("memcached miss")
			resKey := hotelId + "_" + indate + "_" + outdate
			s.mu.RLock()
			reserve := s.reservations[resKey]
			s.mu.RUnlock()

			for _, r := range reserve {
				count += r.Number
			}

			memc_date_num_map[memc_key] = count + int(req.RoomNumber)
		}

		// check capacity
		// check memc capacity
		memc_cap_key := hotelId + "_cap"
		s.mu.RLock()
		cachedCapVal, capMemcHit := s.memcache[memc_cap_key]
		s.mu.RUnlock()

		hotel_cap := 0
		if capMemcHit {
			// memcached hit
			hotel_cap, _ = strconv.Atoi(string(cachedCapVal))
			log.Debug(fmt.Sprintf("memcached hit %s = %d", memc_cap_key, hotel_cap))
		} else {
			// memcached miss - check numbers
			s.mu.RLock()
			hotel_cap = s.numbers[hotelId]
			s.mu.RUnlock()

			// write to memcache
			s.mu.Lock()
			s.memcache[memc_cap_key] = []byte(strconv.Itoa(hotel_cap))
			s.mu.Unlock()
		}

		if count+int(req.RoomNumber) > hotel_cap {
			return res, nil
		}
		indate = outdate
	}

	// only update reservation number cache after check succeeds
	s.mu.Lock()
	for key, val := range memc_date_num_map {
		s.memcache[key] = []byte(strconv.Itoa(val))
	}
	s.mu.Unlock()

	inDate, _ = time.Parse(
		time.RFC3339,
		req.InDate+"T12:00:00+00:00")

	indate = inDate.String()[0:10]

	// Insert reservations
	s.mu.Lock()
	for inDate.Before(outDate) {
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]
		resKey := hotelId + "_" + indate + "_" + outdate
		s.reservations[resKey] = append(s.reservations[resKey], reservation{
			HotelId:      hotelId,
			CustomerName: req.CustomerName,
			InDate:       indate,
			OutDate:      outdate,
			Number:       int(req.RoomNumber),
		})
		indate = outdate
	}
	s.mu.Unlock()

	res.HotelId = append(res.HotelId, hotelId)

	return res, nil
}

// CheckAvailability checks if given information is available
func (s *Server) CheckAvailability(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	res := new(pb.Result)
	res.HotelId = make([]string, 0)

	hotelMemKeys := []string{}
	keysMap := make(map[string]struct{})
	resMap := make(map[string]bool)
	// cache capacity since it will not change
	for _, hotelId := range req.HotelId {
		hotelMemKeys = append(hotelMemKeys, hotelId+"_cap")
		resMap[hotelId] = true
		keysMap[hotelId+"_cap"] = struct{}{}
	}

	//capMemSpan, _ := opentracing.StartSpanFromContext(ctx, "memcached_capacity_get_multi_number")
	_, capMemSpan := tracer.Start(ctx, "memcached_capacity_get_multi_number")
	//capMemSpan.SetTag("span.kind", "client")
	capMemSpan.SetAttributes(attribute.String("span.kind", "client"))

	// Get capacities from memcache
	s.mu.RLock()
	cacheMemRes := make(map[string][]byte)
	for _, key := range hotelMemKeys {
		if val, ok := s.memcache[key]; ok {
			cacheMemRes[key] = val
		}
	}
	s.mu.RUnlock()

	capMemSpan.End()

	misKeys := []string{}
	// gather cache miss key to query
	for key := range keysMap {
		if _, ok := cacheMemRes[key]; !ok {
			misKeys = append(misKeys, key)
		}
	}

	// store whole capacity result in cacheCap
	cacheCap := make(map[string]int)
	for k, v := range cacheMemRes {
		hotelCap, _ := strconv.Atoi(string(v))
		hotelId := strings.Split(k, "_")[0]
		cacheCap[hotelId] = hotelCap
	}

	// Fill in missing capacities from numbers store
	if len(misKeys) > 0 {
		_, capMongoSpan := tracer.Start(ctx, "mongodb_capacity_get_multi_number")
		capMongoSpan.SetAttributes(attribute.String("span.kind", "client"))

		s.mu.Lock()
		for _, k := range misKeys {
			hotelId := strings.Split(k, "_")[0]
			if cap, ok := s.numbers[hotelId]; ok {
				cacheCap[hotelId] = cap
				// update memcache
				s.memcache[hotelId+"_cap"] = []byte(strconv.Itoa(cap))
			}
		}
		s.mu.Unlock()

		capMongoSpan.End()
	}

	reqCommand := []string{}
	queryMap := make(map[string]map[string]string)
	for _, hotelId := range req.HotelId {
		log.Debug(fmt.Sprintf("reservation check hotel %s", hotelId))
		inDate, _ := time.Parse(
			time.RFC3339,
			req.InDate+"T12:00:00+00:00")
		outDate, _ := time.Parse(
			time.RFC3339,
			req.OutDate+"T12:00:00+00:00")
		for inDate.Before(outDate) {
			indate := inDate.String()[:10]
			inDate = inDate.AddDate(0, 0, 1)
			outdate := inDate.String()[:10]
			memcKey := hotelId + "_" + indate + "_" + outdate
			reqCommand = append(reqCommand, memcKey)
			queryMap[memcKey] = map[string]string{
				"hotelId":   hotelId,
				"startDate": indate,
				"endDate":   outdate,
			}
		}
	}

	type taskRes struct {
		hotelId  string
		checkRes bool
	}
	//reserveMemSpan, _ := opentracing.StartSpanFromContext(ctx, "memcached_reserve_get_multi_number")
	_, reserveMemSpan := tracer.Start(ctx, "memcached_reserve_get_multi_number")
	ch := make(chan taskRes)
	//reserveMemSpan.SetTag("span.kind", "client")
	reserveMemSpan.SetAttributes(attribute.String("span.kind", "client"))

	// Get reservation counts from memcache
	s.mu.RLock()
	itemsMap := make(map[string][]byte)
	for _, key := range reqCommand {
		if val, ok := s.memcache[key]; ok {
			itemsMap[key] = val
		}
	}
	s.mu.RUnlock()

	reserveMemSpan.End()

	// go through reservation count from memcached
	go func() {
		for k, v := range itemsMap {
			id := strings.Split(k, "_")[0]
			val, _ := strconv.Atoi(string(v))
			var res bool
			if cap, ok := cacheCap[id]; ok && val+int(req.RoomNumber) <= cap {
				res = true
			}
			ch <- taskRes{
				hotelId:  id,
				checkRes: res,
			}
		}
		// Process cache misses
		var wg sync.WaitGroup
		for k := range itemsMap {
			delete(queryMap, k)
		}
		wg.Add(len(queryMap))
		go func() {
			wg.Wait()
			close(ch)
		}()
		for command := range queryMap {
			go func(comm string) {
				defer wg.Done()

				queryItem := queryMap[comm]
				resKey := queryItem["hotelId"] + "_" + queryItem["startDate"] + "_" + queryItem["endDate"]

				_, reserveMongoSpan := tracer.Start(ctx, "mongodb_capacity_get_multi_number"+comm)
				reserveMongoSpan.SetAttributes(attribute.String("span.kind", "client"))

				s.mu.RLock()
				reserve := s.reservations[resKey]
				s.mu.RUnlock()

				reserveMongoSpan.End()

				var count int
				for _, r := range reserve {
					log.Error(fmt.Sprintf("reservation check reservation number = %s", queryItem["hotelId"]))
					count += r.Number
				}

				// update memcached
				s.mu.Lock()
				s.memcache[comm] = []byte(strconv.Itoa(count))
				s.mu.Unlock()

				var res bool
				if cap, ok := cacheCap[queryItem["hotelId"]]; ok && count+int(req.RoomNumber) <= cap {
					res = true
				}
				ch <- taskRes{
					hotelId:  queryItem["hotelId"],
					checkRes: res,
				}
			}(command)
		}
	}()

	for task := range ch {
		if !task.checkRes {
			resMap[task.hotelId] = false
		}
	}
	for k, v := range resMap {
		if v {
			res.HotelId = append(res.HotelId, k)
		}
	}

	return res, nil
}

type reservation struct {
	HotelId      string `bson:"hotelId"`
	CustomerName string `bson:"customerName"`
	InDate       string `bson:"inDate"`
	OutDate      string `bson:"outDate"`
	Number       int    `bson:"number"`
}

type number struct {
	HotelId string `bson:"hotelId"`
	Number  int    `bson:"numberOfRoom"`
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
	/* queueHistogram, ok = otel.GetMeterProvider().Meter(serviceName).Int64Histogram("queue_length",
		metric.WithDescription("Queue length for each RPC method"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 25, 30, 40, 50, 60, 80, 100, 150, 200, 300, 400, 500))
	if ok != nil {
		log.Error("Failed to create queue_length histogram")
		panic("Failed to create queue_length histogram")
	} */
	log.Info("Initializing in-memory data stores...")
	reservations, numbers := initializeDatabase()

	srv := &Server{
		reservations: reservations,
		numbers:      numbers,
		memcache:     make(map[string][]byte),
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

type Reservation struct {
	HotelId      string `bson:"hotelId"`
	CustomerName string `bson:"customerName"`
	InDate       string `bson:"inDate"`
	OutDate      string `bson:"outDate"`
	Number       int    `bson:"number"`
}

type Number struct {
	HotelId string `bson:"hotelId"`
	Number  int    `bson:"numberOfRoom"`
}

func initializeDatabase() (map[string][]reservation, map[string]int) {
	log.Info("Generating test data...")

	reservations := make(map[string][]reservation)
	numbers := make(map[string]int)

	// Initialize reservations
	resKey := "4_2015-04-09_2015-04-10"
	reservations[resKey] = []reservation{
		{"4", "Alice", "2015-04-09", "2015-04-10", 1},
	}

	// Initialize numbers
	numbers["1"] = 200
	numbers["2"] = 200
	numbers["3"] = 200
	numbers["4"] = 200
	numbers["5"] = 200
	numbers["6"] = 200

	for i := 7; i <= 80; i++ {
		hotelID := strconv.Itoa(i)

		roomNumber := 200
		if i%3 == 1 {
			roomNumber = 300
		} else if i%3 == 2 {
			roomNumber = 250
		}

		numbers[hotelID] = roomNumber
	}

	log.Info("Successfully initialized in-memory reservation data stores")
	return reservations, numbers
}
