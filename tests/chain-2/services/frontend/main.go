package main

import (
	"fmt"
	"net/http"
	"chain2/utils"

	"google.golang.org/grpc/metadata"

	"chain2/pkg"
	pb "chain2/protobuf"
	"google.golang.org/grpc"

)

type Server struct {
	BackendClient pb.BackendClient

}

const serviceName = "frontend"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing HTTP server...")
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("frontend_EGRESS", true))
	}
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("backend_ADDR", true))
	}
	s.BackendClient = pb.NewBackendClient(conn)


	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("frontend_PORT", true))
	}
	mux := http.NewServeMux()
	var handler http.Handler = http.HandlerFunc(s.handler)
	if sidecar && utils.GetEnvVar("queuing_export", false) == "true" {
		counter := utils.NewCounterState(serviceName)
		handler = counter.GetHTTP1Middleware()(handler)
	}
	mux.Handle("/f1", handler)
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
	var rpcID string
	if sidecar {
		rpcID = r.Header.Get("rpc-id")
		if rpcID == "" {
			http.Error(w, "rpc-id header required", http.StatusBadRequest)
			return
		}
	}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("method", "f1", "rpc-id", rpcID))
	utils.BusyLoop(96)
	req := &pb.Request{}
	var err error
	_, err = s.BackendClient.F2(ctx, req)
	if err != nil {
		log.Error("downstream call failed", "error", err)
		http.Error(w, err.Error(), 500)
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
