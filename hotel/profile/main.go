package main

import (
	"context"
	"encoding/json"
	"fmt"
	oteltool "hotel/otel_tool"
	pb "hotel/profile/proto"
	"hotel/utils"
	"net"
	"strconv"
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

var log = utils.GetLogger("profile")
var tracer trace.Tracer

type Server struct {
	pb.UnimplementedProfileServer

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

func (s *Server) Run() error {

	s.uuid = uuid.New().String()

	log.Info(fmt.Sprintf("in run s.IpAddr = %s, port = %d", "localhost", utils.StrToInt(utils.GetEnvVar("ProfilePort", true))))

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
	if _, shutdownList, ok := configOTL(ctx, "profile", false); ok {
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
	resMap, err := s.MemcClient.GetMulti(hotelIds)
	//memSpan.Finish()
	memSpan.End()

	res := new(pb.Result)
	hotels := make([]*pb.Hotel, 0)

	if err != nil && err != memcache.ErrCacheMiss {
		log.Error(fmt.Sprintf("Tried to get hotelIds [%v], but got memmcached error = %s", hotelIds, err))
	} else {
		for hotelId, item := range resMap {
			profileStr := string(item.Value)
			log.Debug(fmt.Sprintf("memc hit with %v", profileStr))

			hotelProf := new(pb.Hotel)
			json.Unmarshal(item.Value, hotelProf)
			hotels = append(hotels, hotelProf)
			delete(profileMap, hotelId)
		}

		wg.Add(len(profileMap))
		for hotelId := range profileMap {
			go func(hotelId string) {
				var hotelProf *pb.Hotel

				collection := s.MongoClient.Database("profile-db").Collection("hotels")

				//mongoSpan, _ := opentracing.StartSpanFromContext(ctx, "mongo_profile")
				_, mongoSpan := tracer.Start(ctx, "mongo_profile")
				//mongoSpan.SetTag("span.kind", "client")
				mongoSpan.SetAttributes(attribute.String("span.kind", "client"))
				err := collection.FindOne(context.TODO(), bson.D{{Key: "id", Value: hotelId}}).Decode(&hotelProf)
				//mongoSpan.Finish()
				mongoSpan.End()

				if err != nil {
					log.Error(fmt.Sprintf("Failed get hotels data: %v", err))
				}

				mutex.Lock()
				hotels = append(hotels, hotelProf)
				mutex.Unlock()

				profJson, err := json.Marshal(hotelProf)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to marshal hotel [id: %v] with err: %v", hotelProf.Id, err))
				}
				memcStr := string(profJson)

				// write to memcached
				go s.MemcClient.Set(&memcache.Item{Key: hotelId, Value: []byte(memcStr)})
				defer wg.Done()
			}(hotelId)
		}
	}
	wg.Wait()

	res.Hotels = hotels
	log.Debug("In GetProfiles after getting resp")
	return res, nil
}

func main() {
	log.Info("Reading config...")

	log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("ProfileMongoAddress", true))
	defer mongoClose()

	log.Info(fmt.Sprintf("Read profile memcashed address: %v", utils.GetEnvVar("ProfileMemcAddress", true)))
	log.Info("Initializing Memcashed client...")
	memcClient := utils.NewMemCClient2(utils.GetEnvVar("ProfileMemcAddress", true))
	log.Info("Success")

	srv := &Server{
		MongoClient: mongoClient,
		MemcClient:  memcClient,
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

type Hotel struct {
	Id          string   `bson:"id"`
	Name        string   `bson:"name"`
	PhoneNumber string   `bson:"phoneNumber"`
	Description string   `bson:"description"`
	Address     *Address `bson:"address"`
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

func initializeDatabase(url string) (*mongo.Client, func()) {
	log.Info("Generating test data...")

	newProfiles := []interface{}{
		Hotel{
			"1",
			"Clift Hotel",
			"(415) 775-4700",
			"A 6-minute walk from Union Square and 4 minutes from a Muni Metro station, this luxury hotel designed by Philippe Starck features an artsy furniture collection in the lobby, including work by Salvador Dali.",
			&Address{
				"495",
				"Geary St",
				"San Francisco",
				"CA",
				"United States",
				"94102",
				37.7867,
				-122.4112,
			},
		},
		Hotel{
			"2",
			"W San Francisco",
			"(415) 777-5300",
			"Less than a block from the Yerba Buena Center for the Arts, this trendy hotel is a 12-minute walk from Union Square.",
			&Address{
				"181",
				"3rd St",
				"San Francisco",
				"CA",
				"United States",
				"94103",
				37.7854,
				-122.4005,
			},
		},
		Hotel{
			"3",
			"Hotel Zetta",
			"(415) 543-8555",
			"A 3-minute walk from the Powell Street cable-car turnaround and BART rail station, this hip hotel 9 minutes from Union Square combines high-tech lodging with artsy touches.",
			&Address{
				"55",
				"5th St",
				"San Francisco",
				"CA",
				"United States",
				"94103",
				37.7834,
				-122.4071,
			},
		},
		Hotel{
			"4",
			"Hotel Vitale",
			"(415) 278-3700",
			"This waterfront hotel with Bay Bridge views is 3 blocks from the Financial District and a 4-minute walk from the Ferry Building.",
			&Address{
				"8",
				"Mission St",
				"San Francisco",
				"CA",
				"United States",
				"94105",
				37.7936,
				-122.3930,
			},
		},
		Hotel{
			"5",
			"Phoenix Hotel",
			"(415) 776-1380",
			"Located in the Tenderloin neighborhood, a 10-minute walk from a BART rail station, this retro motor lodge has hosted many rock musicians and other celebrities since the 1950s. It’s a 4-minute walk from the historic Great American Music Hall nightclub.",
			&Address{
				"601",
				"Eddy St",
				"San Francisco",
				"CA",
				"United States",
				"94109",
				37.7831,
				-122.4181,
			},
		},
		Hotel{
			"6",
			"St. Regis San Francisco",
			"(415) 284-4000",
			"St. Regis Museum Tower is a 42-story, 484 ft skyscraper in the South of Market district of San Francisco, California, adjacent to Yerba Buena Gardens, Moscone Center, PacBell Building and the San Francisco Museum of Modern Art.",
			&Address{
				"125",
				"3rd St",
				"San Francisco",
				"CA",
				"United States",
				"94109",
				37.7863,
				-122.4015,
			},
		},
	}

	for i := 7; i <= 80; i++ {
		hotelID := strconv.Itoa(i)
		phoneNumber := fmt.Sprintf("(415) 284-40%s", hotelID)

		lat := 37.7835 + float32(i)/500.0*3
		lon := -122.41 + float32(i)/500.0*4

		newProfiles = append(
			newProfiles,
			Hotel{
				hotelID,
				"St. Regis San Francisco",
				phoneNumber,
				"St. Regis Museum Tower is a 42-story, 484 ft skyscraper in the South of Market district of San Francisco, California, adjacent to Yerba Buena Gardens, Moscone Center, PacBell Building and the San Francisco Museum of Modern Art.",
				&Address{
					"125",
					"3rd St",
					"San Francisco",
					"CA",
					"United States",
					"94109",
					lat,
					lon,
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

	collection := client.Database("profile-db").Collection("hotels")
	_, err = collection.InsertMany(context.TODO(), newProfiles)
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Successfully inserted test data into profile DB")

	return client, func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			log.Error(err.Error())
		}
	}
}
