package alert

import (
	"fmt"
	"sync"
)

// State represents the high-level health condition of a monitored target.
type State string

const (
	StateHealthy  State = "HEALTHY"
	StateIncident State = "INCIDENT"
)

// ProbeState tracks consecutive successes/failures and status of a single probe.
type ProbeState struct {
	ProbeID              string
	CurrentState         State
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
}

// Tracker records probe execution outcomes and manages state transitions.
type Tracker interface {
	// RecordResult logs a success/failure tick and determines if a state transition occurred.
	RecordResult(probeID string, success bool, incidentThreshold, recoveryThreshold int) (*TransitionEvent, error)
	// GetState returns a copy of the current state for a given probe.
	GetState(probeID string) ProbeState
}

// TransitionEvent details status changes between Healthy and Incident states.
type TransitionEvent struct {
	ProbeID   string
	FromState State
	ToState   State
	Message   string
}

type memTracker struct {
	mu     sync.RWMutex
	states map[string]*ProbeState
}

// NewTracker constructs a thread-safe in-memory state Tracker.
func NewTracker() Tracker {
	return &memTracker{
		states: make(map[string]*ProbeState),
	}
}

// RecordResult updates consecutive counters and evaluates transitions. Thread-safe.
func (t *memTracker) RecordResult(probeID string, success bool, incidentThreshold, recoveryThreshold int) (*TransitionEvent, error) {
	if probeID == "" {
		return nil, fmt.Errorf("probeID cannot be empty")
	}
	if incidentThreshold <= 0 {
		incidentThreshold = 5 // default fallback threshold
	}
	if recoveryThreshold <= 0 {
		recoveryThreshold = 5 // default fallback threshold
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.states[probeID]
	if !exists {
		state = &ProbeState{
			ProbeID:      probeID,
			CurrentState: StateHealthy,
		}
		t.states[probeID] = state
	}

	var event *TransitionEvent

	if success {
		state.ConsecutiveFailures = 0

		if state.CurrentState == StateIncident {
			state.ConsecutiveSuccesses++
			if state.ConsecutiveSuccesses >= recoveryThreshold {
				event = &TransitionEvent{
					ProbeID:   probeID,
					FromState: StateIncident,
					ToState:   StateHealthy,
					Message:   fmt.Sprintf("Probe %s recovered after %d consecutive successes", probeID, state.ConsecutiveSuccesses),
				}
				state.CurrentState = StateHealthy
				state.ConsecutiveSuccesses = 0
			}
		} else {
			state.ConsecutiveSuccesses = 0
		}
	} else {
		state.ConsecutiveSuccesses = 0

		if state.CurrentState == StateHealthy {
			state.ConsecutiveFailures++
			if state.ConsecutiveFailures >= incidentThreshold {
				event = &TransitionEvent{
					ProbeID:   probeID,
					FromState: StateHealthy,
					ToState:   StateIncident,
					Message:   fmt.Sprintf("Probe %s entered incident state after %d consecutive failures", probeID, state.ConsecutiveFailures),
				}
				state.CurrentState = StateIncident
				state.ConsecutiveFailures = 0
			}
		} else {
			state.ConsecutiveFailures = 0
		}
	}

	return event, nil
}

// GetState returns a snapshot of the current state. Thread-safe.
func (t *memTracker) GetState(probeID string) ProbeState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, exists := t.states[probeID]
	if !exists {
		return ProbeState{
			ProbeID:      probeID,
			CurrentState: StateHealthy,
		}
	}
	return *state
}
