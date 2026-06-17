package pkg

import (
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	_ "google.golang.org/grpc/resolver/dns"
)

func GetConn(addr string, extra ...grpc.DialOption) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if os.Getenv("plain_lb") == "true" {
		addr = "dns:///" + addr
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`))
	}
	opts = append(opts, extra...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}

func GetRajomonClient(addr string, interceptor grpc.DialOption) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), interceptor}
	if os.Getenv("plain_lb") == "true" {
		addr = "dns:///" + addr
		opts = append([]grpc.DialOption{grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`)}, opts...)
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}

func GetServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
	}
	return opts
}
