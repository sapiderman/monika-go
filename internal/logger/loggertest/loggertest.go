// Package loggertest provides test implementations of logger.Logger.
// Use NopLogger for benchmarks or tests that don't assert logs.
// Use CaptureLogger for tests that need to inspect recorded log entries.
package loggertest

import (
	"sync"

	"monika-go/internal/logger"
)

// NopLogger discards all log output.
type NopLogger struct{}

func (NopLogger) Info(string, ...logger.Field)  {}
func (NopLogger) Warn(string, ...logger.Field)  {}
func (NopLogger) Error(string, ...logger.Field) {}
func (NopLogger) Debug(string, ...logger.Field) {}
func (NopLogger) With(...logger.Field) logger.Logger { return NopLogger{} }

// CaptureEntry is a single recorded log entry.
type CaptureEntry struct {
	Level  string
	Msg    string
	Fields []logger.Field
}

type captureStore struct {
	mu      sync.Mutex
	entries []CaptureEntry
}

// CaptureLogger records log entries for test assertions.
// Children created via With() share the same backing store.
type CaptureLogger struct {
	store *captureStore
	bound []logger.Field // fields pre-bound via With()
}

func (c *CaptureLogger) getStore() *captureStore {
	if c.store == nil {
		c.store = &captureStore{}
	}
	return c.store
}

func (c *CaptureLogger) record(level, msg string, extra []logger.Field) {
	all := make([]logger.Field, 0, len(c.bound)+len(extra))
	all = append(all, c.bound...)
	all = append(all, extra...)
	s := c.getStore()
	s.mu.Lock()
	s.entries = append(s.entries, CaptureEntry{Level: level, Msg: msg, Fields: all})
	s.mu.Unlock()
}

func (c *CaptureLogger) Info(msg string, f ...logger.Field)  { c.record("INFO", msg, f) }
func (c *CaptureLogger) Warn(msg string, f ...logger.Field)  { c.record("WARN", msg, f) }
func (c *CaptureLogger) Error(msg string, f ...logger.Field) { c.record("ERROR", msg, f) }
func (c *CaptureLogger) Debug(msg string, f ...logger.Field) { c.record("DEBUG", msg, f) }

// With returns a child CaptureLogger that shares the same backing store.
func (c *CaptureLogger) With(fields ...logger.Field) logger.Logger {
	return &CaptureLogger{
		store: c.getStore(),
		bound: append(append([]logger.Field{}, c.bound...), fields...),
	}
}

// Entries returns a snapshot of all recorded log entries.
func (c *CaptureLogger) Entries() []CaptureEntry {
	s := c.getStore()
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]CaptureEntry(nil), s.entries...)
}
