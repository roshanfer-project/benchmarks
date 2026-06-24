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

func ReplicaJobName(base string) string {
	if GetEnvVar("plain_lb", false) != "true" {
		return base
	}
	pod := GetEnvVar("POD_NAME", false)
	if pod == "" {
		return base
	}
	if strings.HasPrefix(pod, base+"-") {
		return base + "-" + strings.TrimPrefix(pod, base+"-")
	}
	return base + "-" + pod
}

type CounterState struct {
	failedRPCCounter        map[string]int64
	acceptedRPCCounter      map[string]int64
	inReq                   map[string]int64
	outReq                  map[string]int64
	maxQueue                map[string]int64
	queueIntegral           map[string]float64
	lock                    sync.Mutex
	startOnce               sync.Once
	startTime               time.Time
	lastEventTime           time.Time
	maxQueueGauge           *prometheus.GaugeVec
	avgQueueGauge           *prometheus.GaugeVec
	failedRPCCounterGauge   *prometheus.GaugeVec
	acceptedRPCCounterGauge *prometheus.GaugeVec
	registry                *prometheus.Registry
	promAddr                string
	serviceName             string
	pushJobName             string
}

func NewCounterState(serviceName string) *CounterState {
	s := &CounterState{
		failedRPCCounter:   make(map[string]int64),
		acceptedRPCCounter: make(map[string]int64),
		inReq:              make(map[string]int64),
		outReq:             make(map[string]int64),
		maxQueue:           make(map[string]int64),
		queueIntegral:      make(map[string]float64),
		lock:               sync.Mutex{},
		registry:           prometheus.NewRegistry(),
		startTime:          time.Now(),
	}
	if strings.HasSuffix(serviceName, "-grpc") {
		s.serviceName = serviceName[:len(serviceName)-len("-grpc")]
	} else {
		s.serviceName = serviceName
	}
	s.pushJobName = ReplicaJobName(s.serviceName)
	s.promAddr = GetEnvVar("PROM_ADDR", false)

	s.maxQueueGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "max_queue", Help: "Maximum queue length for each RPC method"},
		[]string{"api"},
	)
	s.avgQueueGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "avg_queue", Help: "Time-averaged queue length for each RPC method"},
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
	s.registry.MustRegister(s.avgQueueGauge)
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
		keys := md.Get("api")
		if len(keys) == 0 || len(keys) > 1 {
			panic("api not found in metadata")
		}
		api := keys[0]
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

func (s *CounterState) accumulateIntegral(now time.Time) {
	if s.lastEventTime.IsZero() {
		s.lastEventTime = now
		return
	}
	dt := now.Sub(s.lastEventTime).Seconds()
	if dt <= 0 {
		return
	}
	for m := range s.inReq {
		q := s.inReq[m] - s.outReq[m]
		s.queueIntegral[m] += float64(q) * dt
	}
	s.lastEventTime = now
}

func (s *CounterState) IncrementInReq(method string) {
	s.lock.Lock()
	s.accumulateIntegral(time.Now())
	s.inReq[method]++
	s.UpdateMaxQueue(method)
	s.lock.Unlock()
}

func (s *CounterState) IncrementOutReq(method string) {
	s.lock.Lock()
	s.accumulateIntegral(time.Now())
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
		s.lock.Lock()
		now := time.Now()
		s.accumulateIntegral(now)
		for method, queueSize := range s.maxQueue {
			s.maxQueueGauge.WithLabelValues(method).Set(float64(queueSize))
		}
		elapsed := now.Sub(s.startTime).Seconds()
		if elapsed < 1e-9 {
			elapsed = 1e-9
		}
		for method := range s.inReq {
			avg := s.queueIntegral[method] / elapsed
			s.avgQueueGauge.WithLabelValues(method).Set(avg)
		}
		s.lock.Unlock()
		s.PushAcceptedRPCCounter()
		s.PushFailedRPCCounter()
		if err := push.New(s.promAddr, s.pushJobName).Gatherer(s.registry).Push(); err != nil {
			logCounter.Error("Could not push to Pushgateway", "error", err)
		}
	}
}
