# Envoy Logger Package

This package provides a logger implementation that implements the log interface expected by the Envoy Go control plane library. It's built on top of Go's built-in `slog` package for structured logging.

## Features

- **Envoy Control Plane Compatible**: Implements the standard log interface
- **Structured Logging**: Built on `slog` for efficient structured logging
- **Multiple Output Formats**: JSON and text output formats
- **Configurable Levels**: Debug, Info, Warn, Error, and Fatal levels
- **Context Support**: Support for adding context and fields
- **File Logging**: Support for writing logs to files
- **Source Information**: Optional source file and line number logging

## Usage

### Basic Usage

```go
import "github.com/flowc-labs/flowc/pkg/logger"

// Create a default logger
log := logger.NewDefaultEnvoyLogger()

// Log messages
log.Info("Server starting")
log.Debug("Debug information")
log.Error("An error occurred")
```

### JSON Logger

```go
// Create a JSON logger
log := logger.NewJSONLogger(logger.InfoLevel)

// Log with structured data
log.WithField("port", 8080).Info("Server started")
log.WithFields(map[string]interface{}{
    "user": "john",
    "action": "login",
}).Info("User action")
```

### Text Logger

```go
// Create a text logger
log := logger.NewTextLogger(logger.DebugLevel)

// Log messages
log.Info("This is a text log message")
log.WithError(err).Error("Operation failed")
```

### File Logging

```go
// Create a file logger
log, err := logger.NewFileLogger("/var/log/xds-server.log", logger.InfoLevel)
if err != nil {
    log.Fatal("Failed to create file logger")
}
```

### Using with Envoy Control Plane

```go
import (
    "github.com/flowc-labs/flowc/pkg/logger"
    cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// Create logger
log := logger.NewDefaultEnvoyLogger()

// Use with Envoy cache
cache := cachev3.NewSnapshotCache(true, cachev3.IDHash{}, log)
```

## Logger Interface

The logger implements the following interface:

```go
type Logger interface {
    Debug(msg string)
    Debugf(format string, args ...interface{})
    Info(msg string)
    Infof(format string, args ...interface{})
    Warn(msg string)
    Warnf(format string, args ...interface{})
    Error(msg string)
    Errorf(format string, args ...interface{})
    Fatal(msg string)
    Fatalf(format string, args ...interface{})
    WithField(key string, value interface{}) Logger
    WithFields(fields map[string]interface{}) Logger
    WithError(err error) Logger
    WithContext(ctx context.Context) Logger
    SetLevel(level Level)
    GetLevel() Level
    IsDebugEnabled() bool
    IsInfoEnabled() bool
    IsWarnEnabled() bool
    IsErrorEnabled() bool
}
```

## Configuration

### Logger Levels

- `DebugLevel`: Debug messages
- `InfoLevel`: Informational messages
- `WarnLevel`: Warning messages
- `ErrorLevel`: Error messages
- `FatalLevel`: Fatal messages (exits program)

### Logger Types

- `JSONLogger`: Outputs structured JSON logs
- `TextLogger`: Outputs human-readable text logs

### Custom Configuration

```go
config := &logger.LoggerConfig{
    Type:      logger.JSONLogger,
    Level:     logger.DebugLevel,
    Output:    os.Stdout,
    AddSource: true,
}

log := logger.NewLogger(config)
```

## Examples

### Basic Server Logging

```go
func main() {
    log := logger.NewDefaultEnvoyLogger()
    
    log.Info("Starting XDS server")
    
    // Start server
    if err := startServer(); err != nil {
        log.WithError(err).Fatal("Failed to start server")
    }
    
    log.Info("Server started successfully")
}
```

### Structured Logging

```go
log := logger.NewJSONLogger(logger.InfoLevel)

// Log with context
log.WithFields(map[string]interface{}{
    "component": "xds-server",
    "version": "1.0.0",
    "port": 8080,
}).Info("Server configuration")

// Log with error
if err := processRequest(); err != nil {
    log.WithError(err).WithField("request_id", "12345").Error("Request failed")
}
```

### Debug Logging

```go
log := logger.NewDebugEnvoyLogger()

if log.IsDebugEnabled() {
    log.Debug("Processing configuration", "config", config)
}
```
