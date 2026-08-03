package main

import (
	"fmt"
	"net/http"
	"strings"
	"alibabalb/utils"

	"google.golang.org/grpc/metadata"
	"alibabalb/pkg"
	pb "alibabalb/protobuf"
	"google.golang.org/grpc"
	"sync/atomic"
)

var envoyRPCSeq uint64


type Server struct {
	MS_2687Client pb.MS_2687Client
	MS_40087Client pb.MS_40087Client
	MS_51787Client pb.MS_51787Client
	MS_70124Client pb.MS_70124Client
}

const serviceName = "MS_64512"
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
		conn = pkg.GetConn(utils.GetEnvVar("MS_64512_EGRESS", true))
	}
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("MS_2687_ADDR", true))
	}
	s.MS_2687Client = pb.NewMS_2687Client(conn)
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("MS_40087_ADDR", true))
	}
	s.MS_40087Client = pb.NewMS_40087Client(conn)
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("MS_51787_ADDR", true))
	}
	s.MS_51787Client = pb.NewMS_51787Client(conn)
	if !meshProxy {
		conn = pkg.GetConn(utils.GetEnvVar("MS_70124_ADDR", true))
	}
	s.MS_70124Client = pb.NewMS_70124Client(conn)


	port := 2000
	if meshProxy {
		port = utils.StrToInt(utils.GetEnvVar("MS_64512_PORT", true))
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
	mux.Handle("/Z8trRkp4mp", baseHandler)

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
	case "Z8trRkp4mp":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		utils.BusyLoop(160)

		req := &pb.Request{}
		var err error
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.MS_2687Client.VdboDuPbKj(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.MS_40087Client.M2QxmWDHq1O(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.MS_40087Client.M5ISZV1SCx(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.MS_51787Client.RypaFB4PfJ(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID, "deadline", deadline))
		_, err = s.MS_70124Client.V0Gqd6H7Nw(ctx, req)
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
