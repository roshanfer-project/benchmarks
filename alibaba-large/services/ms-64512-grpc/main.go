package main

import (
	"context"
	"fmt"
	"net"
	"alibabalarge/pkg"
	pb "alibabalarge/protobuf"
	dagor "alibabalarge/dagor"
	dagorinit "alibabalarge/dagor_init"
	rajomoninit "alibabalarge/rajomon_init"
	"alibabalarge/pkg/rpcpolicy"
	"alibabalarge/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)



func init() {
	rpcpolicy.MustValidatePolicyEnv([]string{		"Z8trRkp4mp",
	})
}

type Server struct {
	pb.UnimplementedMS_64512Server
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
	MS_7103Client pb.MS_7103Client
	MS_9105Client pb.MS_9105Client
}

const serviceName = "ms-64512-grpc"
var log = utils.GetLogger(serviceName)

func (s *Server) Run() error {
	log.Info("Initializing gRPC server...")
	useRajomon := utils.GetEnvVar("rajomon", false) == "true"
	useDagor := utils.GetEnvVar("dagor", false) == "true"
	if useRajomon == useDagor {
		panic("entry-grpc requires exactly one of rajomon=true or dagor=true")
	}
	opts := pkg.GetServerOptions()
	var pt *rajomon.PriceTable
	var dn *dagor.Dagor
	if useRajomon {
		pt = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			pt.UnaryInterceptor))
	} else {
		dn = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			utils.NewCounterState(serviceName).GetInterceptor(),
			dn.UnaryInterceptorServer))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterMS_64512Server(srv, s)
	{
		addr := utils.GetEnvVar("MS_14758_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_14758Client = pb.NewMS_14758Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_19439_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_19439Client = pb.NewMS_19439Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_21298_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_21298Client = pb.NewMS_21298Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_25781_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_25781Client = pb.NewMS_25781Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_25806_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_25806Client = pb.NewMS_25806Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_2687_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_2687Client = pb.NewMS_2687Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_40087_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_40087Client = pb.NewMS_40087Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_43032_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_43032Client = pb.NewMS_43032Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_51783_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_51783Client = pb.NewMS_51783Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_51787_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_51787Client = pb.NewMS_51787Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_53792_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_53792Client = pb.NewMS_53792Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_58796_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_58796Client = pb.NewMS_58796Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_62039_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_62039Client = pb.NewMS_62039Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_66921_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_66921Client = pb.NewMS_66921Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_67465_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_67465Client = pb.NewMS_67465Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_70124_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_70124Client = pb.NewMS_70124Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_7103_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_7103Client = pb.NewMS_7103Client(conn)
	}
	{
		addr := utils.GetEnvVar("MS_9105_ADDR", true)
		var conn *grpc.ClientConn
		if useRajomon {
			conn = pkg.DialClient(addr, false, pt.UnaryInterceptorClient)
		} else {
			conn = pkg.DialClient(addr, false, dn.UnaryInterceptorClient)
		}
		s.MS_9105Client = pb.NewMS_9105Client(conn)
	}

	reflection.Register(srv)
	listenPort := utils.StrToInt(utils.GetEnvVar("EntryGRPCPort", true))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return srv.Serve(lis)
}


func (s *Server) Z8TrRkp4Mp(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	utils.BusyLoop(288)

	md, _ := metadata.FromIncomingContext(ctx)
	api := ""
	if v := md.Get("api"); len(v) == 1 {
		api = v[0]
	}
	switch api {
	case "Z8trRkp4mp":
		var err error
		_, err = s.MS_14758Client.MuJZ40NDv(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_19439Client.KvuxGZYcwm(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_21298Client.Te9DKpWLH7(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_21298Client.QRB35KFger(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_25781Client.QsLpARXiz2(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_25806Client.M0PIREyu4Tb(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_25806Client.FQN3ARekoW(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_2687Client.VdboDuPbKj(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_40087Client.M2QxmWDHq1O(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_40087Client.M5ISZV1SCx(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_43032Client.ZSdnWDdKmj(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_51783Client.ZMa4ZJ012X(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_51787Client.RypaFB4PfJ(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_53792Client.M8JkkxghEWB(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_58796Client.AbNb_BH36(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_62039Client.NK4Gw2Phix(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_66921Client.EFOECNqigM(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_67465Client.WIe9Cm5AqE(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_70124Client.V0Gqd6H7Nw(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_7103Client.RswWe4AhfE(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_9105Client.ByihMu7_9Z(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}
		_, err = s.MS_9105Client.MsD67GoyH2(ctx, req)
		if err != nil {
			log.Error("downstream call failed", "error", err)
			return nil, err
		}


	default:
	}
	return &pb.Response{}, nil
}


func main() {
	s := &Server{}
	log.Info("Starting server...")
	if err := s.Run(); err != nil {
		log.Error("Server failed", "error", err)
	}
}
