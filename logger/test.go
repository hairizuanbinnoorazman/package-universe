package logger

import (
	"context"
	"sync"
)

// LogEntry represents a single log entry captured by the test logger.
type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

// TestLogger is a logger implementation for testing that captures log entries.
type TestLogger struct {
	mu      sync.RWMutex
	entries []LogEntry
	fields  map[string]interface{}
}

// NewTestLogger creates a new test logger.
func NewTestLogger() *TestLogger {
	return &TestLogger{
		entries: make([]LogEntry, 0),
		fields:  make(map[string]interface{}),
	}
}

// Debug logs a debug-level message.
func (l *TestLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log("debug", msg, fields)
}

// Info logs an info-level message.
func (l *TestLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log("info", msg, fields)
}

// Warn logs a warning-level message.
func (l *TestLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log("warn", msg, fields)
}

// Error logs an error-level message.
func (l *TestLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log("error", msg, fields)
}

// WithField returns a new logger with the given field added.
func (l *TestLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &TestLogger{
		entries: l.entries,
		fields:  newFields,
	}
}

// WithFields returns a new logger with the given fields added.
func (l *TestLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &TestLogger{
		entries: l.entries,
		fields:  newFields,
	}
}

// log adds a log entry to the captured entries.
func (l *TestLogger) log(level, msg string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Merge logger fields with call fields
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	if fields != nil {
		for k, v := range fields {
			allFields[k] = v
		}
	}

	l.entries = append(l.entries, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  allFields,
	})
}

// Entries returns all captured log entries.
func (l *TestLogger) Entries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to prevent external modification
	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// Reset clears all captured log entries.
func (l *TestLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LogEntry, 0)
}
