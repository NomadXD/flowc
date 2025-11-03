package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

// EnvoyLogger implements the log.Logger interface from Envoy Go control plane
type EnvoyLogger struct {
	logger *slog.Logger
	level  Level
}

// Level represents the logging level
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// NewEnvoyLogger creates a new Envoy logger
func NewEnvoyLogger(level Level) *EnvoyLogger {
	// Create a structured logger with JSON output
	opts := &slog.HandlerOptions{
		Level:     slog.Level(level),
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the source attribute to show file:line
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					// Extract just the filename from the full path
					parts := strings.Split(source.File, "/")
					filename := parts[len(parts)-1]
					return slog.String("source", fmt.Sprintf("%s:%d", filename, source.Line))
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &EnvoyLogger{
		logger: logger,
		level:  level,
	}
}

// NewEnvoyLoggerWithHandler creates a new Envoy logger with a custom handler
func NewEnvoyLoggerWithHandler(handler slog.Handler) *EnvoyLogger {
	logger := slog.New(handler)
	return &EnvoyLogger{
		logger: logger,
		level:  InfoLevel, // Default level
	}
}

// Debug logs a debug message
func (l *EnvoyLogger) Debug(msg string) {
	l.logger.Debug(msg)
}

// Debugf logs a debug message with formatting
func (l *EnvoyLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

// Info logs an info message
func (l *EnvoyLogger) Info(msg string) {
	l.logger.Info(msg)
}

// Infof logs an info message with formatting
func (l *EnvoyLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *EnvoyLogger) Warn(msg string) {
	l.logger.Warn(msg)
}

// Warnf logs a warning message with formatting
func (l *EnvoyLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *EnvoyLogger) Error(msg string) {
	l.logger.Error(msg)
}

// Errorf logs an error message with formatting
func (l *EnvoyLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits
func (l *EnvoyLogger) Fatal(msg string) {
	l.logger.Error(msg, "level", "FATAL")
	os.Exit(1)
}

// Fatalf logs a fatal message with formatting and exits
func (l *EnvoyLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...), "level", "FATAL")
	os.Exit(1)
}

// WithField adds a field to the logger
func (l *EnvoyLogger) WithField(key string, value interface{}) *EnvoyLogger {
	return &EnvoyLogger{
		logger: l.logger.With(key, value),
		level:  l.level,
	}
}

// WithFields adds multiple fields to the logger
func (l *EnvoyLogger) WithFields(fields map[string]interface{}) *EnvoyLogger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &EnvoyLogger{
		logger: l.logger.With(args...),
		level:  l.level,
	}
}

// WithError adds an error field to the logger
func (l *EnvoyLogger) WithError(err error) *EnvoyLogger {
	return l.WithField("error", err.Error())
}

// WithContext adds context to the logger
func (l *EnvoyLogger) WithContext(ctx context.Context) *EnvoyLogger {
	// For now, just return the same logger
	// In a more sophisticated implementation, you might extract values from context
	return l
}

// SetLevel sets the logging level
func (l *EnvoyLogger) SetLevel(level Level) {
	l.level = level
	// Note: slog doesn't support changing level at runtime easily
	// This would require recreating the logger
}

// GetLevel returns the current logging level
func (l *EnvoyLogger) GetLevel() Level {
	return l.level
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *EnvoyLogger) IsDebugEnabled() bool {
	return l.level <= DebugLevel
}

// IsInfoEnabled returns true if info logging is enabled
func (l *EnvoyLogger) IsInfoEnabled() bool {
	return l.level <= InfoLevel
}

// IsWarnEnabled returns true if warning logging is enabled
func (l *EnvoyLogger) IsWarnEnabled() bool {
	return l.level <= WarnLevel
}

// IsErrorEnabled returns true if error logging is enabled
func (l *EnvoyLogger) IsErrorEnabled() bool {
	return l.level <= ErrorLevel
}

// Helper function to get caller information
func getCaller() (string, int) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown", 0
	}
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]
	return filename, line
}

// NewDefaultEnvoyLogger creates a default Envoy logger with INFO level
func NewDefaultEnvoyLogger() *EnvoyLogger {
	return NewEnvoyLogger(InfoLevel)
}

// NewDebugEnvoyLogger creates a debug Envoy logger
func NewDebugEnvoyLogger() *EnvoyLogger {
	return NewEnvoyLogger(DebugLevel)
}
