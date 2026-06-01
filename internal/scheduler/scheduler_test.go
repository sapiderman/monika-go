package scheduler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"monika-go/internal/config"
	"monika-go/internal/logger/loggertest"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{}
	nopLog := loggertest.NopLogger{}
	s := New(cfg, nopLog)

	if s == nil {
		t.Fatal("expected New to return a non-nil Scheduler")
	}
	if s.cfg != cfg {
		t.Error("expected Scheduler to hold the passed configuration")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Probes: []config.Probe{
			{
				ID:       "test-http",
				Name:     "Test HTTP Probe",
				Interval: 1, // 1 second interval
				Spec: &config.HTTPSpec{
					Requests: []config.Request{
						{
							URL:     server.URL,
							Method:  "GET",
							Timeout: 1000,
						},
					},
				},
			},
		},
	}

	captureLog := &loggertest.CaptureLogger{}
	s := New(cfg, captureLog)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Wait for at least one tick to execute (interval is 1 second)
	time.Sleep(1200 * time.Millisecond)

	s.Stop()

	// Verify that the request actually reached our test server
	count := atomic.LoadInt32(&requestCount)
	if count == 0 {
		t.Error("expected at least one probe execution request to be handled, got 0")
	}

	// Verify scheduler logs
	entries := captureLog.Entries()
	var foundStart, foundComp bool
	for _, entry := range entries {
		if entry.Msg == "starting scheduler" {
			foundStart = true
		}
		if entry.Msg == "probe completed" {
			foundComp = true
		}
	}

	if !foundStart {
		t.Error("expected log message 'starting scheduler' not found")
	}
	if !foundComp {
		t.Error("expected log message 'probe completed' not found")
	}
}

func TestScheduler_DefaultIntervalFallback(t *testing.T) {
	cfg := &config.Config{
		Probes: []config.Probe{
			{
				ID:       "fallback-probe",
				Name:     "Fallback Interval Probe",
				Interval: 0, // Should fallback to 10 seconds
				Spec: &config.HTTPSpec{
					Requests: []config.Request{
						{
							URL: "https://localhost:9999",
						},
					},
				},
			},
		},
	}

	captureLog := &loggertest.CaptureLogger{}
	s := New(cfg, captureLog)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Sleep briefly so start is recorded but no execution ticks trigger
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	entries := captureLog.Entries()
	var foundFallbackLog bool
	for _, entry := range entries {
		if entry.Msg == "starting probe loop" {
			for _, field := range entry.Fields {
				if field.Key == "interval_seconds" && field.Value == 10 {
					foundFallbackLog = true
				}
			}
		}
	}

	if !foundFallbackLog {
		t.Error("expected probe loop to start with fallback interval of 10 seconds, log not found or incorrect")
	}
}

func TestScheduler_UnsupportedProberType(t *testing.T) {
	// Let's create a custom dummy ProbeSpec that is indeed unsupported
	// (since all native ones: ping, socket, mongo, redis, pg, maria, mysql are supported now!)
	cfg := &config.Config{
		Probes: []config.Probe{
			{
				ID:       "dummy-unsupported-probe",
				Name:     "Unsupported Probe",
				Interval: 1,
				Spec:     &unsupportedDummySpec{},
			},
		},
	}

	captureLog := &loggertest.CaptureLogger{}
	s := New(cfg, captureLog)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Sleep briefly so probe ticks and logs skip warning
	time.Sleep(1050 * time.Millisecond)
	s.Stop()

	entries := captureLog.Entries()
	var foundSkipWarning bool
	for _, entry := range entries {
		if entry.Msg == "skipping probe: unsupported or unimplemented prober type" {
			foundSkipWarning = true
		}
	}

	if !foundSkipWarning {
		t.Error("expected log warning for unsupported/unimplemented prober type, but none was logged")
	}
}

type unsupportedDummySpec struct{}

func (s *unsupportedDummySpec) Kind() config.ProbeKind { return "unsupported-dummy" }
func (s *unsupportedDummySpec) Validate() error        { return nil }
