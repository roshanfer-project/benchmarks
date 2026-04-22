package main

import (
	"fmt"
	"net/http"
	"os"
	"alibabalarge/pkg"
	pb "alibabalarge/protobuf"
	dagorinit "alibabalarge/dagor_init"
	rajomoninit "alibabalarge/rajomon_init"
	"alibabalarge/pkg/rpcpolicy"
	"alibabalarge/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"Z8trRkp4mp",
	})
}

var log = utils.GetLogger("rajomon-client")

type Server struct {
	client     pb.MS_64512Client
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
		"protoService", "MS_64512",
	)
	var conn *grpc.ClientConn
	if useRajomon {
		pt := rajomoninit.GetPriceTable("client", true)
		log.Info("creating gRPC client (connection is lazy until first RPC)", "target", addr)
		conn = pkg.DialClient(addr, false, pt.UnaryInterceptorEnduser)
	} else {
		dn := dagorinit.GetDagorNode("client", false, true)
		log.Info("creating gRPC client (connection is lazy until first RPC)", "target", addr)
		conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
	}
	s.grpcTarget = addr
	s.client = pb.NewMS_64512Client(conn)
	log.Info("gRPC stub ready", "target", addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/Z8trRkp4mp", s.handle_Z8TrRkp4Mp)

	httpPort := utils.StrToInt(clientPort)
	log.Info("Serving HTTP", "listenAddr", fmt.Sprintf(":%d", httpPort), "grpcTarget", addr)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: mux,
	}
	return srv.ListenAndServe()
}


func (s *Server) handle_Z8TrRkp4Mp(w http.ResponseWriter, r *http.Request) {
	ctx := metadata.AppendToOutgoingContext(r.Context(), "method", "Z8trRkp4mp", "api", "Z8trRkp4mp")
	ctx, cancel := rpcpolicy.MaybeDeadlineForAPI(ctx, "Z8trRkp4mp")
	defer cancel()
	_, err := s.client.Z8TrRkp4Mp(ctx, &pb.Request{})
	if err != nil {
		st := status.Code(err)
		log.Error("RPC failed",
			"error", err,
			"grpcCode", st.String(),
			"grpcTarget", s.grpcTarget,
			"grpcMethod", "Z8TrRkp4Mp",
			"api", "Z8trRkp4mp",
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
