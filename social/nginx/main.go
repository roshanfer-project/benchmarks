package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"social"
	pb "social/protobuf"
	"social/utils"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

type Server struct {
	composeClient pb.ComposePostClient
	homeClient    pb.HomeTimelineClient
	userClient    pb.UserTimelineClient
}

const serviceName = "nginx"

var log = utils.GetLogger(serviceName)

var tracer trace.Tracer

type CounterState struct {
	failedRPCCounter atomic.Int64
	inReq            sync.Map
	outReq           sync.Map
	maxQueue         sync.Map
	lock             sync.Mutex
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
var standardMethods map[string]string
var counters = &CounterState{}

func tracingMiddleware1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tracingMiddleware wraps an http.Handler and starts a trace span for each request.
func tracingMiddleware2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		// extract first part of the path as method name
		method := r.URL.Path[1:]
		defer span.End()
		counters.lock.Lock()
		counters.IncrementInReq(method)
		queueSize := counters.GetInReq(method) - counters.GetOutReq(method)
		if queueSize > counters.GetMaxQueue(method) {
			counters.IncrementMaxQueue(method, queueSize)
			maxQueueGuage.Record(ctx, queueSize, metric.WithAttributes(attribute.String("api", standardMethods[method])))
		}
		counters.lock.Unlock()
		next.ServeHTTP(w, r.WithContext(ctx))
		counters.lock.Lock()
		counters.IncrementOutReq(method)
		counters.lock.Unlock()
	})
}

var sidecar bool

func (s *Server) Run() error {
	var tracingMiddleware func(next http.Handler) http.Handler
	if (utils.GetEnvVar("sidecar", false) == "true") && (utils.GetEnvVar("queuing_export", false) == "true") {
		tracingMiddleware = tracingMiddleware1
	} else {
		tracingMiddleware = tracingMiddleware1
	}

	log.Info("Initializing gRPC clients...")
	var composeEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		composeEnv = "NginxEgress"
	} else {
		composeEnv = "ComposeAddr"
	}
	conn := social.GetConn(utils.GetEnvVar(composeEnv, true))
	s.composeClient = pb.NewComposePostClient(conn)

	var home string
	if utils.GetEnvVar("sidecar", false) == "true" {
		home = "NginxEgress"
	} else {
		home = "HomeAddr"
	}
	conn = social.GetConn(utils.GetEnvVar(home, true))
	s.homeClient = pb.NewHomeTimelineClient(conn)

	var userEnv string
	if utils.GetEnvVar("sidecar", false) == "true" {
		userEnv = "NginxEgress"
	} else {
		userEnv = "UserAddr"
	}
	conn = social.GetConn(utils.GetEnvVar(userEnv, true))
	s.userClient = pb.NewUserTimelineClient(conn)

	log.Info("Successful")

	sidecar = utils.GetEnvVar("sidecar", false) == "true"

	// initialize
	/* populatePosts(
		s,
		utils.StrToInt(utils.GetEnvVar("num_of_users", true)),
		utils.StrToInt(utils.GetEnvVar("num_of_posts", true)),
	) */
	log.Info("[PopulatePosts] Finished populating posts")

	// Configure OTL
	//ctx := context.Background()
	//social.ConfigOTL(ctx, serviceName, true)
	tracer = otel.GetTracerProvider().Tracer(serviceName + "-tracer")

	mux := http.NewServeMux()
	mux.Handle("/compose", tracingMiddleware(http.HandlerFunc(s.composeHandler)))
	mux.Handle("/user", tracingMiddleware(http.HandlerFunc(s.userHandler)))
	mux.Handle("/home", tracingMiddleware(http.HandlerFunc(s.homeHandler)))

	log.Info("frontend starts serving")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("NginxPort", true))),
		Handler: mux,
	}

	log.Info("Serving http")
	return srv.ListenAndServe()
}

func main() {
	standardMethods = make(map[string]string)
	standardMethods["compose"] = "compose-post"
	standardMethods["user"] = "read-user-timeline"
	standardMethods["home"] = "read-home-timeline"
	/* var ok error
	maxQueueGuage, ok = otel.GetMeterProvider().Meter(serviceName).Int64Gauge("max_queue",
		metric.WithDescription("Maximum queue length for each RPC method"))
	if ok != nil {
		log.Error("Failed to create max_queue gauge")
		panic("Failed to create max_queue gauge")
	} */
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start server", "error", err)
	}
	log.Info("Server stopped")
}

func getRandomString(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

func populatePosts(s *Server, numOfUsers int, numOfPosts int) {

	for i := 0; i < numOfUsers; i++ {
		userId := fmt.Sprintf("user%d", i)

		for j := 0; j < numOfPosts; j++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			text := getRandomString(100)
			req := &pb.ComposePostRequest{
				CreatorId: userId,
				Text:      text,
			}

			// Compose post
			resp, err := s.composeClient.ComposePost(ctx, req)
			if err != nil {
				log.Error("[ComposePost] Error composing post", "userId", userId, "error", err)
				panic(err)
			}

			postId := resp.PostId
			//log.Printf("Composed post for user %s with post ID: %s", userId, postId)
			log.Debug("[ComposePost] Composed post for user", "userId", userId, "postId", postId)

			// Read from home timeline
			homeReq := &pb.ReadHomeTimelineRequest{UserId: userId}
			homeResp, err := s.homeClient.ReadHomeTimeline(ctx, homeReq)
			if err != nil {
				log.Error("[ReadHomeTimeline] Error reading home timeline", "userId", userId, "error", err)
				panic(err)
			} else {
				log.Debug("[ReadHomeTimeline] Home timeline for user", "userId", userId, "posts", homeResp.Posts)
			}

			// Read from user timeline
			userReq := &pb.ReadUserTimelineRequest{UserId: userId}
			userResp, err := s.userClient.ReadUserTimeline(ctx, userReq)
			if err != nil {
				log.Error("[ReadUserTimeline] Error reading user timeline", "userId", userId, "error", err)
				panic(err)
			} else {
				log.Debug("[ReadUserTimeline] User timeline for user", "userId", userId, "posts", userResp.Posts)
			}
		}
	}
}

func (s *Server) composeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var rpc_id string
	if sidecar {
		rpc_id = r.Header.Get("rpc-id")
		if rpc_id == "" {
			http.Error(w, "Please specify rpc-id", http.StatusBadRequest)
			return
		}
	} else {
		rpc_id = ""
	}
	ctx := r.Context()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("method", "compose-post", "rpc-id", rpc_id))

	log.Debug("Start composeHandler")

	username := "user1"

	req := &pb.ComposePostRequest{
		CreatorId: username,
		Text:      r.URL.Query().Get("text"),
	}
	resp, err := s.composeClient.ComposePost(ctx, req)
	if err != nil {
		log.Error("[ComposePost] Error forwarding compose post request.", "error", err)
		http.Error(w, "Error forwarding compose post request", 500)
		return
	}

	// return json response
	w.Header().Set("Content-Type", "application/json")
	// convert to json
	responseBytes, err := json.Marshal(resp)
	if err != nil {
		log.Error("[ComposePost] Error marshalling response to JSON.", "error", err)
		http.Error(w, "Error marshalling response to JSON", 500)
		return
	}
	// set content-length
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var rpc_id string
	if sidecar {
		rpc_id = r.Header.Get("rpc-id")
		if rpc_id == "" {
			http.Error(w, "Please specify rpc-id", http.StatusBadRequest)
			return
		}
	} else {
		rpc_id = ""
	}
	ctx := r.Context()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("method", "read-user-timeline", "rpc-id", rpc_id))

	log.Debug("Start userHandler")

	username := "user1"

	req := &pb.ReadUserTimelineRequest{UserId: username}
	resp, err := s.userClient.ReadUserTimeline(ctx, req)
	if err != nil {
		log.Error("[ReadUserTimeline] Error forwarding read user timeline request.", "error", err)
		http.Error(w, "Error forwarding read user timeline request", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(resp)
	if err != nil {
		log.Error("[ReadUserTimeline] Error marshalling response to JSON.", "error", err)
		http.Error(w, "Error marshalling response to JSON", 500)
		return
	}
	// set content-length
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var rpc_id string
	if sidecar {
		rpc_id = r.Header.Get("rpc-id")
		if rpc_id == "" {
			http.Error(w, "Please specify rpc-id", http.StatusBadRequest)
			return
		}
	} else {
		rpc_id = ""
	}
	ctx := r.Context()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("method", "read-home-timeline", "rpc-id", rpc_id))

	log.Debug("Start homeHandler")

	username := "user1"

	req := &pb.ReadHomeTimelineRequest{UserId: username}
	resp, err := s.homeClient.ReadHomeTimeline(ctx, req)
	if err != nil {
		log.Error("[ReadHomeTimeline] Error forwarding read home timeline request.", "error", err)
		http.Error(w, "Error forwarding read home timeline request", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(resp)
	if err != nil {
		log.Error("[ReadHomeTimeline] Error marshalling response to JSON.", "error", err)
		http.Error(w, "Error marshalling response to JSON", 500)
		return
	}
	// set content-length
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}
