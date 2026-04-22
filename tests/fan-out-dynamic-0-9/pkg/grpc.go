package pkg

import (
	"time"

	"fanoutdynamic09/pkg/rpcpolicy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// DialClient dials addr with optional retry policy (when sidecar is false) and unary client interceptors (inner chain).
func DialClient(addr string, sidecar bool, unary ...grpc.UnaryClientInterceptor) *grpc.ClientConn {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	var chain []grpc.UnaryClientInterceptor
	if r := rpcpolicy.RetryUnaryInterceptorOpt(sidecar); r != nil {
		chain = append(chain, r)
	}
	if d := rpcpolicy.FixedDeadlineUnaryInterceptorOpt(sidecar); d != nil {
		chain = append(chain, d)
	}
	chain = append(chain, unary...)
	if len(chain) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(chain...))
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		panic("did not connect: " + err.Error())
	}
	return conn
}

func GetServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{Timeout: 120 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
	}
}
