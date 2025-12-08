# Envoy Logger Package

This package provides a logger implementation that implements the log interface expected by the Envoy Go control plane library. It's built on top of Go's built-in `slog` package for structured logging with proper source location tracking.

## Features

- **Envoy Control Plane Compatible**: Implements the standard log interface
- **Structured Logging**: Built on `slog` for efficient structured logging
- **Accurate Source Location**: Logs show the actual caller's file and line number, not the wrapper's location
- **Dynamic Level Changes**: Runtime log level changes supported via `SetLevel()`
- **Multiple Output Formats**: JSON and text output formats
- **Configurable Levels**: Debug, Info, Warn, Error, and Fatal levels
- **Context Support**: Support for adding context and fields
- **File Logging**: Support for writing logs to files

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

## Logger Methods

The logger provides the following methods:

### Basic Logging
- `Debug(msg string)` - Log debug message
- `Debugf(format string, args ...interface{})` - Log formatted debug message
- `Info(msg string)` - Log info message
- `Infof(format string, args ...interface{})` - Log formatted info message
- `Warn(msg string)` - Log warning message
- `Warnf(format string, args ...interface{})` - Log formatted warning message
- `Error(msg string)` - Log error message
- `Errorf(format string, args ...interface{})` - Log formatted error message
- `Fatal(msg string)` - Log fatal message and exit with code 1
- `Fatalf(format string, args ...interface{})` - Log formatted fatal message and exit

### Structured Logging
- `WithField(key string, value interface{}) *EnvoyLogger` - Add a field to logger
- `WithFields(fields map[string]interface{}) *EnvoyLogger` - Add multiple fields
- `WithError(err error) *EnvoyLogger` - Add error field to logger
- `WithContext(ctx context.Context) *EnvoyLogger` - Add context (currently a no-op, extend as needed)

### Level Management
- `SetLevel(level Level)` - Change log level dynamically at runtime
- `GetLevel() Level` - Get current log level
- `IsDebugEnabled() bool` - Check if debug logging is enabled
- `IsInfoEnabled() bool` - Check if info logging is enabled
- `IsWarnEnabled() bool` - Check if warning logging is enabled
- `IsErrorEnabled() bool` - Check if error logging is enabled

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
    Type:       logger.JSONLogger,
    Level:      logger.DebugLevel,
    Output:     os.Stdout,
    AddSource:  true,
    TimeFormat: "2006-01-02 15:04:05.000",
}

log := logger.NewLogger(config)
```

### File Logger Resource Management

**Important**: When using `NewFileLogger()`, the file is opened but not automatically closed. You should manage the file lifecycle:

```go
// Option 1: Accept the file remains open (OS will close on process exit)
log, err := logger.NewFileLogger("/var/log/app.log", logger.InfoLevel)
if err != nil {
    panic(err)
}
defer func() {
    // File will be closed when process exits
    // Consider implementing log rotation externally
}()

// Option 2: Use NewTextEnvoyLoggerWithWriter for explicit file management
file, err := os.OpenFile("/var/log/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
    panic(err)
}
defer file.Close()

log := logger.NewTextEnvoyLoggerWithWriter(file, logger.InfoLevel)
```

## Implementation Details

### Source Location Tracking

The logger correctly captures the **actual caller's location**, not the wrapper's location. This is achieved by using `runtime.Callers()` with proper stack frame skipping.

**Example output:**
```json
{"time":"2025-12-08T14:59:30.227","level":"INFO","source":"main.go:45","msg":"Server starting"}
{"time":"2025-12-08T14:59:30.228","level":"INFO","source":"handler.go:123","msg":"Request received"}
```

Note how `source` correctly shows `main.go:45` and `handler.go:123`, not `envoy_logger.go:XXX`.

### Dynamic Level Changes

The logger uses `slog.LevelVar` to support runtime level changes. This means you can dynamically adjust logging verbosity without restarting your application:

```go
// Production: start with INFO
log := logger.NewLogger(&logger.LoggerConfig{
    Type:  logger.JSONLogger,
    Level: logger.InfoLevel,
})

// Enable debug mode for troubleshooting
log.SetLevel(logger.DebugLevel)

// Disable debug mode
log.SetLevel(logger.InfoLevel)
```

### Context Support

The `WithContext()` method is currently a no-op placeholder. Extend it to extract values from context as needed:

```go
// Future implementation example:
func (l *EnvoyLogger) WithContext(ctx context.Context) *EnvoyLogger {
    fields := make(map[string]interface{})
    
    if traceID := ctx.Value("trace-id"); traceID != nil {
        fields["trace_id"] = traceID
    }
    if requestID := ctx.Value("request-id"); requestID != nil {
        fields["request_id"] = requestID
    }
    
    if len(fields) > 0 {
        return l.WithFields(fields)
    }
    return l
}
```

## Performance Considerations

- **Level Checks**: Use `IsDebugEnabled()` etc. to avoid expensive operations when logging is disabled:
  ```go
  if log.IsDebugEnabled() {
      // Expensive serialization only happens if debug is enabled
      log.WithField("data", expensiveToSerialize()).Debug("Debug info")
  }
  ```

- **Structured Fields**: `WithField()` and `WithFields()` create new logger instances. Reuse loggers when possible:
  ```go
  // Good: reuse logger with common fields
  requestLog := log.WithFields(map[string]interface{}{
      "request_id": reqID,
      "user_id": userID,
  })
  requestLog.Info("Processing request")
  requestLog.Info("Request completed")
  
  // Less efficient: recreate fields every time
  log.WithField("request_id", reqID).Info("Processing request")
  log.WithField("request_id", reqID).Info("Request completed")
  ```

## Limitations

1. **Fatal Logs**: `Fatal()` and `Fatalf()` call `os.Exit(1)` immediately. Ensure all cleanup is done before calling fatal logs, or use `defer` statements.

2. **File Rotation**: The package doesn't provide built-in log rotation. Use external tools (e.g., `logrotate`) or implement rotation separately.

3. **WithContext()**: Currently a no-op. Implement context value extraction based on your application's needs.
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
    // Use WithField to add structured data
    log.WithField("config", config).Debug("Processing configuration")
}
```

### Dynamic Level Changes

```go
log := logger.NewDefaultEnvoyLogger()

// Start with INFO level
log.Info("Server starting")

// Change to DEBUG level at runtime
log.SetLevel(logger.DebugLevel)
log.Debug("Debug mode enabled") // This will now be logged

// Change back to INFO level
log.SetLevel(logger.InfoLevel)
log.Debug("This won't be logged") // Filtered out
```
