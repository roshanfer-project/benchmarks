package utils

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestCounterStateRecordsQueuingDelay(t *testing.T) {
	s := NewCounterState("backend1-grpc")
	ctx := context.WithValue(context.Background(), tapTimeKey{}, time.Now().Add(-1500*time.Microsecond))
	md := metadata.Pairs("api", "f1")
	ctx = metadata.NewIncomingContext(ctx, md)

	ic := s.GetInterceptor()
	_, _ = ic(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	count := testutilHistogramSampleCount(t, s, "f1")
	if count < 1 {
		t.Fatalf("got sample count %d, want >= 1", count)
	}
}

func TestQueuingDelayReset(t *testing.T) {
	s := NewCounterState("backend1-grpc")
	ctx := context.WithValue(context.Background(), tapTimeKey{}, time.Now().Add(-100*time.Microsecond))
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("api", "f1"))
	ic := s.GetInterceptor()
	_, _ = ic(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if testutilHistogramSampleCount(t, s, "f1") < 1 {
		t.Fatal("expected samples before reset")
	}
	s.queuingDelay.Reset()
	if histogramSampleCount(s, "f1") != 0 {
		t.Fatal("expected zero samples after reset")
	}
}

func histogramSampleCount(s *CounterState, api string) uint64 {
	mfs, err := s.registry.Gather()
	if err != nil {
		return 0
	}
	for _, mf := range mfs {
		if mf.GetName() != "queuing_delay_microseconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			apiLabel := ""
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "api" {
					apiLabel = lp.GetValue()
				}
			}
			if apiLabel != api {
				continue
			}
			if h := m.GetHistogram(); h != nil {
				return h.GetSampleCount()
			}
		}
	}
	return 0
}

func testutilHistogramSampleCount(t *testing.T, s *CounterState, api string) uint64 {
	t.Helper()
	count := histogramSampleCount(s, api)
	if count == 0 {
		t.Fatalf("histogram queuing_delay_microseconds{api=%s} not found or empty", api)
	}
	return count
}
