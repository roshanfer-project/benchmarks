package hotel

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func DefaultServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		//grpc.WriteBufferSize(0),
	}
}
