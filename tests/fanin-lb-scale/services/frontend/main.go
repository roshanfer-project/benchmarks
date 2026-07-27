package main

import (
	"fmt"
	"net/http"
	"strings"
	"faninlbscale/utils"

	"google.golang.org/grpc/metadata"
	"faninlbscale/pkg"
	pb "faninlbscale/protobuf"
	"google.golang.org/grpc"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var envoyRPCSeq uint64

var benchRng struct {
	mu sync.Mutex
	r  *rand.Rand
}

func init() {
	seed := time.Now().UnixNano()
	if s := os.Getenv("ROUTING_SEED"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			seed = v
		}
	}
	benchRng.r = rand.New(rand.NewSource(seed))
}

func benchFloat() float64 {
	benchRng.mu.Lock()
	defer benchRng.mu.Unlock()
	return benchRng.r.Float64()
}

func benchExpBusyLoop(mean float64) {
	benchRng.mu.Lock()
	rt := benchRng.r.ExpFloat64() * mean
	benchRng.mu.Unlock()
	repeats := int(rt * 320)
	if repeats < 1 {
		repeats = 1
	}
	utils.BusyLoop(repeats)
}


type Server struct {
	Backend1Client pb.Backend1Client
	Backend2Client pb.Backend2Client
}

const serviceName = "frontend"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing HTTP server...")
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	envoy := utils.GetEnvVar("envoy", false) == "true"
	if sidecar && envoy {
		panic("sidecar and envoy cannot both be enabled")
	}
	meshProxy := sidecar || envoy
	var conn *grpc.ClientConn
	if meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("frontend_EGRESS", true))
	}
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("backend1_ADDR", true))
	}
	s.Backend1Client = pb.NewBackend1Client(conn)
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("backend2_ADDR", true))
	}
	s.Backend2Client = pb.NewBackend2Client(conn)


	port := 2000
	if meshProxy {
		port = utils.StrToInt(utils.GetEnvVar("frontend_PORT", true))
	}
	mux := http.NewServeMux()
	var baseHandler http.Handler = http.HandlerFunc(s.handler)
	plain := utils.GetEnvVar("plain", false) == "true"
	plainLb := utils.GetEnvVar("plain_lb", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if plain || plainLb || (sidecar && queuingExport) {
		counter := utils.NewCounterState(serviceName)
		baseHandler = counter.GetHTTP1Middleware()(baseHandler)
	}
	mux.Handle("/f1", baseHandler)
	mux.Handle("/g1", baseHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	log.Info("Serving HTTP")
	return srv.ListenAndServe()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	envoy := utils.GetEnvVar("envoy", false) == "true"
	var rpcID, rpcLocalID, deadline string
	if sidecar {
		rpcID = r.Header.Get("rpc-id")
		if rpcID == "" {
			http.Error(w, "rpc-id header required", http.StatusBadRequest)
			return
		}
		rpcLocalID = r.Header.Get("rpc-local-id")
		if rpcLocalID == "" {
			http.Error(w, "rpc-local-id header required", http.StatusBadRequest)
			return
		}
		deadline = r.Header.Get("deadline")
		if deadline == "" {
			http.Error(w, "deadline header required", http.StatusBadRequest)
			return
		}
	} else if envoy {
		rpcID = r.Header.Get("rpc-id")
		rpcLocalID = r.Header.Get("rpc-local-id")
		if rpcID == "" {
			rpcID = fmt.Sprintf("%d", atomic.AddUint64(&envoyRPCSeq, 1))
		}
		if rpcLocalID == "" {
			rpcLocalID = rpcID
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	ctx := r.Context()
	switch path {
	case "f1":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f1", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		benchExpBusyLoop(0.5)

		req := &pb.Request{}
		var err error
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f1", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.Backend1Client.F2(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f1", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.Backend2Client.F4(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}


	case "g1":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "g1", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		benchExpBusyLoop(0.5)

		req := &pb.Request{}
		var err error
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "g1", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.Backend1Client.F3(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}


	default:
		http.Error(w, "not found", 404)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func main() {
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start server", "error", err)
	}
}
