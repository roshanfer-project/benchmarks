package utils

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var logCounter = GetLogger("counter")

type CounterState struct {
	failedRPCCounter        map[string]int64
	acceptedRPCCounter      map[string]int64
	inReq                   map[string]int64
	outReq                  map[string]int64
	maxQueue                map[string]int64
	lock                    sync.Mutex
	startOnce               sync.Once
	maxQueueGauge           *prometheus.GaugeVec
	failedRPCCounterGauge   *prometheus.GaugeVec
	acceptedRPCCounterGauge *prometheus.GaugeVec
	registry                *prometheus.Registry
	promAddr                string
	serviceName             string
}

func NewCounterState(serviceName string) *CounterState {
	s := &CounterState{
		failedRPCCounter:   make(map[string]int64),
		acceptedRPCCounter: make(map[string]int64),
		inReq:              make(map[string]int64),
		outReq:             make(map[string]int64),
		maxQueue:           make(map[string]int64),
		lock:               sync.Mutex{},
		registry:           prometheus.NewRegistry(),
	}
	if strings.HasSuffix(serviceName, "-grpc") {
		s.serviceName = serviceName[:len(serviceName)-len("-grpc")]
	} else {
		s.serviceName = serviceName
	}
	s.promAddr = GetEnvVar("PROM_ADDR", false)

	s.maxQueueGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "max_queue", Help: "Maximum queue length for each RPC method"},
		[]string{"api"},
	)
	s.acceptedRPCCounterGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "accepted_rpc_counter", Help: "Accepted RPC counter for each RPC method"},
		[]string{"api"},
	)
	s.failedRPCCounterGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "failed_rpc_counter", Help: "Failed RPC counter for each RPC method"},
		[]string{"api"},
	)
	s.registry.MustRegister(s.maxQueueGauge)
	s.registry.MustRegister(s.acceptedRPCCounterGauge)
	s.registry.MustRegister(s.failedRPCCounterGauge)
	return s
}

func (s *CounterState) start() {
	s.startOnce.Do(func() {
		if s.promAddr != "" {
			go s.PushAll()
		}
	})
}

func (s *CounterState) GetInterceptor() grpc.UnaryServerInterceptor {
	s.start()
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			panic("metadata not found in context")
		}
		method := md.Get("method")
		if len(method) == 0 || len(method) > 1 {
			panic("method not found in metadata")
		}
		api := method[0]
		s.IncrementInReq(api)
		s.IncrementAcceptedRPCCounter(api)
		resp, err := handler(ctx, req)
		if err != nil {
			s.IncrementFailedRPCCounter(api)
		}
		s.IncrementOutReq(api)
		return resp, err
	}
}

func (s *CounterState) GetHTTP1Middleware() func(next http.Handler) http.Handler {
	s.start()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			api := strings.TrimPrefix(r.URL.Path, "/")
			if api == "" {
				api = "unknown"
			}
			s.IncrementInReq(api)
			s.IncrementAcceptedRPCCounter(api)
			next.ServeHTTP(w, r)
			s.IncrementOutReq(api)
		})
	}
}

func (s *CounterState) IncrementInReq(method string) {
	s.lock.Lock()
	s.inReq[method]++
	s.UpdateMaxQueue(method)
	s.lock.Unlock()
}

func (s *CounterState) IncrementOutReq(method string) {
	s.lock.Lock()
	s.outReq[method]++
	s.UpdateMaxQueue(method)
	s.lock.Unlock()
}

func (s *CounterState) UpdateMaxQueue(method string) {
	queueSize := s.inReq[method] - s.outReq[method]
	if queueSize > s.maxQueue[method] {
		s.maxQueue[method] = queueSize
	}
}

func (s *CounterState) PushMaxQueue() {
	s.lock.Lock()
	for method, queueSize := range s.maxQueue {
		s.maxQueueGauge.WithLabelValues(method).Set(float64(queueSize))
	}
	s.lock.Unlock()
}

func (s *CounterState) IncrementAcceptedRPCCounter(method string) {
	s.lock.Lock()
	s.acceptedRPCCounter[method]++
	s.lock.Unlock()
}

func (s *CounterState) PushAcceptedRPCCounter() {
	s.lock.Lock()
	for method, count := range s.acceptedRPCCounter {
		s.acceptedRPCCounterGauge.WithLabelValues(method).Set(float64(count))
	}
	s.lock.Unlock()
}

func (s *CounterState) IncrementFailedRPCCounter(method string) {
	s.lock.Lock()
	s.failedRPCCounter[method]++
	s.lock.Unlock()
}

func (s *CounterState) PushFailedRPCCounter() {
	s.lock.Lock()
	for method, count := range s.failedRPCCounter {
		s.failedRPCCounterGauge.WithLabelValues(method).Set(float64(count))
	}
	s.lock.Unlock()
}

func (s *CounterState) PushAll() {
	t := time.NewTicker(1 * time.Second)
	for range t.C {
		s.PushMaxQueue()
		s.PushAcceptedRPCCounter()
		s.PushFailedRPCCounter()
		if err := push.New(s.promAddr, s.serviceName).Gatherer(s.registry).Push(); err != nil {
			logCounter.Error("Could not push to Pushgateway", "error", err)
		}
	}
}
