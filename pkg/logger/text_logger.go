package logger

import (
	"io"
	"log/slog"
	"os"
)

// TextEnvoyLogger creates a text-based Envoy logger
type TextEnvoyLogger struct {
	*EnvoyLogger
}

// NewTextEnvoyLogger creates a new text-based Envoy logger
func NewTextEnvoyLogger(level Level) *TextEnvoyLogger {
	// Create a text handler with custom formatting
	opts := &slog.HandlerOptions{
		Level:     slog.Level(level),
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006-01-02 15:04:05.000"))
			}
			// Customize level format
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				return slog.String("level", level.String())
			}
			return a
		},
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	envoyLogger := NewEnvoyLoggerWithHandler(handler)
	envoyLogger.level = level

	return &TextEnvoyLogger{
		EnvoyLogger: envoyLogger,
	}
}

// NewTextEnvoyLoggerWithWriter creates a text logger that writes to a specific writer
func NewTextEnvoyLoggerWithWriter(w io.Writer, level Level) *TextEnvoyLogger {
	opts := &slog.HandlerOptions{
		Level:     slog.Level(level),
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006-01-02 15:04:05.000"))
			}
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				return slog.String("level", level.String())
			}
			return a
		},
	}

	handler := slog.NewTextHandler(w, opts)
	envoyLogger := NewEnvoyLoggerWithHandler(handler)
	envoyLogger.level = level

	return &TextEnvoyLogger{
		EnvoyLogger: envoyLogger,
	}
}

// NewDefaultTextEnvoyLogger creates a default text-based Envoy logger
func NewDefaultTextEnvoyLogger() *TextEnvoyLogger {
	return NewTextEnvoyLogger(InfoLevel)
}
