package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func captureOutput(fn func()) string {
	var buf bytes.Buffer
	sink := getSink()
	oldOut := sink.Out
	sink.SetOutput(&buf)
	defer func() { sink.SetOutput(oldOut) }()
	fn()
	return buf.String()
}

func parseJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, raw)
	}
	return m
}

// --- Field constructors ------------------------------------------------------

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		key      string
		checkVal func(any) bool
	}{
		{"F", F("custom", 42), "custom", func(v any) bool { return v == 42 }},
		{"Component", Component("http"), "component", func(v any) bool { return v == "http" }},
		{"TraceID", TraceID("abc-123"), "trace_id", func(v any) bool { return v == "abc-123" }},
		{"DurationMS", DurationMS(12.5), "duration_ms", func(v any) bool { return v == 12.5 }},
		{"Err", Err(errors.New("boom")), "error", func(v any) bool {
			e, ok := v.(error)
			return ok && e.Error() == "boom"
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.key {
				t.Errorf("Key = %q, want %q", tt.field.Key, tt.key)
			}
			if !tt.checkVal(tt.field.Value) {
				t.Errorf("Value = %v (type %T), unexpected", tt.field.Value, tt.field.Value)
			}
		})
	}
}

// --- Interface compliance ----------------------------------------------------

func TestNew_ReturnsValidLogger(t *testing.T) {
	log := New("test-component")
	if log == nil {
		t.Fatal("New returned nil")
	}
	// All methods should be callable without panic
	log.Info("info msg")
	log.Warn("warn msg")
	log.Error("error msg")
	log.Debug("debug msg")
}

func TestLogger_With_ReturnsNewInstance(t *testing.T) {
	log1 := New("comp-a")
	log2 := log1.With(F("extra", "value"))

	if log1 == log2 {
		t.Error("With should return a new Logger instance")
	}
}

func TestLogger_With_Chaining(t *testing.T) {
	log := New("chain-test")
	chained := log.With(F("a", 1)).With(F("b", 2))
	if chained == nil {
		t.Fatal("chained logger is nil")
	}
	chained.Info("chained message")
}

// --- Structured output -------------------------------------------------------

func TestLogger_Info_StructuredOutput(t *testing.T) {
	output := captureOutput(func() {
		log := New("http")
		log.Info("request completed", F("status", 200), DurationMS(10.5))
	})

	m := parseJSON(t, output)

	if m["component"] != "http" {
		t.Errorf("component = %v, want http", m["component"])
	}
	if m["level"] != "info" {
		t.Errorf("level = %v, want info", m["level"])
	}
	if m["msg"] != "request completed" {
		t.Errorf("msg = %v, want request completed", m["msg"])
	}
	if m["status"] != float64(200) {
		t.Errorf("status = %v, want 200", m["status"])
	}
	if m["duration_ms"] != 10.5 {
		t.Errorf("duration_ms = %v, want 10.5", m["duration_ms"])
	}
}

func TestLogger_Warn_StructuredOutput(t *testing.T) {
	output := captureOutput(func() {
		log := New("scheduler")
		log.Warn("retry exhausted", F("attempts", 3))
	})

	m := parseJSON(t, output)
	if m["level"] != "warning" {
		t.Errorf("level = %v, want warning", m["level"])
	}
}

func TestLogger_Error_StructuredOutput(t *testing.T) {
	output := captureOutput(func() {
		log := New("prober")
		log.Error("connection failed", Err(errors.New("timeout")), F("host", "example.com"))
	})

	m := parseJSON(t, output)
	if m["level"] != "error" {
		t.Errorf("level = %v, want error", m["level"])
	}
	if m["host"] != "example.com" {
		t.Errorf("host = %v, want example.com", m["host"])
	}
	if m["error"] != "timeout" {
		t.Errorf("error = %v, want timeout", m["error"])
	}
}

func TestLogger_Debug_StructuredOutput(t *testing.T) {
	output := captureOutput(func() {
		log := New("debugger")
		log.Debug("trace detail", TraceID("trace-999"))
	})

	m := parseJSON(t, output)
	if m["level"] != "debug" {
		t.Errorf("level = %v, want debug", m["level"])
	}
	if m["trace_id"] != "trace-999" {
		t.Errorf("trace_id = %v, want trace-999", m["trace_id"])
	}
}

func TestLogger_With_PreservesBaseFields(t *testing.T) {
	output := captureOutput(func() {
		log := New("base-comp")
		child := log.With(DurationMS(5.0))
		child.Info("child message", F("payload", "hello"))
	})

	m := parseJSON(t, output)
	if m["component"] != "base-comp" {
		t.Errorf("component = %v, want base-comp", m["component"])
	}
	if m["duration_ms"] != 5.0 {
		t.Errorf("duration_ms = %v, want 5.0", m["duration_ms"])
	}
	if m["payload"] != "hello" {
		t.Errorf("payload = %v, want hello", m["payload"])
	}
}

// --- Zero / nil fields -------------------------------------------------------

func TestLogger_NoFields(t *testing.T) {
	output := captureOutput(func() {
		log := New("no-fields")
		log.Info("bare message")
	})

	m := parseJSON(t, output)
	if m["msg"] != "bare message" {
		t.Errorf("msg = %v, want bare message", m["msg"])
	}
	// Should not contain extra keys beyond standard logrus fields
	if m["component"] != "no-fields" {
		t.Errorf("component = %v, want no-fields", m["component"])
	}
}

func TestLogger_MultipleComponents(t *testing.T) {
	output := captureOutput(func() {
		logA := New("comp-a")
		logB := New("comp-b")
		logA.Info("from A")
		logB.Info("from B")
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d", len(lines))
	}

	m1 := parseJSON(t, lines[0])
	m2 := parseJSON(t, lines[1])

	if m1["component"] != "comp-a" {
		t.Errorf("first line component = %v, want comp-a", m1["component"])
	}
	if m2["component"] != "comp-b" {
		t.Errorf("second line component = %v, want comp-b", m2["component"])
	}
}

// --- Error field with nil ----------------------------------------------------

func TestLogger_ErrorField_Nil(t *testing.T) {
	output := captureOutput(func() {
		log := New("nil-err-test")
		log.Error("no error", Err(nil))
	})

	m := parseJSON(t, output)
	if m["level"] != "error" {
		t.Errorf("level = %v, want error", m["level"])
	}
	// Nil error should not blow up; logrus renders it as <nil>
}

// --- InitLogger idempotency --------------------------------------------------

func TestInitLogger_Idempotent(t *testing.T) {
	InitLogger()
	InitLogger() // should not panic or create duplicate
	output := captureOutput(func() {
		New("idempotent").Info("still works")
	})
	if output == "" {
		t.Error("expected non-empty output after InitLogger called twice")
	}
}

// --- toLogrusFields helper ---------------------------------------------------

func TestToLogrusFields_Empty(t *testing.T) {
	f := toLogrusFields(nil)
	if f != nil {
		t.Errorf("expected nil for empty fields, got %v", f)
	}

	f = toLogrusFields([]Field{})
	if f != nil {
		t.Errorf("expected nil for empty fields slice, got %v", f)
	}
}

// --- JSON formatter output shape ---------------------------------------------

func TestLogger_OutputHasTimestamp(t *testing.T) {
	output := captureOutput(func() {
		New("ts-test").Info("check timestamp")
	})

	m := parseJSON(t, output)
	if _, ok := m["time"]; !ok {
		t.Error("expected time field in JSON output")
	}
}

// --- Level compliance --------------------------------------------------------

func TestLogger_Levels(t *testing.T) {
	// All levels should produce output since sink is set to DebugLevel.
	expected := map[string]string{
		"info":  "info",
		"warn":  "warning",
		"error": "error",
		"debug": "debug",
	}

	for _, level := range []struct {
		fn   func(Logger)
		name string
	}{
		{func(l Logger) { l.Info("msg") }, "info"},
		{func(l Logger) { l.Warn("msg") }, "warn"},
		{func(l Logger) { l.Error("msg") }, "error"},
		{func(l Logger) { l.Debug("msg") }, "debug"},
	} {
		t.Run(level.name, func(t *testing.T) {
			output := captureOutput(func() {
				level.fn(New("level-test"))
			})
			m := parseJSON(t, output)
			if m["level"] != expected[level.name] {
				t.Errorf("level = %v, want %v", m["level"], expected[level.name])
			}
		})
	}
}
