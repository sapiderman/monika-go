package alert

import (
	"sync"

	"monika-go/internal/logger"
)

// Listener defines the callback type executed upon probe state transitions.
type Listener func(event TransitionEvent)

// AlertManager handles subscription and distribution of state transition events.
type AlertManager interface {
	// RegisterListener registers a handler for state transitions.
	RegisterListener(listener Listener)
	// PublishTransition distributes a state transition event to all subscribers.
	PublishTransition(event TransitionEvent)
}

type memAlertManager struct {
	log       logger.Logger
	mu        sync.RWMutex
	listeners []Listener
}

// NewAlertManager constructs a new AlertManager.
func NewAlertManager(log logger.Logger) AlertManager {
	return &memAlertManager{
		log:       log.With(logger.Component("alert-manager")),
		listeners: make([]Listener, 0),
	}
}

// RegisterListener appends a subscriber to the registry. Thread-safe.
func (m *memAlertManager) RegisterListener(listener Listener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// PublishTransition logs transition state changes and broadcasts them concurrently to subscribers. Thread-safe.
func (m *memAlertManager) PublishTransition(event TransitionEvent) {
	m.log.Info("probe state transitioned",
		logger.F("probe_id", event.ProbeID),
		logger.F("from_state", string(event.FromState)),
		logger.F("to_state", string(event.ToState)),
		logger.F("message", event.Message),
	)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, listener := range m.listeners {
		go listener(event)
	}
}
