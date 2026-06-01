package alert

import (
	"sync"
	"testing"

	"monika-go/internal/logger/loggertest"
)

func TestTracker_Transitions(t *testing.T) {
	t.Run("Standard Thresholds (5 failures to incident, 5 successes to recovery)", func(t *testing.T) {
		tracker := NewTracker()
		probeID := "p1"

		// 1. Send 4 failures -> State should remain HEALTHY, no event
		for i := 0; i < 4; i++ {
			event, err := tracker.RecordResult(probeID, false, 5, 5)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event != nil {
				t.Errorf("expected no transition event, got %v", event)
			}
			state := tracker.GetState(probeID)
			if state.CurrentState != StateHealthy {
				t.Errorf("expected state HEALTHY, got %v", state.CurrentState)
			}
			if state.ConsecutiveFailures != i+1 {
				t.Errorf("expected consecutive failures = %d, got %d", i+1, state.ConsecutiveFailures)
			}
		}

		// 2. 5th failure -> Transitions HEALTHY -> INCIDENT
		event, err := tracker.RecordResult(probeID, false, 5, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event == nil {
			t.Fatal("expected transition event, got nil")
		}
		if event.FromState != StateHealthy || event.ToState != StateIncident {
			t.Errorf("expected transition HEALTHY -> INCIDENT, got %s -> %s", event.FromState, event.ToState)
		}

		state := tracker.GetState(probeID)
		if state.CurrentState != StateIncident {
			t.Errorf("expected state INCIDENT, got %v", state.CurrentState)
		}
		if state.ConsecutiveFailures != 0 {
			t.Errorf("expected consecutive failures reset to 0, got %d", state.ConsecutiveFailures)
		}

		// 3. Send 4 successes -> State should remain INCIDENT, no event
		for i := 0; i < 4; i++ {
			event, err := tracker.RecordResult(probeID, true, 5, 5)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event != nil {
				t.Errorf("expected no transition event, got %v", event)
			}
			state := tracker.GetState(probeID)
			if state.CurrentState != StateIncident {
				t.Errorf("expected state INCIDENT, got %v", state.CurrentState)
			}
			if state.ConsecutiveSuccesses != i+1 {
				t.Errorf("expected consecutive successes = %d, got %d", i+1, state.ConsecutiveSuccesses)
			}
		}

		// 4. 5th success -> Transitions INCIDENT -> HEALTHY
		event, err = tracker.RecordResult(probeID, true, 5, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event == nil {
			t.Fatal("expected transition event, got nil")
		}
		if event.FromState != StateIncident || event.ToState != StateHealthy {
			t.Errorf("expected transition INCIDENT -> HEALTHY, got %s -> %s", event.FromState, event.ToState)
		}

		state = tracker.GetState(probeID)
		if state.CurrentState != StateHealthy {
			t.Errorf("expected state HEALTHY, got %v", state.CurrentState)
		}
		if state.ConsecutiveSuccesses != 0 {
			t.Errorf("expected consecutive successes reset to 0, got %d", state.ConsecutiveSuccesses)
		}
	})

	t.Run("Custom Thresholds (2 failures to incident, 3 successes to recovery)", func(t *testing.T) {
		tracker := NewTracker()
		probeID := "p2"

		// 1st failure
		event, _ := tracker.RecordResult(probeID, false, 2, 3)
		if event != nil {
			t.Error("expected no transition")
		}

		// 2nd failure -> HEALTHY -> INCIDENT
		event, _ = tracker.RecordResult(probeID, false, 2, 3)
		if event == nil || event.ToState != StateIncident {
			t.Error("expected transition to INCIDENT")
		}

		// 1st success
		event, _ = tracker.RecordResult(probeID, true, 2, 3)
		if event != nil {
			t.Error("expected no transition")
		}

		// 2nd success
		event, _ = tracker.RecordResult(probeID, true, 2, 3)
		if event != nil {
			t.Error("expected no transition")
		}

		// 3rd success -> INCIDENT -> HEALTHY
		event, _ = tracker.RecordResult(probeID, true, 2, 3)
		if event == nil || event.ToState != StateHealthy {
			t.Error("expected transition to HEALTHY")
		}
	})
}

func TestTracker_ThreadSafety(t *testing.T) {
	tracker := NewTracker()
	probeID := "concurrent-p"

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			success := workerID%2 == 0
			for j := 0; j < iterations; j++ {
				_, _ = tracker.RecordResult(probeID, success, 5, 5)
				_ = tracker.GetState(probeID)
			}
		}(i)
	}

	wg.Wait()
}

func TestAlertManager_PublishSubscribe(t *testing.T) {
	nopLog := loggertest.NopLogger{}
	manager := NewAlertManager(nopLog)

	var wg sync.WaitGroup
	var receivedEvent TransitionEvent
	wg.Add(1)

	manager.RegisterListener(func(event TransitionEvent) {
		receivedEvent = event
		wg.Done()
	})

	sentEvent := TransitionEvent{
		ProbeID:   "p-test",
		FromState: StateHealthy,
		ToState:   StateIncident,
		Message:   "Entering incident",
	}

	manager.PublishTransition(sentEvent)

	// Wait for listener callback execution
	wg.Wait()

	if receivedEvent.ProbeID != sentEvent.ProbeID {
		t.Errorf("expected probe ID %s, got %s", sentEvent.ProbeID, receivedEvent.ProbeID)
	}
	if receivedEvent.FromState != sentEvent.FromState || receivedEvent.ToState != sentEvent.ToState {
		t.Errorf("expected state transition %s -> %s, got %s -> %s",
			sentEvent.FromState, sentEvent.ToState, receivedEvent.FromState, receivedEvent.ToState)
	}
	if receivedEvent.Message != sentEvent.Message {
		t.Errorf("expected message %s, got %s", sentEvent.Message, receivedEvent.Message)
	}
}
