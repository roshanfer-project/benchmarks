package pkg

import (
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	_ "google.golang.org/grpc/balancer/leastrequest"
	_ "google.golang.org/grpc/balancer/roundrobin"
	_ "google.golang.org/grpc/resolver/dns"
)

func lbServiceConfig() string {
	switch os.Getenv("load_balancing_policy") {
	case "round_robin":
		return `{"loadBalancingConfig":[{"round_robin":{}}]}`
	case "weighted_round_robin":
		return `{"loadBalancingConfig":[{"weighted_round_robin":{"blackoutPeriod":"1s"}}]}`
	default:
		return `{"loadBalancingConfig":[{"least_request_experimental":{}}]}`
	}
}

func GetConn(addr string, extra ...grpc.DialOption) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if os.Getenv("plain_lb") == "true" {
		addr = "dns:///" + addr
		opts = append(opts, grpc.WithDefaultServiceConfig(lbServiceConfig()))
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
		opts = append([]grpc.DialOption{grpc.WithDefaultServiceConfig(lbServiceConfig())}, opts...)
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
