// Package logger provides an injectable structured logging interface.
// Only this package imports logrus — all other packages use the Logger interface.
package logger

import (
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Field is a structured key-value pair attached to a log entry.
type Field struct {
	Key   string
	Value any
}

// Typed constructors for the fields required by AGENTS.md.
func F(key string, value any) Field  { return Field{key, value} }
func Component(v string) Field       { return Field{"component", v} }
func TraceID(v string) Field         { return Field{"trace_id", v} }
func DurationMS(v float64) Field     { return Field{"duration_ms", v} }
func Err(err error) Field            { return Field{"error", err} }

// Logger is the injectable structured logging interface.
type Logger interface {
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	With(fields ...Field) Logger
}

// --- global logrus sink ------------------------------------------------------

var (
	sinkOnce sync.Once
	sink     *logrus.Logger
)

func getSink() *logrus.Logger {
	sinkOnce.Do(func() {
		sink = logrus.New()
		sink.SetLevel(logrus.DebugLevel)
		sink.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			PrettyPrint:     false,
		})
		sink.SetOutput(os.Stdout)
	})
	return sink
}

// InitLogger initialises the global logrus sink. Safe to call multiple times.
func InitLogger() { getSink() }

// New returns a Logger pre-bound with the given component name.
func New(component string) Logger {
	return &logrusLogger{
		entry: getSink().WithField("component", component),
	}
}

// --- logrus adapter ----------------------------------------------------------

type logrusLogger struct {
	entry *logrus.Entry
}

func toLogrusFields(fields []Field) logrus.Fields {
	if len(fields) == 0 {
		return nil
	}
	f := make(logrus.Fields, len(fields))
	for _, kv := range fields {
		f[kv.Key] = kv.Value
	}
	return f
}

func (l *logrusLogger) Info(msg string, fields ...Field) {
	l.entry.WithFields(toLogrusFields(fields)).Info(msg)
}
func (l *logrusLogger) Warn(msg string, fields ...Field) {
	l.entry.WithFields(toLogrusFields(fields)).Warn(msg)
}
func (l *logrusLogger) Error(msg string, fields ...Field) {
	l.entry.WithFields(toLogrusFields(fields)).Error(msg)
}
func (l *logrusLogger) Debug(msg string, fields ...Field) {
	l.entry.WithFields(toLogrusFields(fields)).Debug(msg)
}
func (l *logrusLogger) With(fields ...Field) Logger {
	return &logrusLogger{entry: l.entry.WithFields(toLogrusFields(fields))}
}
