package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"social"
	dagorinit "social/dagor_init"
	rajomoninit "social/rajomon_init"
	"social/utils"
	"strconv"

	pb "social/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const serviceName = "client"

var log = utils.GetLogger(serviceName)

type Server struct {
	pb.UnimplementedRajomonClientServer

	nginxClient pb.NginxServiceClient
}

func tracingMiddleware1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Run() error {

	log.Info("Initializing client...")
	var tracingMiddleware func(next http.Handler) http.Handler
	tracingMiddleware = tracingMiddleware1

	log.Info("Initializing gRPC clients...")

	nginxEnv := utils.GetEnvVar("NginxGRPCAddr", true)
	var conn *grpc.ClientConn

	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("Rajomon is enabled, initializing Rajomon client...")
		priceTable := rajomoninit.GetPriceTable(serviceName, true)
		conn = social.GetRajomonClient(nginxEnv, grpc.WithUnaryInterceptor(priceTable.UnaryInterceptorEnduser))
	} else if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("Dagor is enabled, initializing Dagor client...")
		dagorNode := dagorinit.GetDagorNode(serviceName, false, true)
		conn = social.GetConn(nginxEnv, grpc.WithUnaryInterceptor(dagorNode.UnaryInterceptorClient))
	} else {
		panic("Either Rajomon or Dagor must be enabled")
	}
	s.nginxClient = pb.NewNginxServiceClient(conn)
	mux := http.NewServeMux()
	mux.Handle("/compose", tracingMiddleware(http.HandlerFunc(s.composeHandler)))
	mux.Handle("/user", tracingMiddleware(http.HandlerFunc(s.userHandler)))
	mux.Handle("/home", tracingMiddleware(http.HandlerFunc(s.homeHandler)))
	log.Info("Successful")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("ClientPort", true))),
		Handler: mux,
	}

	log.Info("Serving http")
	return srv.ListenAndServe()
}

func main() {
	src := &Server{}
	if err := src.Run(); err != nil {
		log.Error("Failed to run server: " + err.Error())
		panic(err)
	}
}

func (s *Server) composeHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("ComposePost RPC called")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "compose-post")
	username := "user1"
	arg := &pb.ComposePostRequest{
		CreatorId: username,
		Text:      r.URL.Query().Get("text"),
	}
	resp, err := s.nginxClient.ComposePost(ctx, arg)
	if err != nil {
		log.Error("Error composing post", "error", err)
		if status.Code(err) == codes.ResourceExhausted {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)

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

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("ReadHomeTimeline RPC called")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "read-home-timeline")
	arg := &pb.ReadHomeTimelineRequest{
		UserId: "user1",
	}
	resp, err := s.nginxClient.ReadHomeTimeline(ctx, arg)
	if err != nil {
		log.Error("Error reading home timeline", "error", err)
		if status.Code(err) == codes.ResourceExhausted {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
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

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("ReadUserTimeline RPC called")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "read-user-timeline")
	arg := &pb.ReadUserTimelineRequest{
		UserId: "user1",
	}
	resp, err := s.nginxClient.ReadUserTimeline(ctx, arg)
	if err != nil {
		log.Error("Error reading user timeline", "error", err)
		if status.Code(err) == codes.ResourceExhausted {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
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
