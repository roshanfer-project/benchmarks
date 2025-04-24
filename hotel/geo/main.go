package main

import (
	"context"
	"fmt"
	oteltool "hotel/otel_tool"
	"net"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats/opentelemetry"

	pb "hotel/geo/proto"
	"hotel/utils"

	"github.com/google/uuid"
	"github.com/hailocab/go-geoindex"
)

var log = utils.GetLogger("geo")
var tracer trace.Tracer

const (
	name             = "srv-geo"
	maxSearchRadius  = 10
	maxSearchResults = 5
)

type Server struct {
	pb.UnimplementedGeoServer

	index *geoindex.ClusteringIndex
	uuid  string

	MongoClient *mongo.Client
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
		grpc.UnaryInterceptor(tracingInterceptor),
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

	center := &geoindex.GeoPoint{
		Pid:  "",
		Plat: lat,
		Plon: lon,
	}

	return s.index.KNearest(
		center,
		maxSearchResults,
		geoindex.Km(maxSearchRadius), func(p geoindex.Point) bool {
			return true
		},
	)
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

	log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("GeoMongoAddress", true))
	defer mongoClose()

	srv := &Server{
		MongoClient: mongoClient,
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
