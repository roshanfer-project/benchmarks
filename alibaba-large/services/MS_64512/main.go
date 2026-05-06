package main

import (
	"fmt"
	"net/http"
	"strings"
	"alibabalarge/utils"

	"google.golang.org/grpc/metadata"
	"alibabalarge/pkg"
	pb "alibabalarge/protobuf"
	"google.golang.org/grpc"
)


type Server struct {
	MS_14758Client pb.MS_14758Client
	MS_19439Client pb.MS_19439Client
	MS_21298Client pb.MS_21298Client
	MS_25781Client pb.MS_25781Client
	MS_25806Client pb.MS_25806Client
	MS_2687Client pb.MS_2687Client
	MS_40087Client pb.MS_40087Client
	MS_43032Client pb.MS_43032Client
	MS_51783Client pb.MS_51783Client
	MS_51787Client pb.MS_51787Client
	MS_53792Client pb.MS_53792Client
	MS_58796Client pb.MS_58796Client
	MS_62039Client pb.MS_62039Client
	MS_66921Client pb.MS_66921Client
	MS_67465Client pb.MS_67465Client
	MS_70124Client pb.MS_70124Client
	MS_9105Client pb.MS_9105Client
}

const serviceName = "MS_64512"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing HTTP server...")
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
	var conn *grpc.ClientConn
	if sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_64512_EGRESS", true))
	}
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_14758_ADDR", true))
	}
	s.MS_14758Client = pb.NewMS_14758Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_19439_ADDR", true))
	}
	s.MS_19439Client = pb.NewMS_19439Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_21298_ADDR", true))
	}
	s.MS_21298Client = pb.NewMS_21298Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_25781_ADDR", true))
	}
	s.MS_25781Client = pb.NewMS_25781Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_25806_ADDR", true))
	}
	s.MS_25806Client = pb.NewMS_25806Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_2687_ADDR", true))
	}
	s.MS_2687Client = pb.NewMS_2687Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_40087_ADDR", true))
	}
	s.MS_40087Client = pb.NewMS_40087Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_43032_ADDR", true))
	}
	s.MS_43032Client = pb.NewMS_43032Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_51783_ADDR", true))
	}
	s.MS_51783Client = pb.NewMS_51783Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_51787_ADDR", true))
	}
	s.MS_51787Client = pb.NewMS_51787Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_53792_ADDR", true))
	}
	s.MS_53792Client = pb.NewMS_53792Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_58796_ADDR", true))
	}
	s.MS_58796Client = pb.NewMS_58796Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_62039_ADDR", true))
	}
	s.MS_62039Client = pb.NewMS_62039Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_66921_ADDR", true))
	}
	s.MS_66921Client = pb.NewMS_66921Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_67465_ADDR", true))
	}
	s.MS_67465Client = pb.NewMS_67465Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_70124_ADDR", true))
	}
	s.MS_70124Client = pb.NewMS_70124Client(conn)
	if !sidecar {
		conn = pkg.GetConn(utils.GetEnvVar("MS_9105_ADDR", true))
	}
	s.MS_9105Client = pb.NewMS_9105Client(conn)


	port := 2000
	if sidecar {
		port = utils.StrToInt(utils.GetEnvVar("MS_64512_PORT", true))
	}
	mux := http.NewServeMux()
	var baseHandler http.Handler = http.HandlerFunc(s.handler)
	plain := utils.GetEnvVar("plain", false) == "true"
	queuingExport := utils.GetEnvVar("queuing_export", false) == "true"
	if plain || (sidecar && queuingExport) {
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
	ctx := r.Context()
	sidecar := utils.GetEnvVar("sidecar", false) == "true"
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
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	switch path {
	case "Z8trRkp4mp":
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("api", "Z8trRkp4mp", "rpc-id", rpcID, "rpc-local-id", rpcLocalID))
		utils.BusyLoop(160)

		req := &pb.Request{}
		var err error
		_, err = s.MS_14758Client.MuJZ40NDv(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_19439Client.KvuxGZYcwm(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_21298Client.Te9DKpWLH7(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_21298Client.QRB35KFger(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_25781Client.QsLpARXiz2(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_25806Client.M0PIREyu4Tb(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_2687Client.VdboDuPbKj(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_40087Client.M2QxmWDHq1O(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_40087Client.M5ISZV1SCx(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_43032Client.ZSdnWDdKmj(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_51783Client.ZMa4ZJ012X(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_51787Client.RypaFB4PfJ(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_53792Client.M8JkkxghEWB(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_58796Client.AbNb_BH36(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_62039Client.NK4Gw2Phix(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_66921Client.EFOECNqigM(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_67465Client.WIe9Cm5AqE(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_70124Client.V0Gqd6H7Nw(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_9105Client.ByihMu7_9Z(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = s.MS_9105Client.MsD67GoyH2(ctx, req)
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
