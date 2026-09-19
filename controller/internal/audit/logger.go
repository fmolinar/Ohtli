// Package audit emits the structured JSON audit/log events described in
// README.md section 8: timestamp, device, path, event type, correlation
// ID, old/new value, operation, outcome, and duration. It never logs
// credentials or private-key material -- callers must not pass them in
// OldValue/NewValue.
package audit

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// EventType categorizes an audit event for filtering/alerting.
type EventType string

const (
	EventCapabilities EventType = "capabilities"
	EventGet          EventType = "get"
	EventSubscribe    EventType = "subscribe"
	EventUpdate       EventType = "update" // a single streamed telemetry update
	EventError        EventType = "error"
)

// Outcome is the result of the operation the event describes.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
)

// Event is one structured audit record.
type Event struct {
	Device        string
	Path          string
	EventType     EventType
	CorrelationID string
	OldValue      interface{}
	NewValue      interface{}
	Operation     string
	Outcome       Outcome
	Duration      time.Duration
}

// Logger emits Events as JSON to an underlying slog.Logger (stdout by
// default).
type Logger struct {
	slog *slog.Logger
}

// NewLogger returns a Logger writing JSON lines to stdout.
func NewLogger() *Logger {
	return &Logger{slog: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
}

// Emit logs a single audit event.
func (l *Logger) Emit(ctx context.Context, e Event) {
	l.slog.LogAttrs(ctx, slog.LevelInfo, "audit_event",
		slog.Time("timestamp", time.Now().UTC()),
		slog.String("device", e.Device),
		slog.String("path", e.Path),
		slog.String("event_type", string(e.EventType)),
		slog.String("correlation_id", e.CorrelationID),
		slog.Any("old_value", e.OldValue),
		slog.Any("new_value", e.NewValue),
		slog.String("operation", e.Operation),
		slog.String("outcome", string(e.Outcome)),
		slog.Duration("duration", e.Duration),
	)
}
