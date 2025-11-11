# xDS Translator Architecture

This package implements a pluggable architecture for translating FlowC deployment representations into Envoy xDS resources.

## Overview

The translator architecture follows a **Strategy Pattern** that separates the internal FlowC deployment representation from the xDS resource generation logic. This allows for:

1. **Multiple translation strategies** (basic, canary, blue-green, etc.)
2. **External xDS servers** for custom translation logic
3. **Easy extensibility** for new deployment patterns
4. **Clear separation of concerns**

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     FlowC Deployment                        │
│                   (Metadata + OpenAPI)                      │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   DeploymentModel                           │
│              (Internal Representation)                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
           ┌──────────┴──────────┐
           │   Translator         │
           │    Interface         │
           └──────────┬──────────┘
                      │
     ┌────────────────┼────────────────┬───────────────┐
     ▼                ▼                ▼               ▼
┌─────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Basic   │    │ Canary   │    │BlueGreen │    │ External │
│Strategy │    │ Strategy │    │ Strategy │    │  Server  │
└────┬────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘
     │              │               │               │
     └──────────────┴───────────────┴───────────────┘
                      │
                      ▼
           ┌──────────────────────┐
           │   xDS Resources       │
           │ (Clusters, Routes,    │
           │  Listeners, Endpoints)│
           └──────────────────────┘
```

## Core Components

### 1. DeploymentModel

The internal representation that all translators work with. Contains:
- FlowC metadata (name, version, upstream, gateway config)
- OpenAPI specification
- Deployment context (node ID, namespace, labels)
- Traffic strategy configuration

```go
model := translator.NewDeploymentModel(metadata, openAPISpec, deploymentID)
model.WithNodeID("envoy-node-1").
      WithTrafficStrategy(&translator.TrafficStrategy{
          Type: "canary",
          Canary: &translator.CanaryConfig{
              BaselineVersion: "v1",
              CanaryVersion: "v2",
              CanaryWeight: 20,
          },
      })
```

### 2. Translator Interface

All translators must implement:

```go
type Translator interface {
    // Translate converts a deployment model into xDS resources
    Translate(ctx context.Context, model *DeploymentModel) (*XDSResources, error)
    
    // Name returns the name/type of this translator
    Name() string
    
    // Validate checks if the deployment model is valid for this translator
    Validate(model *DeploymentModel) error
}
```

### 3. Built-in Translators

#### BasicTranslator
- **Purpose**: Simple 1:1 mapping from deployment to xDS resources
- **Use Case**: Standard API deployments without advanced routing
- **Resources**: Creates one cluster per deployment, routes based on OpenAPI paths

```go
translator := translator.NewBasicTranslator(options, logger)
resources, err := translator.Translate(ctx, model)
```

#### CanaryTranslator
- **Purpose**: Implements weighted traffic splitting between versions
- **Use Case**: Gradual rollout of new API versions
- **Resources**: Creates clusters for baseline and canary, weighted routes
- **Features**:
  - Percentage-based traffic splitting
  - Header-based routing for targeted testing
  - Configurable canary weight (0-100%)

```go
strategy := &translator.TrafficStrategy{
    Type: "canary",
    Canary: &translator.CanaryConfig{
        BaselineVersion: "v1",
        CanaryVersion: "v2",
        CanaryWeight: 20, // 20% to canary, 80% to baseline
        MatchCriteria: &translator.MatchCriteria{
            Headers: map[string]string{
                "x-canary": "true",
            },
        },
    },
}
```

#### BlueGreenTranslator
- **Purpose**: Maintains two complete environments for zero-downtime switches
- **Use Case**: Risk-free deployments with instant rollback capability
- **Resources**: Creates clusters for both environments, routes to active only
- **Features**:
  - Instant traffic switching
  - Both environments always available
  - Easy rollback

```go
strategy := &translator.TrafficStrategy{
    Type: "blue-green",
    BlueGreen: &translator.BlueGreenConfig{
        ActiveVersion: "v1",
        StandbyVersion: "v2",
        AutoPromote: false,
    },
}
```

#### ExternalTranslator
- **Purpose**: Delegates translation to external HTTP/gRPC service
- **Use Case**: Custom translation logic, organization-specific patterns
- **Protocol**: HTTP POST with JSON request/response
- **Features**:
  - Custom translation logic outside FlowC
  - Supports any deployment pattern
  - Organization-specific requirements

```go
config := &translator.ExternalTranslatorConfig{
    Endpoint: "http://xds-translator.example.com/translate",
    Timeout: 30 * time.Second,
    Headers: map[string]string{
        "Authorization": "Bearer token",
    },
}
ext, err := translator.NewExternalTranslator(config, options, logger)
```

### 4. Factory

Manages translator registration and creation:

```go
// Create factory with all built-in translators
factory, err := translator.DefaultFactory(options, logger)

// Get a specific translator
basic, err := factory.Get("basic")

// Register custom translator
factory.Register("custom", myCustomTranslator)

// Create from configuration
config := &translator.TranslatorConfig{
    Type: "canary",
    Options: options,
}
t, err := translator.CreateFromConfig(config, logger)
```

## Usage Examples

### Example 1: Basic Deployment

```go
// Create deployment model
model := translator.NewDeploymentModel(metadata, openAPISpec, "deploy-123")
model.WithNodeID("envoy-node-1")

// Use basic translator
translator := translator.NewBasicTranslator(nil, logger)
resources, err := translator.Translate(context.Background(), model)

// Deploy to xDS cache
cache.SetSnapshot(nodeID, resources)
```

### Example 2: Canary Deployment

```go
// Create model with canary strategy
model := translator.NewDeploymentModel(metadata, openAPISpec, "deploy-456")
model.WithNodeID("envoy-node-1").
      WithTrafficStrategy(&translator.TrafficStrategy{
          Type: "canary",
          Canary: &translator.CanaryConfig{
              BaselineVersion: "v1.0.0",
              CanaryVersion: "v1.1.0",
              CanaryWeight: 10, // Start with 10% traffic to canary
          },
      })

// Use canary translator
translator := translator.NewCanaryTranslator(nil, logger)
resources, err := translator.Translate(context.Background(), model)

// Deploy to xDS cache
cache.SetSnapshot(nodeID, resources)

// Later, increase canary weight
model.Context.TrafficStrategy.Canary.CanaryWeight = 50
resources, err = translator.Translate(context.Background(), model)
cache.SetSnapshot(nodeID, resources)
```

### Example 3: External Translator

```go
// Configure external translator
config := &translator.ExternalTranslatorConfig{
    Endpoint: "http://xds-translator.internal/v1/translate",
    Timeout: 30 * time.Second,
}

ext, err := translator.NewExternalTranslator(config, options, logger)

// Translate using external service
model := translator.NewDeploymentModel(metadata, openAPISpec, "deploy-789")
resources, err := ext.Translate(context.Background(), model)
```

### Example 4: Factory-based Creation

```go
// Create factory
factory, err := translator.DefaultFactory(options, logger)

// Dynamically select translator based on configuration
translatorType := "canary" // From config file or API request
t, err := factory.Get(translatorType)

// Translate
resources, err := t.Translate(context.Background(), model)
```

## External Translator Protocol

External translators communicate via HTTP POST with JSON payloads.

### Request Format

```json
{
  "deployment_id": "deploy-123",
  "metadata": {
    "name": "my-api",
    "version": "v1.0.0",
    "context": "/api/v1",
    "upstream": {
      "host": "backend.example.com",
      "port": 8080,
      "scheme": "http"
    },
    "gateway": {
      "node_id": "envoy-node-1",
      "listener": "default"
    }
  },
  "openapi_spec": {
    "openapi": "3.0.0",
    "info": {...},
    "paths": {...}
  },
  "context": {
    "node_id": "envoy-node-1",
    "namespace": "production",
    "labels": {...}
  }
}
```

### Response Format

```json
{
  "success": true,
  "resources": {
    "clusters": [
      {
        "name": "my-api-v1-cluster",
        "type": "LOGICAL_DNS",
        ...
      }
    ],
    "routes": [
      {
        "name": "my-api-v1-route",
        "virtual_hosts": [...]
      }
    ],
    "listeners": [...],
    "endpoints": [...]
  }
}
```

The resources are Envoy protobuf messages serialized as JSON using `protojson`.

## Extension Points

### Creating a Custom Translator

```go
type CustomTranslator struct {
    options *translator.TranslatorOptions
    logger  *logger.EnvoyLogger
}

func (t *CustomTranslator) Name() string {
    return "custom"
}

func (t *CustomTranslator) Validate(model *translator.DeploymentModel) error {
    // Validate the model
    return nil
}

func (t *CustomTranslator) Translate(ctx context.Context, model *translator.DeploymentModel) (*translator.XDSResources, error) {
    // Your custom translation logic
    return &translator.XDSResources{
        Clusters: [...],
        Routes: [...],
    }, nil
}

// Register with factory
factory.Register("custom", &CustomTranslator{...})
```

## Configuration

Translator selection can be configured per deployment:

```yaml
# flowc.yaml
name: my-api
version: v1.0.0
context: /api/v1
upstream:
  host: backend.example.com
  port: 8080

# Translator configuration
translator:
  type: canary  # basic, canary, blue-green, external
  options:
    default_listener_port: 9095
    enable_https: true
  
  # For canary deployments
  traffic_strategy:
    type: canary
    canary:
      baseline_version: v1.0.0
      canary_version: v1.1.0
      canary_weight: 20
```

## Benefits

1. **Separation of Concerns**: Internal representation separate from xDS details
2. **Pluggable Strategies**: Easy to add new deployment patterns
3. **External Extensibility**: Support custom translation logic via external services
4. **Testability**: Each translator can be tested independently
5. **Flexibility**: Choose the right strategy for each deployment
6. **Future-Proof**: Architecture supports any xDS translation need

## Future Enhancements

- gRPC support for external translators
- A/B testing translator
- Shadow traffic translator
- Multi-cluster translator
- Service mesh integration
- Rate limiting and circuit breaker translators

