package main

import (
	"context"
	"fmt"
	oteltool "hotel/otel_tool"
	pb "hotel/reservation/proto"
	"hotel/utils"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats/opentelemetry"
)

var log = utils.GetLogger("reservation")
var tracer trace.Tracer

type Server struct {
	pb.UnimplementedReservationServer

	uuid string

	MongoClient *mongo.Client
	MemcClient  *memcache.Client
}

func tracingInterceptor(ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

	_, span := tracer.Start(ctx, info.FullMethod)
	defer span.End()
	return handler(ctx, req)

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

	ctx := context.Background()
	if _, shutdownList, ok := configOTL(ctx, "reservation"); ok {
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

	database := s.MongoClient.Database("reservation-db")
	resCollection := database.Collection("reservation")
	numCollection := database.Collection("number")

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
		item, err := s.MemcClient.Get(memc_key)
		if err == nil {
			// memcached hit
			count, _ = strconv.Atoi(string(item.Value))
			log.Debug("memcached hit %s = %d", memc_key, count)
			memc_date_num_map[memc_key] = count + int(req.RoomNumber)

		} else if err == memcache.ErrCacheMiss {
			// memcached miss
			log.Debug("memcached miss")
			var reserve []reservation

			filter := bson.D{{Key: "hotelId", Value: hotelId}, {Key: "inDate", Value: indate}, {Key: "outDate", Value: outdate}}
			curr, err := resCollection.Find(context.TODO(), filter)
			if err != nil {
				log.Debug(fmt.Sprintf("Failed get reservation data: %v", err))
			}
			curr.All(context.TODO(), &reserve)
			if err != nil {
				log.Error(fmt.Sprintf("Tried to find hotelId [%v] from date [%v] to date [%v], but got error [%v]", hotelId, indate, outdate, err.Error()))
			}

			for _, r := range reserve {
				count += r.Number
			}

			memc_date_num_map[memc_key] = count + int(req.RoomNumber)

		} else {
			log.Error(fmt.Sprintf("Tried to get memc_key [%v], but got memmcached error = %s", memc_key, err))
		}

		// check capacity
		// check memc capacity
		memc_cap_key := hotelId + "_cap"
		item, err = s.MemcClient.Get(memc_cap_key)
		hotel_cap := 0
		if err == nil {
			// memcached hit
			hotel_cap, _ = strconv.Atoi(string(item.Value))
			log.Debug(fmt.Sprintf("memcached hit %s = %d", memc_cap_key, hotel_cap))
		} else if err == memcache.ErrCacheMiss {
			// memcached miss
			var num number
			err = numCollection.FindOne(context.TODO(), &bson.D{{Key: "hotelId", Value: hotelId}}).Decode(&num)
			if err != nil {
				log.Debug(fmt.Sprintf("Tried to find hotelId [%v], but got error %v", hotelId, err.Error()))
			}
			hotel_cap = int(num.Number)

			// write to memcache
			s.MemcClient.Set(&memcache.Item{Key: memc_cap_key, Value: []byte(strconv.Itoa(hotel_cap))})
		} else {
			log.Error(fmt.Sprintf("Tried to get memc_cap_key [%v], but got memmcached error = %s", memc_cap_key, err))
		}

		if count+int(req.RoomNumber) > hotel_cap {
			return res, nil
		}
		indate = outdate
	}

	// only update reservation number cache after check succeeds
	for key, val := range memc_date_num_map {
		s.MemcClient.Set(&memcache.Item{Key: key, Value: []byte(strconv.Itoa(val))})
	}

	inDate, _ = time.Parse(
		time.RFC3339,
		req.InDate+"T12:00:00+00:00")

	indate = inDate.String()[0:10]

	for inDate.Before(outDate) {
		inDate = inDate.AddDate(0, 0, 1)
		outdate := inDate.String()[0:10]
		_, err := resCollection.InsertOne(
			context.TODO(),
			reservation{
				HotelId:      hotelId,
				CustomerName: req.CustomerName,
				InDate:       indate,
				OutDate:      outdate,
				Number:       int(req.RoomNumber),
			},
		)
		if err != nil {
			log.Error(fmt.Sprintf("Tried to insert hotel [hotelId %v], but got error: %s", hotelId, err.Error()))
		}
		indate = outdate
	}

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
	cacheMemRes, err := s.MemcClient.GetMulti(hotelMemKeys)
	//capMemSpan.Finish()
	capMemSpan.End()

	numCollection := s.MongoClient.Database("reservation-db").Collection("number")

	misKeys := []string{}
	// gather cache miss key to query in mongodb
	if err == memcache.ErrCacheMiss {
		for key := range keysMap {
			if _, ok := cacheMemRes[key]; !ok {
				misKeys = append(misKeys, key)
			}
		}
	} else if err != nil {
		log.Error(fmt.Sprintf("Tried to get memc_cap_key [%v], but got memmcached error = %s", hotelMemKeys, err))
	}
	// store whole capacity result in cacheCap
	cacheCap := make(map[string]int)
	for k, v := range cacheMemRes {
		hotelCap, _ := strconv.Atoi(string(v.Value))
		cacheCap[k] = hotelCap
	}
	if len(misKeys) > 0 {
		queryMissKeys := []string{}
		for _, k := range misKeys {
			queryMissKeys = append(queryMissKeys, strings.Split(k, "_")[0])
		}
		var nums []number
		//capMongoSpan, _ := opentracing.StartSpanFromContext(ctx, "mongodb_capacity_get_multi_number")
		_, capMongoSpan := tracer.Start(ctx, "mongodb_capacity_get_multi_number")
		//capMongoSpan.SetTag("span.kind", "client")
		capMongoSpan.SetAttributes(attribute.String("span.kind", "client"))
		curr, err := numCollection.Find(context.TODO(), bson.D{{Key: "$in", Value: queryMissKeys}})
		if err != nil {
			log.Error(fmt.Sprintf("Failed get reservation number data: %v", err))
		}
		curr.All(context.TODO(), &nums)
		if err != nil {
			log.Error(fmt.Sprintf("Failed get reservation number data: %v", err))
		}
		//capMongoSpan.Finish()
		capMongoSpan.End()
		if err != nil {
			log.Error(fmt.Sprintf("Tried to find hotelId [%v], but got error: %v", misKeys, err.Error()))
		}
		for _, num := range nums {
			cacheCap[num.HotelId] = num.Number
			// we don't care set successfully or not
			go s.MemcClient.Set(&memcache.Item{Key: num.HotelId + "_cap", Value: []byte(strconv.Itoa(num.Number))})
		}
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
			outDate := inDate.String()[:10]
			memcKey := hotelId + "_" + outDate + "_" + outDate
			reqCommand = append(reqCommand, memcKey)
			queryMap[memcKey] = map[string]string{
				"hotelId":   hotelId,
				"startDate": indate,
				"endDate":   outDate,
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
	// check capacity in memcached and mongodb
	if itemsMap, err := s.MemcClient.GetMulti(reqCommand); err != nil && err != memcache.ErrCacheMiss {
		//reserveMemSpan.Finish()
		log.Error(fmt.Sprintf("Tried to get memc_key [%v], but got memmcached error = %s", reqCommand, err))
	} else {
		//reserveMemSpan.Finish()
		reserveMemSpan.End()
		// go through reservation count from memcached
		go func() {
			for k, v := range itemsMap {
				id := strings.Split(k, "_")[0]
				val, _ := strconv.Atoi(string(v.Value))
				var res bool
				if val+int(req.RoomNumber) <= cacheCap[id] {
					res = true
				}
				ch <- taskRes{
					hotelId:  id,
					checkRes: res,
				}
			}
			if err == nil {
				close(ch)
			}
		}()
		// use miss reservation to get data from mongo
		// rever string to indata and outdate
		if err == memcache.ErrCacheMiss {
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

					var reserve []reservation

					queryItem := queryMap[comm]
					resCollection := s.MongoClient.Database("reservation-db").Collection("reservation")
					filter := bson.D{{Key: "hotelId", Value: queryItem["hotelId"]}, {Key: "inDate", Value: queryItem["startDate"]}, {Key: "outDate", Value: queryItem["endDate"]}}

					//reserveMongoSpan, _ := opentracing.StartSpanFromContext(ctx, "mongodb_capacity_get_multi_number"+comm)
					_, reserveMongoSpan := tracer.Start(ctx, "mongodb_capacity_get_multi_number"+comm)
					//reserveMongoSpan.SetTag("span.kind", "client")
					reserveMongoSpan.SetAttributes(attribute.String("span.kind", "client"))
					curr, err := resCollection.Find(context.TODO(), filter)
					if err != nil {
						log.Error(fmt.Sprintf("Failed get reservation data: %v", err))
					}
					curr.All(context.TODO(), &reserve)
					if err != nil {
						log.Error(fmt.Sprintf("Failed get reservation data: %v", err))
					}
					//reserveMongoSpan.Finish()
					reserveMongoSpan.End()

					if err != nil {
						log.Error("Tried to find hotelId reservation error",
							"hotelId", queryItem["hotelId"],
							"startDate", queryItem["startDate"],
							"endDate", queryItem["endDate"],
							"error", err.Error())
					}
					var count int
					for _, r := range reserve {
						log.Error(fmt.Sprintf("reservation check reservation number = %s", queryItem["hotelId"]))
						count += r.Number
					}
					// update memcached
					go s.MemcClient.Set(&memcache.Item{Key: comm, Value: []byte(strconv.Itoa(count))})
					var res bool
					if count+int(req.RoomNumber) <= cacheCap[queryItem["hotelId"]] {
						res = true
					}
					ch <- taskRes{
						hotelId:  queryItem["hotelId"],
						checkRes: res,
					}
				}(command)
			}
		}
	}

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

	log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("ReserveMongoAddress", true))
	defer mongoClose()

	log.Info(fmt.Sprintf("Read profile memcashed address: %v", utils.GetEnvVar("ReserveMemcAddress", true)))
	log.Info("Initializing Memcashed client...")
	memcClient := utils.NewMemCClient2(utils.GetEnvVar("ReserveMemcAddress", true))
	log.Info("Success")

	srv := &Server{
		MongoClient: mongoClient,
		MemcClient:  memcClient,
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

func initializeDatabase(url string) (*mongo.Client, func()) {
	log.Info("Generating test data...")

	newReservations := []interface{}{
		Reservation{"4", "Alice", "2015-04-09", "2015-04-10", 1},
	}

	newNumbers := []interface{}{
		Number{"1", 200},
		Number{"2", 200},
		Number{"3", 200},
		Number{"4", 200},
		Number{"5", 200},
		Number{"6", 200},
	}

	for i := 7; i <= 80; i++ {
		hotelID := strconv.Itoa(i)

		roomNumber := 200
		if i%3 == 1 {
			roomNumber = 300
		} else if i%3 == 2 {
			roomNumber = 250
		}

		newNumbers = append(newNumbers, Number{hotelID, roomNumber})
	}

	uri := fmt.Sprintf("mongodb://%s", url)
	log.Info(fmt.Sprintf("Attempting connection to %v", uri))

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully connected to MongoDB")

	database := client.Database("reservation-db")
	resCollection := database.Collection("reservation")
	numCollection := database.Collection("number")

	_, err = resCollection.InsertMany(context.TODO(), newReservations)
	if err != nil {
		log.Error(err.Error())
	}

	_, err = numCollection.InsertMany(context.TODO(), newNumbers)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully inserted test data into reservation DB")

	return client, func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			log.Error(err.Error())
		}
	}
}
