# FlowC Configuration Management

This package provides comprehensive configuration management for the FlowC control plane. It supports YAML-based configuration with environment variable overrides and sensible defaults.

## Overview

The configuration system provides three levels of configuration:

1. **Built-in defaults** - Hardcoded sensible defaults for all settings
2. **YAML configuration file** - Optional configuration file for customization
3. **Environment variables** - Runtime overrides for specific settings

## Configuration File Locations

The configuration loader searches for configuration files in the following order:

1. Path specified in `FLOWC_CONFIG` environment variable
2. `./flowc-config.yaml` (current directory)
3. `./flowc-config.yml` (current directory)
4. `./config/flowc-config.yaml`
5. `./config/flowc-config.yml`
6. `/etc/flowc/config.yaml`
7. `/etc/flowc/config.yml`

If no configuration file is found, the system uses built-in defaults.

## Usage

### Basic Usage

```go
package main

import (
    "log"
    "github.com/flowc-labs/flowc/internal/flowc/config"
)

func main() {
    // Load configuration (auto-discovers config file)
    cfg, err := config.Load("")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Access configuration values
    apiPort := cfg.Server.APIPort
    xdsPort := cfg.Server.XDSPort
    
    // Get parsed duration values
    readTimeout := cfg.GetServerReadTimeout()
    
    // Access default strategies
    defaultStrategies := cfg.Defaults.Strategies
}
```

### Load from Specific Path

```go
cfg, err := config.Load("/path/to/flowc-config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

### Load from Byte Data

```go
configData := []byte(`
server:
  api_port: 8080
  xds_port: 18000
`)

cfg, err := config.LoadFromData(configData)
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

## Configuration Structure

### Server Configuration

Controls the API and XDS server behavior:

```yaml
server:
  api_port: 8080              # REST API server port
  xds_port: 18000             # XDS gRPC server port
  read_timeout: "30s"         # HTTP read timeout
  write_timeout: "30s"        # HTTP write timeout
  idle_timeout: "60s"         # HTTP idle timeout
  graceful_shutdown: true     # Enable graceful shutdown
  shutdown_timeout: "10s"     # Graceful shutdown timeout
```

### XDS Configuration

Controls XDS server and Envoy proxy defaults:

```yaml
xds:
  default_listener_port: 9095         # Default listener port for Envoy
  default_node_id: "test-envoy-node"  # Default node ID for testing
  
  snapshot_cache:
    ads: true                          # Enable Aggregated Discovery Service
  
  grpc:
    keepalive_time: "30s"                    # Time between keepalive pings
    keepalive_timeout: "5s"                  # Keepalive response timeout
    keepalive_min_time: "5s"                 # Min time between pings
    keepalive_permit_without_stream: true    # Allow pings without streams
```

### Default Strategy Configuration (Optional - Not Recommended)

**⚠️ IMPORTANT: Strategies are deployment-specific configuration!**

Strategies define how individual APIs behave and should be configured per-deployment in `flowc.yaml`, not in the global control plane configuration.

The `defaults.strategies` section is **optional** and only serves as an organization-wide fallback for deployments that don't specify their own strategies. Most users should **leave this section commented out** or omitted entirely.

**Why strategies don't belong in control plane config:**
- Different APIs have different retry requirements
- Load balancing depends on the service architecture
- Rate limiting varies by API usage patterns
- Canary vs blue-green is a deployment decision, not infrastructure

**Correct approach:**
```yaml
# In flowc.yaml (per-deployment)
name: "my-api"
version: "v1"

# Define strategies here, specific to this API
strategies:
  deployment:
    type: "canary"
  retry:
    type: "aggressive"
    max_retries: 3
```

**If you still want organization-wide fallback defaults** (not recommended):
```yaml
# In flowc-config.yaml (control plane)
defaults:
  strategies:
    deployment:
      type: "basic"
    retry:
      type: "conservative"
```

### Logging Configuration

Controls logging behavior:

```yaml
logging:
  level: "info"              # Options: debug, info, warn, error
  format: "json"             # Options: json, text
  output: "stdout"           # Options: stdout, stderr, or file path
  structured: true           # Enable structured logging
  enable_caller: false       # Include caller info in logs
  enable_stacktrace: false   # Include stack traces for errors
```

### Feature Flags

Enable/disable specific features:

```yaml
features:
  external_translators: true   # Enable external translator support
  openapi_validation: true     # Enable OpenAPI validation
  metrics: false               # Enable metrics collection
  tracing: false               # Enable distributed tracing
  rate_limiting: false         # Enable rate limiting
```

## Environment Variable Overrides

All configuration values can be overridden using environment variables:

### Server Configuration

- `FLOWC_API_PORT` - API server port
- `FLOWC_XDS_PORT` - XDS server port
- `FLOWC_READ_TIMEOUT` - HTTP read timeout
- `FLOWC_WRITE_TIMEOUT` - HTTP write timeout
- `FLOWC_IDLE_TIMEOUT` - HTTP idle timeout
- `FLOWC_SHUTDOWN_TIMEOUT` - Graceful shutdown timeout
- `FLOWC_GRACEFUL_SHUTDOWN` - Enable graceful shutdown (true/false)

### XDS Configuration

- `FLOWC_DEFAULT_LISTENER_PORT` - Default Envoy listener port
- `FLOWC_DEFAULT_NODE_ID` - Default Envoy node ID
- `FLOWC_XDS_ADS` - Enable ADS (true/false)
- `FLOWC_GRPC_KEEPALIVE_TIME` - gRPC keepalive time
- `FLOWC_GRPC_KEEPALIVE_TIMEOUT` - gRPC keepalive timeout
- `FLOWC_GRPC_KEEPALIVE_MIN_TIME` - gRPC keepalive min time
- `FLOWC_GRPC_KEEPALIVE_PERMIT_WITHOUT_STREAM` - Allow keepalive without stream (true/false)

### Logging Configuration

- `FLOWC_LOG_LEVEL` - Log level (debug, info, warn, error)
- `FLOWC_LOG_FORMAT` - Log format (json, text)
- `FLOWC_LOG_OUTPUT` - Log output (stdout, stderr, or file path)
- `FLOWC_LOG_STRUCTURED` - Enable structured logging (true/false)
- `FLOWC_LOG_ENABLE_CALLER` - Enable caller info (true/false)
- `FLOWC_LOG_ENABLE_STACKTRACE` - Enable stack traces (true/false)

### Feature Flags

- `FLOWC_FEATURE_EXTERNAL_TRANSLATORS` - Enable external translators (true/false)
- `FLOWC_FEATURE_OPENAPI_VALIDATION` - Enable OpenAPI validation (true/false)
- `FLOWC_FEATURE_METRICS` - Enable metrics (true/false)
- `FLOWC_FEATURE_TRACING` - Enable tracing (true/false)
- `FLOWC_FEATURE_RATE_LIMITING` - Enable rate limiting (true/false)

### Configuration File Path Override

- `FLOWC_CONFIG` - Path to configuration file

## Examples

### Example 1: Development Configuration

```yaml
server:
  api_port: 8080
  xds_port: 18000

xds:
  default_node_id: "dev-envoy-node"

logging:
  level: "debug"
  format: "text"
  enable_caller: true

features:
  external_translators: true
  openapi_validation: true
```

### Example 2: Production Configuration

```yaml
server:
  api_port: 8080
  xds_port: 18000
  read_timeout: "60s"
  write_timeout: "60s"
  shutdown_timeout: "30s"

xds:
  default_listener_port: 9095
  grpc:
    keepalive_time: "60s"
    keepalive_timeout: "10s"

defaults:
  strategies:
    deployment:
      type: "canary"
    
    retry:
      type: "aggressive"
      max_retries: 3
    
    observability:
      tracing:
        enabled: true
        provider: "jaeger"
        sampling_rate: 0.1
      
      metrics:
        enabled: true
        provider: "prometheus"
        path: "/metrics"
        port: 9090

logging:
  level: "info"
  format: "json"
  output: "/var/log/flowc/flowc.log"
  structured: true

features:
  external_translators: true
  openapi_validation: true
  metrics: true
  tracing: true
  rate_limiting: true
```

### Example 3: Using Environment Variables

```bash
# Override API port
export FLOWC_API_PORT=9090

# Override log level
export FLOWC_LOG_LEVEL=debug

# Enable metrics
export FLOWC_FEATURE_METRICS=true

# Start flowc (will use config file + env overrides)
./flowc
```

## Validation

The configuration system performs comprehensive validation:

1. **Port validation** - Ensures ports are in valid range (1-65535) and don't conflict
2. **Duration validation** - Validates all duration strings (e.g., "30s", "5m")
3. **Enum validation** - Validates enum values (log levels, formats, etc.)
4. **Required fields** - Ensures all required fields are present

Validation errors are returned with descriptive messages indicating what's wrong and how to fix it.

## Configuration vs Per-Deployment Settings

It's important to understand the difference between global configuration and per-deployment settings:

### Global Configuration (`flowc-config.yaml`)

- **Location**: `internal/flowc/config/`
- **Scope**: Control plane settings, server configuration, global defaults
- **Examples**: API port, XDS port, logging, default strategies
- **Loaded**: Once at startup

### Per-Deployment Configuration (`flowc.yaml`)

- **Location**: `pkg/types/types.go`
- **Scope**: Individual API deployment metadata
- **Examples**: API name, version, upstream config, gateway settings
- **Loaded**: Each time an API is deployed

### Strategy Configuration (`internal/flowc/xds/translator/config.go`)

- **Location**: Can be used in both global config and per-deployment config
- **Scope**: XDS generation strategies
- **Examples**: Deployment strategy, retry policy, load balancing
- **Priority**: Per-deployment overrides global defaults

## Best Practices

1. **Use defaults when possible** - Built-in defaults work well for most use cases
2. **Override only what you need** - No need to specify every setting
3. **Use environment variables for secrets** - Don't put sensitive data in config files
4. **Use environment variables for deployment-specific overrides** - Different ports per environment
5. **Keep strategy defaults conservative** - Individual deployments can be more aggressive
6. **Enable features gradually** - Test each feature before enabling in production
7. **Version your config files** - Keep config files in version control
8. **Document custom settings** - Add comments explaining non-standard configurations

## Migration Guide

If you're migrating from hardcoded configuration:

1. Create a basic `flowc-config.yaml` with your current settings
2. Test that the application starts with the config file
3. Gradually move hardcoded values to the config file
4. Remove hardcoded values from the code
5. Add environment variable overrides for deployment-specific settings

## Future Enhancements

Potential future improvements to the configuration system:

- Hot reload of configuration without restart
- Remote configuration backends (etcd, Consul)
- Configuration validation at build time
- Configuration schema documentation generation
- Configuration migration tools between versions

