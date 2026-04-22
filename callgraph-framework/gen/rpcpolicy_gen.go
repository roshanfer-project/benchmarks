package gen

import (
	"os"
	"path/filepath"
)

func writeRPCPolicy(outDir string) error {
	dir := filepath.Join(outDir, "pkg", "rpcpolicy")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "rpcpolicy.go"), []byte(rpcpolicyGo), 0644)
}

const rpcpolicyGo = `package rpcpolicy

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
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
	DeadlineFixed   = "fixed" // BENCH_RPC_DEADLINE_MODE: per-outbound-call 2s timeout (see fixedRPCDeadline)
	RetryNone       = "none"
	RetryFixed      = "fixed" // BENCH_RPC_RETRY_MODE
	RetryTokenBucket = "token_bucket"
)

const fixedRPCDeadline = 1 * time.Second

func init() {
	logPolicyConfigAtStartup()
}

// logPolicyConfigAtStartup logs deadline/retry env once per process (any binary that imports this package).
func logPolicyConfigAtStartup() {
	dm := strings.TrimSpace(os.Getenv(EnvDeadlineMode))
	rm := strings.TrimSpace(os.Getenv(EnvRetryMode))
	if dm == "" {
		log.Printf("rpcpolicy: %s unset — effective %q", EnvDeadlineMode, DeadlineNone)
	} else {
		log.Printf("rpcpolicy: %s=%q", EnvDeadlineMode, dm)
	}
	if rm == "" {
		log.Printf("rpcpolicy: %s unset — effective %q", EnvRetryMode, RetryNone)
	} else {
		log.Printf("rpcpolicy: %s=%q", EnvRetryMode, rm)
	}

	switch dm {
	case DeadlineRemain:
		log.Printf("rpcpolicy: deadline policy remaining_slo — HTTP entry deadline = 0.6×BENCH_RPC_SLO_MS_<api> (propagates on gRPC); startup validation via MustValidatePolicyEnv")
	case DeadlineFixed:
		log.Printf("rpcpolicy: deadline policy fixed — per outbound unary RPC timeout=%v (DialClient interceptor; not sidecar); no entry SLO deadline", fixedRPCDeadline)
	}
	if dm != "" && dm != DeadlineNone && dm != DeadlineRemain && dm != DeadlineFixed {
		log.Printf("rpcpolicy: warning: %s=%q is not none|remaining_slo|fixed — deadline interceptors/MaybeDeadline unchanged", EnvDeadlineMode, dm)
	}

	switch rm {
	case RetryFixed:
		log.Printf("rpcpolicy: retry policy fixed — up to 4 unary attempts; backoff = slo_ms + uniform[0,slo_ms] from BENCH_RPC_SLO_MS_<api> (requires metadata api|method)")
	case RetryTokenBucket:
		capStr := strings.TrimSpace(os.Getenv("BENCH_RPC_RETRY_BUCKET_CAPACITY"))
		if capStr == "" {
			capStr = "10 (default)"
		}
		log.Printf("rpcpolicy: retry policy token_bucket — up to 4 attempts; BENCH_RPC_RETRY_BUCKET_CAPACITY=%s; success refund %.2f token/call; same backoff as fixed", capStr, tokenBucketSuccessCredit)
	}
	if rm != "" && rm != RetryNone && rm != RetryFixed && rm != RetryTokenBucket {
		log.Printf("rpcpolicy: warning: %s=%q is not none|fixed|token_bucket — no retry interceptor", EnvRetryMode, rm)
	}

	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "BENCH_RPC_SLO_MS_") {
			log.Printf("rpcpolicy: %s", e)
		}
	}
}

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

// FixedDeadlineUnaryInterceptorOpt applies a fresh 1s timeout per unary RPC when BENCH_RPC_DEADLINE_MODE=fixed.
// Placed inside the retry interceptor so each retry attempt gets its own 2s budget. No SLO / entry deadline; not end-to-end.
func FixedDeadlineUnaryInterceptorOpt(sidecar bool) grpc.UnaryClientInterceptor {
	if sidecar {
		return nil
	}
	if os.Getenv(EnvDeadlineMode) != DeadlineFixed {
		return nil
	}
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, fixedRPCDeadline)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// MaybeDeadlineForAPI returns ctx with deadline at now + 500% of SLO ms when mode is remaining_slo.
func MaybeDeadlineForAPI(parent context.Context, api string) (context.Context, context.CancelFunc) {
	if os.Getenv(EnvDeadlineMode) != DeadlineRemain {
		return parent, func() {}
	}
	v := os.Getenv(sloEnvKey(api))
	ms, err := strconv.Atoi(v)
	if err != nil || ms <= 0 {
		log.Fatalf("rpcpolicy: missing or invalid %s for api %q", sloEnvKey(api), api)
	}
	d := time.Duration(ms*5) * time.Millisecond
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

// retryBackoffDuration returns slo_ms as base plus uniform jitter in [0, slo_ms] ms.
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
	baseMs := slo
	jitterMs := rand.Float64() * slo
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

func logBeforeUnaryRetry(method string, retryNum, maxExtraAttempts int, last error) {
	if maxExtraAttempts < 1 {
		maxExtraAttempts = 1
	}
	log.Printf("rpcpolicy: before unary retry %d/%d method=%s err=%v", retryNum, maxExtraAttempts, method, last)
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
			logBeforeUnaryRetry(method, attempt+1, maxAttempts-1, last)
			if err := sleepRetryBackoff(ctx); err != nil {
				return err
			}
		}
		return last
	}
}

const tokenBucketSuccessCredit = 0.1 // tokens returned per successful RPC

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
}

func newTokenBucketFromEnv() *tokenBucket {
	cap := 100.0
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
				logBeforeUnaryRetry(method, attempt, maxAttempts-1, last)
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
`
