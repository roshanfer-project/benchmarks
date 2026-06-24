package main

import (
	"fmt"
	"net/http"
	"strings"
	"pfanout2/utils"

	"google.golang.org/grpc/metadata"
	"pfanout2/pkg"
	pb "pfanout2/protobuf"
	"google.golang.org/grpc"
	"sync/atomic"
	"sync"
)

var envoyRPCSeq uint64


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
	mux.Handle("/api", baseHandler)

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
	ctx := r.Context()
	switch path {
	case "api":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "api", "rpc-id", rpcID, "rpc-local-id", rpcLocalID))
		utils.BusyLoop(96)

		req := &pb.Request{}
		var wg sync.WaitGroup
		var errMu sync.Mutex
		var firstErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Backend1Client.Svc(ctx, req)
			if e != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				errMu.Unlock()
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Backend2Client.Svc(ctx, req)
			if e != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = e
				}
				errMu.Unlock()
			}
		}()
		wg.Wait()
		if firstErr != nil {
			log.Error("downstream call failed", "error", firstErr)
			http.Error(w, firstErr.Error(), 500)
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
