package logger

import (
	"io"
	"os"
)

// LoggerType represents the type of logger to create
type LoggerType int

const (
	JSONLogger LoggerType = iota
	TextLogger
)

// LoggerConfig holds configuration for creating loggers
type LoggerConfig struct {
	Type       LoggerType
	Level      Level
	Output     io.Writer
	AddSource  bool
	TimeFormat string
}

// DefaultLoggerConfig returns a default logger configuration
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Type:       TextLogger,
		Level:      InfoLevel,
		Output:     os.Stdout,
		AddSource:  true,
		TimeFormat: "2006-01-02 15:04:05.000",
	}
}

// NewLogger creates a new logger based on the configuration
func NewLogger(config *LoggerConfig) *EnvoyLogger {
	if config == nil {
		config = DefaultLoggerConfig()
	}

	switch config.Type {
	case JSONLogger:
		return NewEnvoyLogger(config.Level)
	case TextLogger:
		if config.Output != nil {
			return NewTextEnvoyLoggerWithWriter(config.Output, config.Level).EnvoyLogger
		}
		return NewTextEnvoyLogger(config.Level).EnvoyLogger
	default:
		return NewDefaultEnvoyLogger()
	}
}

// NewJSONLogger creates a JSON logger with the specified level
func NewJSONLogger(level Level) *EnvoyLogger {
	return NewEnvoyLogger(level)
}

// NewTextLogger creates a text logger with the specified level
func NewTextLogger(level Level) *EnvoyLogger {
	return NewTextEnvoyLogger(level).EnvoyLogger
}

// NewFileLogger creates a logger that writes to a file
func NewFileLogger(filename string, level Level) (*EnvoyLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	config := &LoggerConfig{
		Type:   TextLogger,
		Level:  level,
		Output: file,
	}

	return NewLogger(config), nil
}
