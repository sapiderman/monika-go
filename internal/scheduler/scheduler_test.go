package scheduler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"monika-go/internal/alert"
	"monika-go/internal/assertion"
	"monika-go/internal/config"
	"monika-go/internal/logger"
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
	var completedEntry loggertest.CaptureEntry
	for _, entry := range entries {
		if entry.Msg == "starting scheduler" {
			foundStart = true
		}
		if entry.Msg == "probe completed" {
			foundComp = true
			completedEntry = entry
		}
	}

	if !foundStart {
		t.Error("expected log message 'starting scheduler' not found")
	}
	if !foundComp {
		t.Error("expected log message 'probe completed' not found")
	}

	// Verify the completed entry carries base URL + status per request.
	summary, ok := fieldsOf(completedEntry.Fields)["requests"].([]requestLog)
	if !ok {
		t.Fatalf("expected 'requests' field, got %v", fieldsOf(completedEntry.Fields)["requests"])
	}
	if len(summary) != 1 || summary[0].URL != baseURL(server.URL) || summary[0].Status != http.StatusOK {
		t.Errorf("unexpected request summary: %+v", summary)
	}
}

func fieldsOf(fields []logger.Field) map[string]any {
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return m
}

func TestBaseURL(t *testing.T) {
	tests := []struct {
		name, raw, want string
	}{
		{"trims path and query", "https://www.google.com/search?q=x&lang=en", "https://www.google.com"},
		{"keeps port", "http://localhost:8080/api/health", "http://localhost:8080"},
		{"no path", "https://example.com", "https://example.com"},
		{"no scheme falls back to raw", "localhost:8080", "localhost:8080"}, // parses as scheme only, host empty
		{"unparseable falls back to raw", "http://[::1", "http://[::1"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		if got := baseURL(tt.raw); got != tt.want {
			t.Errorf("%s: baseURL(%q) = %q, want %q", tt.name, tt.raw, got, tt.want)
		}
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

func TestScheduler_ImmediateFirstProbe(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Probes: []config.Probe{
			{
				ID:       "immediate-probe",
				Name:     "Immediate First Probe",
				Interval: 10, // too long for a tick to fire during the test
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

	s := New(cfg, loggertest.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	s.Stop()

	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected exactly 1 request (immediate run, zero ticks), got %d", count)
	}
}

func TestScheduler_ConcurrentMultiProbe(t *testing.T) {
	const probeCount = 3
	var counts [probeCount]int32
	servers := make([]*httptest.Server, probeCount)
	for i := range servers {
		i := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&counts[i], 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer servers[i].Close()
	}

	probes := make([]config.Probe, probeCount)
	for i := range probes {
		probes[i] = config.Probe{
			ID:       fmt.Sprintf("probe-%c", 'a'+i),
			Name:     fmt.Sprintf("Concurrent Probe %d", i),
			Interval: 1,
			Spec: &config.HTTPSpec{
				Requests: []config.Request{
					{
						URL:     servers[i].URL,
						Method:  "GET",
						Timeout: 1000,
					},
				},
			},
		}
	}

	s := New(&config.Config{Probes: probes}, loggertest.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	time.Sleep(2200 * time.Millisecond)
	s.Stop()

	for i := range counts {
		count := atomic.LoadInt32(&counts[i])
		if count < 1 {
			t.Errorf("probe %d: expected at least 1 request, got %d", i, count)
		}
	}
}

func TestScheduler_StateTransitionIntegration(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requestCount, 1) <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	assertionExpr, err := assertion.Parse("response.status == 200")
	if err != nil {
		t.Fatalf("failed to parse assertion: %v", err)
	}

	probe := config.Probe{
		ID:                "transition-probe",
		Name:              "State Transition Probe",
		Interval:          1,
		IncidentThreshold: 3,
		RecoveryThreshold: 2,
		Spec: &config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:     server.URL,
					Method:  "GET",
					Timeout: 1000,
					Alerts: []config.Alert{
						{
							Assertion: assertionExpr,
							Message:   "status bad",
						},
					},
				},
			},
		},
	}

	s := New(&config.Config{Probes: []config.Probe{probe}}, loggertest.NopLogger{})

	// PublishTransition dispatches listeners via goroutines, so collect events
	// on a buffered channel and receive with timeouts.
	eventCh := make(chan alert.TransitionEvent, 8)
	s.AlertManager().RegisterListener(func(event alert.TransitionEvent) {
		eventCh <- event
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Drive the pipeline synchronously: 3 failures (500) then 2 successes (200).
	for i := 0; i < 5; i++ {
		s.executeProbe(ctx, probe)
	}

	readEvent := func() alert.TransitionEvent {
		select {
		case ev := <-eventCh:
			return ev
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for transition event")
			return alert.TransitionEvent{}
		}
	}

	events := []alert.TransitionEvent{readEvent(), readEvent()}

	// PublishTransition spawns one goroutine per event with no ordering
	// guarantee between two publishes, so assert the pair as a set.
	sawIncident, sawRecovery := false, false
	for _, ev := range events {
		if ev.ProbeID != "transition-probe" {
			t.Errorf("unexpected probe ID: %s", ev.ProbeID)
		}
		if ev.FromState == alert.StateHealthy && ev.ToState == alert.StateIncident {
			sawIncident = true
		} else if ev.FromState == alert.StateIncident && ev.ToState == alert.StateHealthy {
			sawRecovery = true
		} else {
			t.Errorf("unexpected transition: %s -> %s", ev.FromState, ev.ToState)
		}
	}
	if !sawIncident {
		t.Error("expected a HEALTHY -> INCIDENT transition event after 3 consecutive failures")
	}
	if !sawRecovery {
		t.Error("expected an INCIDENT -> HEALTHY transition event after 2 consecutive successes")
	}

	// Exactly 2 events must have been published.
	select {
	case ev := <-eventCh:
		t.Errorf("unexpected extra transition event: %s -> %s", ev.FromState, ev.ToState)
	case <-time.After(150 * time.Millisecond):
	}

	if state := s.Tracker().GetState("transition-probe"); state.CurrentState != alert.StateHealthy {
		t.Errorf("expected final state HEALTHY, got %s", state.CurrentState)
	}
}

func TestScheduler_ProbeLevelAlert_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	probe := config.Probe{
		ID:                "probe-level-fail",
		Name:              "Probe Level Alert Failure",
		Interval:          1,
		IncidentThreshold: 3,
		RecoveryThreshold: 2,
		Alerts: []config.Alert{
			// Probe-level alert applies to every request and fails against the 200 server.
			{Assertion: assertion.MustParse("response.status == 500"), Message: "probe-level failure"},
		},
		Spec: &config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:     server.URL,
					Method:  "GET",
					Timeout: 1000,
					Alerts: []config.Alert{
						// Passing request-level alert: proves the failing probe-level
						// alert alone drives the failure (AND semantics).
						{Assertion: assertion.MustParse("response.status == 200"), Message: "request-level pass"},
					},
				},
			},
		},
	}

	s := New(&config.Config{Probes: []config.Probe{probe}}, loggertest.NopLogger{})

	eventCh := make(chan alert.TransitionEvent, 8)
	s.AlertManager().RegisterListener(func(event alert.TransitionEvent) {
		eventCh <- event
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 3; i++ {
		s.executeProbe(ctx, probe)
	}

	select {
	case ev := <-eventCh:
		if ev.ProbeID != "probe-level-fail" {
			t.Errorf("unexpected probe ID: %s", ev.ProbeID)
		}
		if ev.FromState != alert.StateHealthy || ev.ToState != alert.StateIncident {
			t.Errorf("expected HEALTHY -> INCIDENT, got %s -> %s", ev.FromState, ev.ToState)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HEALTHY -> INCIDENT transition event")
	}

	select {
	case ev := <-eventCh:
		t.Errorf("unexpected extra transition event: %s -> %s", ev.FromState, ev.ToState)
	case <-time.After(150 * time.Millisecond):
	}

	if state := s.Tracker().GetState("probe-level-fail"); state.CurrentState != alert.StateIncident {
		t.Errorf("expected final state INCIDENT, got %s", state.CurrentState)
	}
}

func TestScheduler_ProbeLevelAlert_Pass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	probe := config.Probe{
		ID:                "probe-level-pass",
		Name:              "Probe Level Alert Pass",
		Interval:          1,
		IncidentThreshold: 3,
		RecoveryThreshold: 2,
		Alerts: []config.Alert{
			{Assertion: assertion.MustParse("response.status == 200"), Message: "all good"},
		},
		Spec: &config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:     server.URL,
					Method:  "GET",
					Timeout: 1000,
				},
			},
		},
	}

	s := New(&config.Config{Probes: []config.Probe{probe}}, loggertest.NopLogger{})

	eventCh := make(chan alert.TransitionEvent, 8)
	s.AlertManager().RegisterListener(func(event alert.TransitionEvent) {
		eventCh <- event
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 3; i++ {
		s.executeProbe(ctx, probe)
	}

	select {
	case ev := <-eventCh:
		t.Errorf("unexpected transition event: %s -> %s", ev.FromState, ev.ToState)
	case <-time.After(150 * time.Millisecond):
	}

	if state := s.Tracker().GetState("probe-level-pass"); state.CurrentState != alert.StateHealthy {
		t.Errorf("expected final state HEALTHY, got %s", state.CurrentState)
	}
}

type unsupportedDummySpec struct{}

func (s *unsupportedDummySpec) Kind() config.ProbeKind { return "unsupported-dummy" }
func (s *unsupportedDummySpec) Validate() error        { return nil }

func TestScheduler_RepeatExits(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Repeat: 2,
		Probes: []config.Probe{
			{ID: "repeat-test", Interval: 1, Spec: &config.HTTPSpec{Requests: []config.Request{{URL: server.URL}}}},
		},
	}
	s := New(cfg, loggertest.NopLogger{})

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	done := make(chan struct{})
	go func() {
		s.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not exit after reaching repeat count")
	}

	if got := atomic.LoadInt32(&requestCount); got != 2 {
		t.Errorf("expected exactly 2 probe runs, got %d", got)
	}
	s.Stop()
}
