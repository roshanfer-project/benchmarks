package utils

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// EnvAdminAddr is the listen address for the fail-slow admin HTTP server (loopback only recommended).
const EnvAdminAddr = "BENCH_FAILSLOW_ADMIN_ADDR"

func jsonNumberToInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case json.Number:
		i, err := x.Int64()
		return i, err == nil
	case int64:
		return x, true
	case int:
		return int64(x), true
	default:
		return 0, false
	}
}

var failslowState struct {
	mu        sync.Mutex
	windowEnd time.Time
	extra     time.Duration
}

// Arm sets the fail-slow window: until windowEnd, each PostProcess sleeps up to extra (canceled by ctx).
func Arm(windowEnd time.Time, extra time.Duration) {
	failslowState.mu.Lock()
	defer failslowState.mu.Unlock()
	failslowState.windowEnd = windowEnd
	failslowState.extra = extra
}

// PostProcess applies extra latency when the armed window is still active (gRPC: after handler, before return).
func PostProcess(ctx context.Context) {
	failslowState.mu.Lock()
	end := failslowState.windowEnd
	extra := failslowState.extra
	failslowState.mu.Unlock()
	if extra <= 0 || time.Now().After(end) {
		return
	}
	t := time.NewTimer(extra)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
	}
}

// FailslowUnaryServerInterceptor delays successful RPC responses while the fail-slow window is active.
func FailslowUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, err
		}
		PostProcess(ctx)
		return resp, err
	}
}

// StartFailslowAdmin listens for POST /failslow on BENCH_FAILSLOW_ADMIN_ADDR (default 127.0.0.1:19090).
func StartFailslowAdmin() {
	addr := os.Getenv(EnvAdminAddr)
	if addr == "" {
		addr = "127.0.0.1:19090"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/failslow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		durMS, ok1 := jsonNumberToInt64(raw["duration_ms"])
		extraMS, ok2 := jsonNumberToInt64(raw["extra_ms"])
		if !ok1 || !ok2 {
			http.Error(w, "need duration_ms and extra_ms", http.StatusBadRequest)
			return
		}
		if durMS < 0 || extraMS < 0 {
			http.Error(w, "duration_ms and extra_ms must be non-negative", http.StatusBadRequest)
			return
		}
		end := time.Now().Add(time.Duration(durMS) * time.Millisecond)
		Arm(end, time.Duration(extraMS)*time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	})
	go func() {
		s := &http.Server{Addr: addr, Handler: mux}
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("failslow admin: %v", err)
		}
	}()
}
