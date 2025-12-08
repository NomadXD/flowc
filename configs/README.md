# FlowC Configuration Files

This directory contains example configuration files for the FlowC control plane.

## Files

- **flowc-config.example.yaml** - Basic example configuration with sensible defaults
- **flowc-config.production.yaml** - Production-ready configuration with observability and advanced features

## Quick Start

1. Copy the example configuration:
   ```bash
   cp config/flowc-config.example.yaml flowc-config.yaml
   ```

2. Edit `flowc-config.yaml` to match your requirements

3. Start FlowC (it will auto-discover the config file):
   ```bash
   ./flowc
   ```

## Configuration File Locations

FlowC searches for configuration files in the following order:

1. Path specified in `FLOWC_CONFIG` environment variable
2. `./flowc-config.yaml` (current directory)
3. `./flowc-config.yml`
4. `./config/flowc-config.yaml`
5. `./config/flowc-config.yml`
6. `/etc/flowc/config.yaml`
7. `/etc/flowc/config.yml`

## Environment Variable Overrides

You can override any configuration value using environment variables:

```bash
# Override API port
export FLOWC_API_PORT=9090

# Override log level
export FLOWC_LOG_LEVEL=debug

# Enable metrics
export FLOWC_FEATURE_METRICS=true

# Start FlowC with overrides
./flowc
```

See the main README in `internal/flowc/config/` for a complete list of environment variables.

## Examples

### Development Setup

For local development, use the default configuration or minimal overrides:

```yaml
server:
  api_port: 8080
  xds_port: 18000

logging:
  level: "debug"
  format: "text"
```

### Production Setup

For production, use the production example as a starting point:

```bash
cp config/flowc-config.production.yaml flowc-config.yaml
# Edit as needed
./flowc
```

### Docker Setup

When running in Docker, you can mount the config file:

```bash
docker run -v $(pwd)/flowc-config.yaml:/app/flowc-config.yaml flowc:latest
```

Or use environment variables:

```bash
docker run \
  -e FLOWC_API_PORT=8080 \
  -e FLOWC_XDS_PORT=18000 \
  -e FLOWC_LOG_LEVEL=info \
  flowc:latest
```

### Kubernetes Setup

When deploying FlowC in Kubernetes, you can use a ConfigMap to provide the configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: flowc-config
data:
  flowc-config.yaml: |
    server:
      api_port: 8080
      xds_port: 18000
    
    logging:
      level: "info"
      format: "json"
    
    # ... rest of your configuration

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flowc
spec:
  replicas: 1
  selector:
    matchLabels:
      app: flowc
  template:
    metadata:
      labels:
        app: flowc
    spec:
      containers:
      - name: flowc
        image: flowc:latest
        ports:
        - containerPort: 8080
          name: api
        - containerPort: 18000
          name: xds
        volumeMounts:
        - name: config
          mountPath: /app/flowc-config.yaml
          subPath: flowc-config.yaml
      volumes:
      - name: config
        configMap:
          name: flowc-config
```

Or use environment variables for configuration:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flowc
spec:
  template:
    spec:
      containers:
      - name: flowc
        image: flowc:latest
        env:
        - name: FLOWC_API_PORT
          value: "8080"
        - name: FLOWC_XDS_PORT
          value: "18000"
        - name: FLOWC_LOG_LEVEL
          value: "info"
        - name: FLOWC_LOG_FORMAT
          value: "json"
```

## Validation

The configuration system performs comprehensive validation:

- Port numbers (1-65535)
- Duration formats (e.g., "30s", "5m", "1h")
- Log levels (debug, info, warn, error)
- Required fields

If validation fails, FlowC will print a detailed error message and exit.

## Support

For more information, see:
- Main configuration package documentation: `internal/flowc/config/README.md`
- FlowC documentation: `docs/`

