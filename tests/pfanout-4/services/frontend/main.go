package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"pfanout4/pkg/rpcpolicy"
	"pfanout4/utils"

	"google.golang.org/grpc/metadata"
	"pfanout4/pkg"
	pb "pfanout4/protobuf"
	"google.golang.org/grpc"
	"sync"
)



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"f1",
	})
}

type Server struct {
	Backend1Client pb.Backend1Client
	Backend2Client pb.Backend2Client
	Backend3Client pb.Backend3Client
	Backend4Client pb.Backend4Client
}

const serviceName = "frontend"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing HTTP server...")
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("frontend_EGRESS", true), sidecar)
	}
	if !sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("backend1_ADDR", true), sidecar)
	}
	s.Backend1Client = pb.NewBackend1Client(conn)
	if !sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("backend2_ADDR", true), sidecar)
	}
	s.Backend2Client = pb.NewBackend2Client(conn)
	if !sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("backend3_ADDR", true), sidecar)
	}
	s.Backend3Client = pb.NewBackend3Client(conn)
	if !sidecar {
		conn = pkg.DialClient(utils.GetEnvVar("backend4_ADDR", true), sidecar)
	}
	s.Backend4Client = pb.NewBackend4Client(conn)


	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("frontend_PORT", true))
	}
	mux := http.NewServeMux()
	var baseHandler http.Handler = http.HandlerFunc(s.handler)
	plain := utils.GetEnvVar("plain", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if plain || (sidecar && queuingExport) {
		counter := utils.NewCounterState(serviceName)
		baseHandler = counter.GetHTTP1Middleware()(baseHandler)
	}
	mux.Handle("/f1", baseHandler)

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
	path := strings.TrimPrefix(r.URL.Path, "/")
	switch path {
	case "f1":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "f1", "rpc-id", rpcID))
		if !sidecar {
			var cancel context.CancelFunc
			ctx, cancel = rpcpolicy.MaybeDeadlineForAPI(ctx, "f1")
			defer cancel()
		}
		utils.BusyLoop(64)

		req := &pb.Request{}
		var wg sync.WaitGroup
		var errMu sync.Mutex
		var firstErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Backend1Client.F2(ctx, req)
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
			_, e := s.Backend2Client.F3(ctx, req)
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
			_, e := s.Backend3Client.F4(ctx, req)
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
			_, e := s.Backend4Client.F5(ctx, req)
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
