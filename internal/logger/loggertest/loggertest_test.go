package loggertest

import (
	"testing"

	"monika-go/internal/logger"
)

func TestNopLogger(t *testing.T) {
	var l logger.Logger = NopLogger{}
	l.Info("i")
	l.Warn("w")
	l.Error("e")
	l.Debug("d")
	if child := l.With(logger.F("k", "v")); child == nil {
		t.Error("With() returned nil")
	}
}

func TestCaptureLogger(t *testing.T) {
	c := &CaptureLogger{}
	c.Info("info-msg", logger.F("k", "v"))

	child := c.With(logger.F("bound", 1))
	child.Warn("warn-msg")

	entries := c.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Level != "INFO" || entries[0].Msg != "info-msg" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if len(entries[0].Fields) != 1 || entries[0].Fields[0].Key != "k" {
		t.Errorf("unexpected first entry fields: %+v", entries[0].Fields)
	}
	if entries[1].Level != "WARN" || entries[1].Msg != "warn-msg" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
	// Child entries must carry the fields bound via With().
	if len(entries[1].Fields) != 1 || entries[1].Fields[0].Key != "bound" {
		t.Errorf("expected bound field on child entry, got %+v", entries[1].Fields)
	}
}
