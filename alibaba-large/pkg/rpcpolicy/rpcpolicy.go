package rpcpolicy

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	EnvDeadlineMode = "BENCH_RPC_DEADLINE_MODE"
	EnvRetryMode    = "BENCH_RPC_RETRY_MODE"
	DeadlineNone    = "none"
	DeadlineRemain  = "remaining_slo"
	RetryNone       = "none"
	RetryFixed      = "fixed"
	RetryTokenBucket = "token_bucket"
)

func sloEnvKey(api string) string {
	return "BENCH_RPC_SLO_MS_" + api
}

// MustValidatePolicyEnv exits if remaining_slo is set but any entry API SLO env is missing or invalid.
func MustValidatePolicyEnv(entryAPIs []string) {
	if os.Getenv(EnvDeadlineMode) != DeadlineRemain {
		return
	}
	for _, api := range entryAPIs {
		k := sloEnvKey(api)
		v := os.Getenv(k)
		if v == "" {
			log.Fatalf("rpcpolicy: %s=%s requires %s to be set", EnvDeadlineMode, DeadlineRemain, k)
		}
		ms, err := strconv.Atoi(v)
		if err != nil || ms <= 0 {
			log.Fatalf("rpcpolicy: invalid %s=%q", k, v)
		}
	}
}

// MaybeDeadlineForAPI returns ctx with deadline at now + 60% of SLO ms when mode is remaining_slo.
func MaybeDeadlineForAPI(parent context.Context, api string) (context.Context, context.CancelFunc) {
	if os.Getenv(EnvDeadlineMode) != DeadlineRemain {
		return parent, func() {}
	}
	v := os.Getenv(sloEnvKey(api))
	ms, err := strconv.Atoi(v)
	if err != nil || ms <= 0 {
		log.Fatalf("rpcpolicy: missing or invalid %s for api %q", sloEnvKey(api), api)
	}
	d := time.Duration(ms*60/100) * time.Millisecond
	return context.WithDeadline(parent, time.Now().Add(d))
}

func firstOutgoingMeta(ctx context.Context, key string) string {
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

// retryBackoffDuration returns sleep duration: 5% of SLO ms plus uniform jitter in [0, 1%] of SLO (same ms basis).
// Requires outgoing metadata "api" or "method" and a valid BENCH_RPC_SLO_MS_<api> env var.
func retryBackoffDuration(ctx context.Context) (time.Duration, error) {
	api := firstOutgoingMeta(ctx, "api")
	if api == "" {
		api = firstOutgoingMeta(ctx, "method")
	}
	if api == "" {
		return 0, fmt.Errorf("rpcpolicy: retry backoff requires outgoing metadata \"api\" or \"method\"")
	}
	v := os.Getenv(sloEnvKey(api))
	if v == "" {
		return 0, fmt.Errorf("rpcpolicy: retry backoff requires %s (api %q)", sloEnvKey(api), api)
	}
	ms, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("rpcpolicy: invalid %s=%q: %w", sloEnvKey(api), v, err)
	}
	if ms <= 0 {
		return 0, fmt.Errorf("rpcpolicy: %s must be positive, got %q", sloEnvKey(api), v)
	}
	slo := float64(ms)
	baseMs := slo * 0.05
	if baseMs < 1.0 {
		baseMs = 1.0
	}
	jitterMs := rand.Float64() * (slo * 0.01)
	totalMs := baseMs + jitterMs
	return time.Duration(totalMs * float64(time.Millisecond)), nil
}

func sleepRetryBackoff(ctx context.Context) error {
	d, err := retryBackoffDuration(ctx)
	if err != nil {
		return err
	}
	time.Sleep(d)
	return nil
}

func RetryUnaryInterceptorOpt(sidecar bool) grpc.UnaryClientInterceptor {
	if sidecar {
		return nil
	}
	switch os.Getenv(EnvRetryMode) {
	case RetryFixed:
		return fixedRetryUnary(4)
	case RetryTokenBucket:
		return tokenBucketRetryUnary()
	default:
		return nil
	}
}

func fixedRetryUnary(maxAttempts int) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var last error
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			last = invoker(ctx, method, req, reply, cc, opts...)
			if last == nil {
				return nil
			}
			if attempt+1 >= maxAttempts {
				break
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := sleepRetryBackoff(ctx); err != nil {
				return err
			}
		}
		return last
	}
}

const tokenBucketSuccessCredit = 0.02 // tokens returned per successful RPC

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
}

func newTokenBucketFromEnv() *tokenBucket {
	cap := 10.0
	if s := os.Getenv("BENCH_RPC_RETRY_BUCKET_CAPACITY"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			cap = v
		}
	}
	return &tokenBucket{tokens: cap, capacity: cap}
}

// take removes one token if available.
func (b *tokenBucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

// rewardSuccess adds tokenBucketSuccessCredit, capped at capacity.
func (b *tokenBucket) rewardSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens += tokenBucketSuccessCredit
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
}

func tokenBucketRetryUnary() grpc.UnaryClientInterceptor {
	b := newTokenBucketFromEnv()
	maxAttempts := 4
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var last error
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			if attempt > 0 {
				if !b.take() {
					if last != nil {
						return last
					}
					return status.Errorf(codes.ResourceExhausted, "retry token bucket empty")
				}
				if err := sleepRetryBackoff(ctx); err != nil {
					return err
				}
			}
			last = invoker(ctx, method, req, reply, cc, opts...)
			if last == nil {
				b.rewardSuccess()
				return nil
			}
		}
		return last
	}
}
