package main

import (
	"fmt"
	"net/http"
	"strings"
	"intermediatediverse/utils"

	"google.golang.org/grpc/metadata"
	"intermediatediverse/pkg"
	pb "intermediatediverse/protobuf"
	"google.golang.org/grpc"
	"sync/atomic"
)

var envoyRPCSeq uint64


type Server struct {
	Backend1Client pb.Backend1Client
	Backend2Client pb.Backend2Client
	Backend3Client pb.Backend3Client
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
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("backend3_ADDR", true))
	}
	s.Backend3Client = pb.NewBackend3Client(conn)


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
	mux.Handle("/f2", baseHandler)
	mux.Handle("/f3", baseHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	log.Info("Serving HTTP")
	return srv.ListenAndServe()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	envoy := utils.GetEnvVar("envoy", false) == "true"
	var rpcID, rpcLocalID string
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
	switch path {
	case "f1":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f1", "rpc-id", rpcID, "rpc-local-id", rpcLocalID))
		utils.BusyLoop(64)

		req := &pb.Request{}
		var err error
		_, err = s.Backend1Client.B1(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}


	case "f2":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f2", "rpc-id", rpcID, "rpc-local-id", rpcLocalID))
		utils.BusyLoop(64)

		req := &pb.Request{}
		var err error
		_, err = s.Backend2Client.B2(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}


	case "f3":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f3", "rpc-id", rpcID, "rpc-local-id", rpcLocalID))
		utils.BusyLoop(64)

		req := &pb.Request{}
		var err error
		_, err = s.Backend3Client.B3(ctx, req)
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
