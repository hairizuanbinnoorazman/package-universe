package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

// LogrusLogger wraps a logrus logger to implement the Logger interface.
type LogrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrusLogger creates a new LogrusLogger with JSON formatter.
func NewLogrusLogger(level string) *LogrusLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Parse and set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	return &LogrusLogger{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

// Debug logs a debug-level message.
func (l *LogrusLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	if fields != nil {
		l.entry.WithFields(fields).Debug(msg)
	} else {
		l.entry.Debug(msg)
	}
}

// Info logs an info-level message.
func (l *LogrusLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	if fields != nil {
		l.entry.WithFields(fields).Info(msg)
	} else {
		l.entry.Info(msg)
	}
}

// Warn logs a warning-level message.
func (l *LogrusLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	if fields != nil {
		l.entry.WithFields(fields).Warn(msg)
	} else {
		l.entry.Warn(msg)
	}
}

// Error logs an error-level message.
func (l *LogrusLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	if fields != nil {
		l.entry.WithFields(fields).Error(msg)
	} else {
		l.entry.Error(msg)
	}
}

// WithField returns a new logger with the given field added.
func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields returns a new logger with the given fields added.
func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}
