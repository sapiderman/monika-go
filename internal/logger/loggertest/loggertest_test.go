package loggertest

import (
	"testing"

	"monika-go/internal/logger"
)

// compile-time checks: both types satisfy logger.Logger
var _ logger.Logger = NopLogger{}
var _ logger.Logger = (*CaptureLogger)(nil)

func TestCaptureLogger_RecordsEntry(t *testing.T) {
	cap := &CaptureLogger{}
	cap.Info("hello")

	entries := cap.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Msg != "hello" {
		t.Errorf("expected msg %q, got %q", "hello", entries[0].Msg)
	}
	if entries[0].Level != "INFO" {
		t.Errorf("expected level INFO, got %q", entries[0].Level)
	}
}

func TestCaptureLogger_AllLevels(t *testing.T) {
	cap := &CaptureLogger{}
	cap.Info("a")
	cap.Warn("b")
	cap.Error("c")
	cap.Debug("d")

	entries := cap.Entries()
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	for i, want := range levels {
		if entries[i].Level != want {
			t.Errorf("entry %d: expected level %q, got %q", i, want, entries[i].Level)
		}
	}
}

func TestCaptureLogger_WithBindsFields(t *testing.T) {
	cap := &CaptureLogger{}
	child := cap.With(logger.Component("prober"), logger.TraceID("abc"))
	child.Info("probe started")

	entries := cap.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	fields := fieldMap(entries[0].Fields)
	if fields["component"] != "prober" {
		t.Errorf("expected component=prober, got %v", fields["component"])
	}
	if fields["trace_id"] != "abc" {
		t.Errorf("expected trace_id=abc, got %v", fields["trace_id"])
	}
}

func TestCaptureLogger_WithSharesStore(t *testing.T) {
	cap := &CaptureLogger{}
	child := cap.With(logger.Component("child"))

	cap.Info("from root")
	child.Info("from child")

	entries := cap.Entries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries in shared store, got %d", len(entries))
	}
}

func TestCaptureLogger_WithAddsFieldsPerCall(t *testing.T) {
	cap := &CaptureLogger{}
	cap.Info("no fields")
	cap.Info("with field", logger.DurationMS(42))

	entries := cap.Entries()
	if len(entries[0].Fields) != 0 {
		t.Errorf("expected 0 fields on first entry, got %d", len(entries[0].Fields))
	}
	fields := fieldMap(entries[1].Fields)
	if fields["duration_ms"] != float64(42) {
		t.Errorf("expected duration_ms=42, got %v", fields["duration_ms"])
	}
}

func TestNopLogger_DoesNotPanic(t *testing.T) {
	var log logger.Logger = NopLogger{}
	log.Info("ignored")
	log.Warn("ignored")
	log.Error("ignored")
	log.Debug("ignored")
	log.With(logger.Component("x")).Info("ignored")
}

// fieldMap converts []logger.Field to map for easier assertion.
func fieldMap(fields []logger.Field) map[string]any {
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return m
}
