package logger

import "context"

// Logger defines the interface for structured logging with context support.
type Logger interface {
	// Debug logs a debug-level message with optional fields
	Debug(ctx context.Context, msg string, fields map[string]interface{})

	// Info logs an info-level message with optional fields
	Info(ctx context.Context, msg string, fields map[string]interface{})

	// Warn logs a warning-level message with optional fields
	Warn(ctx context.Context, msg string, fields map[string]interface{})

	// Error logs an error-level message with optional fields
	Error(ctx context.Context, msg string, fields map[string]interface{})

	// WithField returns a new logger with the given field added to all subsequent log entries
	WithField(key string, value interface{}) Logger

	// WithFields returns a new logger with the given fields added to all subsequent log entries
	WithFields(fields map[string]interface{}) Logger
}
