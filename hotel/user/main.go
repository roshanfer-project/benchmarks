package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hotel"

	dagorinit "hotel/dagor_init"
	rajomoninit "hotel/rajomon_init"
	pb "hotel/user/proto"
	"net"
	"strconv"

	"hotel/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	oteltool "hotel/otel_tool"

	"hotel/dagor"

	"github.com/google/uuid"
	"github.com/pennsail/rajomon"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats/opentelemetry"
)

const serviceName = "user"

var log = utils.GetLogger(serviceName)

//var tracer trace.Tracer

type Server struct {
	pb.UnimplementedUserServer
	uuid        string
	users       map[string]string
	MongoClient *mongo.Client
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

	if s.users == nil {
		s.users = loadUsers(s.MongoClient)
	}

	s.uuid = uuid.New().String()

	opts := hotel.DefaultServerOptions()

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			priceTable.UnaryInterceptor))
		//opts = append(opts, grpc.UnaryInterceptor(priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor = nil
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			dagorNode.UnaryInterceptorServer))
		//opts = append(opts, grpc.UnaryInterceptor(dagorNode.UnaryInterceptorServer))
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

	pb.RegisterUserServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("UserPort", true))))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	return srv.Serve(lis)
}

func (s *Server) CheckUser(ctx context.Context, req *pb.Request) (*pb.Result, error) {
	res := new(pb.Result)

	//log.Trace().Msg("CheckUser")

	sum := sha256.Sum256([]byte(req.Password))
	pass := fmt.Sprintf("%x", sum)

	res.Correct = false
	if true_pass, found := s.users[req.Username]; found {
		res.Correct = pass == true_pass
	}

	//log.Trace().Msgf("CheckUser %d", res.Correct)

	return res, nil
}

type User struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

func loadUsers(client *mongo.Client) map[string]string {
	collection := client.Database("user-db").Collection("user")
	curr, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Error("Failed get users data: " + err.Error())
	}

	var users []User
	curr.All(context.TODO(), &users)
	if err != nil {
		log.Error("Failed get users data: " + err.Error())
	}

	res := make(map[string]string)
	for _, user := range users {
		res[user.Username] = user.Password
	}

	//log.Trace().Msg("Done load users")

	return res
}

func initializeDatabase(url string) (*mongo.Client, func()) {
	log.Info("Generating test data...")

	newUsers := []interface{}{}

	for i := 0; i <= 500; i++ {
		suffix := strconv.Itoa(i)

		password := ""
		for j := 0; j < 10; j++ {
			password += suffix
		}
		sum := sha256.Sum256([]byte(password))

		newUsers = append(newUsers, User{
			fmt.Sprintf("Cornell_%x", suffix),
			fmt.Sprintf("%x", sum),
		})
	}

	uri := fmt.Sprintf("mongodb://%s", url)
	log.Info(fmt.Sprintf("Attempting connection to %v", uri))

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err.Error())
	}
	log.Info("Successfully connected to MongoDB")

	collection := client.Database("user-db").Collection("user")
	_, err = collection.InsertMany(context.TODO(), newUsers)
	if err != nil {
		panic(err.Error())
	}
	log.Info("Successfully inserted test data into user DB")

	return client, func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err.Error())
		}
	}
}

func main() {
	log.Info("Reading config...")
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
	log.Info("Initializing DB connection...")
	mongoClient, mongoClose := initializeDatabase(utils.GetEnvVar("UserMongoAddress", true))
	defer mongoClose()

	srv := &Server{
		MongoClient: mongoClient,
	}

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}
