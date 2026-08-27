package main

import (
	"fmt"
	"net/http"
	"os"
	"leaf12p21/pkg"
	pb "leaf12p21/protobuf"
	dagorinit "leaf12p21/dagor_init"
	rajomoninit "leaf12p21/rajomon_init"
	"leaf12p21/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var log = utils.GetLogger("rajomon-client")

type Server struct {
	client     pb.FrontendClient
	grpcTarget string
}

func (s *Server) Run() error {
	log.Info("Initializing rajomon-client...")
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	if useRajomon == useDagor {
		panic("rajomon-client requires exactly one of rajomon=true or dagor=true")
	}
	addr := utils.GetEnvVar("EntryGRPCAddr", true)
	clientPort := utils.GetEnvVar("ClientPort", true)
	deployment := utils.GetEnvVar("deployment", false)
	if hn, err := os.Hostname(); err == nil {
		log.Info("pod identity", "hostname", hn)
	}
	log.Info("rajomon-client config",
		"EntryGRPCAddr", addr,
		"ClientPort", clientPort,
		"deployment", deployment,
		"protoService", "Frontend",
	)
	var conn *grpc.ClientConn
	if useRajomon {
		pt := rajomoninit.GetPriceTable("client", true)
		log.Info("creating gRPC client (connection is lazy until first RPC)", "target", addr)
		conn = pkg.GetRajomonClient(addr, grpc.WithUnaryInterceptor(pt.UnaryInterceptorEnduser))
	} else {
		dn := dagorinit.GetDagorNode("client", false, true)
		log.Info("creating gRPC client (connection is lazy until first RPC)", "target", addr)
		conn = pkg.GetConn(addr, grpc.WithUnaryInterceptor(dn.UnaryInterceptorClient))
	}
	s.grpcTarget = addr
	s.client = pb.NewFrontendClient(conn)
	log.Info("gRPC stub ready", "target", addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/api1", s.handle_Api1)
	mux.HandleFunc("/api2", s.handle_Api2)

	httpPort := utils.StrToInt(clientPort)
	log.Info("Serving HTTP", "listenAddr", fmt.Sprintf(":%d", httpPort), "grpcTarget", addr)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: mux,
	}
	return srv.ListenAndServe()
}


func (s *Server) handle_Api1(w http.ResponseWriter, r *http.Request) {
	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "api1", "api", "api1")
	_, err := s.client.Api1(ctx, &pb.Request{})
	if err != nil {
		st := status.Code(err)
		log.Error("RPC failed",
			"error", err,
			"grpcCode", st.String(),
			"grpcTarget", s.grpcTarget,
			"grpcMethod", "Api1",
			"api", "api1",
		)
		if st == codes.ResourceExhausted {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func (s *Server) handle_Api2(w http.ResponseWriter, r *http.Request) {
	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "api2", "api", "api2")
	_, err := s.client.Api2(ctx, &pb.Request{})
	if err != nil {
		st := status.Code(err)
		log.Error("RPC failed",
			"error", err,
			"grpcCode", st.String(),
			"grpcTarget", s.grpcTarget,
			"grpcMethod", "Api2",
			"api", "api2",
		)
		if st == codes.ResourceExhausted {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}


func main() {
	s := &Server{}
	if err := s.Run(); err != nil {
		log.Error("Failed to start", "error", err)
	}
}
