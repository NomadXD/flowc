# API Type Adapter Pattern Design

This document describes the design for handling different API types (REST, gRPC, WebSocket, SSE, GraphQL) within the FlowC xDS translator architecture using an adapter pattern.

## Problem Statement

FlowC supports multiple API types, each with different protocol requirements:

| Aspect | REST | gRPC | WebSocket | SSE | GraphQL |
|--------|------|------|-----------|-----|---------|
| **Protocol** | HTTP/1.1, HTTP/2 | HTTP/2 (always) | HTTP/1.1 upgrade | HTTP/1.1 chunked | HTTP/1.1, HTTP/2 |
| **Cluster settings** | Standard | `http2_protocol_options` | `upgrade_configs` | Standard | Standard |
| **Route matching** | Path + method | Path + `:authority` | Path + upgrade header | Path | Single `/graphql` path |
| **Timeouts** | Standard | Stream timeouts | Idle timeout (long) | Stream timeout (long) | Standard |
| **Health checks** | HTTP GET | gRPC health protocol | TCP or HTTP | HTTP | HTTP POST |
| **Load balancing** | Any | Any | Sticky (often) | Any | Any |
| **Retry** | Standard | gRPC-specific codes | Usually none | Usually none | Standard |

## Design Decision

**Use the same CompositeTranslator with API-type-aware strategy adapters** rather than creating separate translators for each API type.

### Rationale

The differences between API types are primarily in **how strategies behave**, not in the **translation orchestration flow**. The CompositeTranslator's 7-phase approach works for all API types:

1. Generate Clusters ← cluster settings vary (HTTP/2, upgrades)
2. Apply Load Balancing ← mostly same
3. Generate Routes ← matching logic varies
4. Apply Retry ← retry conditions vary
5. Generate Listeners ← upgrade configs vary
6. Apply Rate Limiting ← same
7. Apply Observability ← same

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           StrategyFactory                                    │
│  CreateStrategySet(config, deployment, ir) → StrategySet                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    API Type Adapter Layer                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │    REST     │  │    gRPC     │  │  WebSocket  │  │     SSE     │        │
│  │  (no-op)    │  │   Adapter   │  │   Adapter   │  │   Adapter   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Base Strategy Set                                      │
│  ┌──────────────┐ ┌────────────┐ ┌─────────────┐ ┌───────┐ ┌─────────────┐ │
│  │ Deployment   │ │ RouteMatch │ │LoadBalancing│ │ Retry │ │ RateLimit   │ │
│  │ (Basic/etc)  │ │ (Prefix/   │ │(RoundRobin/ │ │(Cons/ │ │ (NoOp/etc)  │ │
│  │              │ │  etc)      │ │ etc)        │ │ etc)  │ │             │ │
│  └──────────────┘ └────────────┘ └─────────────┘ └───────┘ └─────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## File Structure

```
internal/flowc/xds/translator/
├── adapters/
│   ├── adapters.go          # APITypeAdapter interface + factory
│   ├── grpc.go              # gRPC adapters
│   ├── websocket.go         # WebSocket adapters
│   ├── sse.go               # SSE adapters
│   └── graphql.go           # GraphQL adapters
├── strategies.go            # (existing) strategy interfaces
├── resolver.go              # (modified) StrategyFactory uses adapters
└── ...
```

## Core Interfaces

### APITypeAdapter Interface

```go
// APITypeAdapter wraps a base StrategySet and adapts it for a specific API type
type APITypeAdapter interface {
    // Adapt wraps the base strategy set with API-type-specific behavior
    Adapt(base *translator.StrategySet) *translator.StrategySet

    // APIType returns which API type this adapter handles
    APIType() ir.APIType
}
```

### AdapterRegistry

```go
// AdapterRegistry holds adapters for different API types
type AdapterRegistry struct {
    adapters map[ir.APIType]APITypeAdapter
}

// NewAdapterRegistry creates a registry with all built-in adapters
func NewAdapterRegistry() *AdapterRegistry {
    r := &AdapterRegistry{
        adapters: make(map[ir.APIType]APITypeAdapter),
    }

    // Register built-in adapters
    r.Register(NewGRPCAdapter())
    r.Register(NewWebSocketAdapter())
    r.Register(NewSSEAdapter())
    r.Register(NewGraphQLAdapter())
    // REST uses base strategies directly (no adapter needed)

    return r
}

// AdaptStrategies applies API-type-specific adaptations to a strategy set
func (r *AdapterRegistry) AdaptStrategies(base *translator.StrategySet, apiType ir.APIType) *translator.StrategySet {
    adapter := r.GetAdapter(apiType)
    if adapter == nil {
        // No adapter = use base strategies as-is (REST)
        return base
    }
    return adapter.Adapt(base)
}
```

## Adapter Implementations

### gRPC Adapter

The gRPC adapter modifies strategies to support HTTP/2 and gRPC-specific behavior.

```go
type GRPCAdapter struct{}

func (a *GRPCAdapter) APIType() ir.APIType {
    return ir.APITypeGRPC
}

func (a *GRPCAdapter) Adapt(base *translator.StrategySet) *translator.StrategySet {
    return &translator.StrategySet{
        Deployment:    &GRPCDeploymentAdapter{base: base.Deployment},
        RouteMatch:    &GRPCRouteMatchAdapter{base: base.RouteMatch},
        LoadBalancing: base.LoadBalancing, // LB works the same for gRPC
        Retry:         &GRPCRetryAdapter{base: base.Retry},
        RateLimit:     base.RateLimit,
        Observability: base.Observability,
    }
}
```

#### GRPCDeploymentAdapter

Wraps the base deployment strategy and adds HTTP/2 protocol options:

```go
func (a *GRPCDeploymentAdapter) GenerateClusters(ctx context.Context, deployment *models.APIDeployment) ([]*clusterv3.Cluster, error) {
    clusters, err := a.base.GenerateClusters(ctx, deployment)
    if err != nil {
        return nil, err
    }

    // Apply gRPC-specific settings to all clusters
    for _, c := range clusters {
        // Enable HTTP/2 for gRPC
        c.Http2ProtocolOptions = &corev3.Http2ProtocolOptions{
            MaxConcurrentStreams:        wrappedUInt32(100),
            InitialStreamWindowSize:     wrappedUInt32(65536),   // 64KB
            InitialConnectionWindowSize: wrappedUInt32(1048576), // 1MB
        }

        // Replace HTTP health check with gRPC health check
        if len(c.HealthChecks) > 0 {
            c.HealthChecks[0].HealthChecker = &corev3.HealthCheck_GrpcHealthCheck_{
                GrpcHealthCheck: &corev3.HealthCheck_GrpcHealthCheck{},
            }
        }
    }

    return clusters, nil
}
```

#### GRPCRouteMatchAdapter

Adds content-type header matching for gRPC:

```go
func (a *GRPCRouteMatchAdapter) CreateMatcher(path, method string, endpoint *ir.Endpoint) *routev3.RouteMatch {
    match := a.base.CreateMatcher(path, method, endpoint)

    // Add gRPC content-type header matcher
    match.Headers = append(match.Headers, &routev3.HeaderMatcher{
        Name: "content-type",
        HeaderMatchSpecifier: &routev3.HeaderMatcher_PrefixMatch{
            PrefixMatch: "application/grpc",
        },
    })

    return match
}
```

#### GRPCRetryAdapter

Uses gRPC-specific retry conditions:

```go
func (a *GRPCRetryAdapter) ConfigureRetry(route *routev3.Route, deployment *models.APIDeployment) error {
    if err := a.base.ConfigureRetry(route, deployment); err != nil {
        return err
    }

    routeAction, ok := route.Action.(*routev3.Route_Route)
    if !ok || routeAction.Route == nil {
        return nil
    }

    if routeAction.Route.RetryPolicy != nil {
        // Replace HTTP retry conditions with gRPC conditions
        routeAction.Route.RetryPolicy.RetryOn = "cancelled,deadline-exceeded,resource-exhausted,unavailable"
        routeAction.Route.RetryPolicy.PerTryTimeout = durationpb.New(10 * time.Second)
    }

    return nil
}
```

### WebSocket Adapter

The WebSocket adapter configures long-lived connection support.

```go
type WebSocketAdapter struct{}

func (a *WebSocketAdapter) Adapt(base *translator.StrategySet) *translator.StrategySet {
    return &translator.StrategySet{
        Deployment:    &WebSocketDeploymentAdapter{base: base.Deployment},
        RouteMatch:    &WebSocketRouteMatchAdapter{base: base.RouteMatch},
        LoadBalancing: &WebSocketLoadBalancingAdapter{base: base.LoadBalancing},
        Retry:         &translator.NoOpRetryStrategy{}, // No retry for WebSocket
        RateLimit:     base.RateLimit,
        Observability: base.Observability,
    }
}
```

#### WebSocketDeploymentAdapter

Configures long idle timeouts and circuit breakers:

```go
func (a *WebSocketDeploymentAdapter) applyWebSocketSettings(c *clusterv3.Cluster) {
    // WebSocket connections are long-lived
    c.CommonHttpProtocolOptions = &corev3.HttpProtocolOptions{
        IdleTimeout: durationpb.New(1 * time.Hour),
    }

    // Connection pool settings for WebSocket
    c.CircuitBreakers = &clusterv3.CircuitBreakers{
        Thresholds: []*clusterv3.CircuitBreakers_Thresholds{
            {
                MaxConnections:     wrappedUInt32(10000),
                MaxPendingRequests: wrappedUInt32(10000),
                MaxRequests:        wrappedUInt32(10000),
            },
        },
    }
}
```

#### WebSocketRouteMatchAdapter

Adds upgrade header matching:

```go
func (a *WebSocketRouteMatchAdapter) CreateMatcher(path, method string, endpoint *ir.Endpoint) *routev3.RouteMatch {
    match := a.base.CreateMatcher(path, method, endpoint)

    // WebSocket requires upgrade header
    match.Headers = append(match.Headers, &routev3.HeaderMatcher{
        Name: "upgrade",
        HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
            ExactMatch: "websocket",
        },
    })

    return match
}
```

### SSE Adapter

The SSE adapter configures streaming timeouts and disables retry.

```go
type SSEAdapter struct{}

func (a *SSEAdapter) Adapt(base *translator.StrategySet) *translator.StrategySet {
    return &translator.StrategySet{
        Deployment:    &SSEDeploymentAdapter{base: base.Deployment},
        RouteMatch:    base.RouteMatch, // SSE uses standard HTTP routes
        LoadBalancing: base.LoadBalancing,
        Retry:         &translator.NoOpRetryStrategy{}, // No retry for streaming
        RateLimit:     base.RateLimit,
        Observability: base.Observability,
    }
}
```

#### SSEDeploymentAdapter

Configures streaming-appropriate timeouts:

```go
func (a *SSEDeploymentAdapter) applySSESettings(c *clusterv3.Cluster) {
    c.CommonHttpProtocolOptions = &corev3.HttpProtocolOptions{
        IdleTimeout: durationpb.New(24 * time.Hour),
    }
}
```

### GraphQL Adapter

The GraphQL adapter is minimal since GraphQL uses standard HTTP semantics, but may configure single-endpoint routing.

```go
type GraphQLAdapter struct{}

func (a *GraphQLAdapter) Adapt(base *translator.StrategySet) *translator.StrategySet {
    return &translator.StrategySet{
        Deployment:    base.Deployment,
        RouteMatch:    &GraphQLRouteMatchAdapter{base: base.RouteMatch},
        LoadBalancing: base.LoadBalancing,
        Retry:         base.Retry,
        RateLimit:     base.RateLimit,
        Observability: base.Observability,
    }
}
```

## StrategyFactory Changes

The `StrategyFactory` is modified to use the adapter registry:

```go
type StrategyFactory struct {
    options         *TranslatorOptions
    logger          *logger.EnvoyLogger
    adapterRegistry *adapters.AdapterRegistry  // NEW
}

func NewStrategyFactory(options *TranslatorOptions, log *logger.EnvoyLogger) *StrategyFactory {
    if options == nil {
        options = DefaultTranslatorOptions()
    }
    return &StrategyFactory{
        options:         options,
        logger:          log,
        adapterRegistry: adapters.NewAdapterRegistry(),
    }
}

// CreateStrategySet now takes API type for adaptation
func (f *StrategyFactory) CreateStrategySet(
    config *types.StrategyConfig,
    deployment *models.APIDeployment,
    apiType ir.APIType,  // NEW parameter
) (*StrategySet, error) {
    if config == nil {
        config = DefaultStrategyConfig()
    }

    // Create base strategy set
    base, err := f.createBaseStrategySet(config, deployment)
    if err != nil {
        return nil, err
    }

    // Apply API-type-specific adaptations
    adapted := f.adapterRegistry.AdaptStrategies(base, apiType)

    return adapted, nil
}
```

## Integration Flow

```
User deploys gRPC API
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  BundleLoader detects gRPC from .proto file                 │
│  IR.Metadata.Type = "grpc"                                  │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  StrategyFactory.CreateStrategySet(config, deploy, "grpc") │
│  1. Creates base strategies (Basic, Prefix, RoundRobin...) │
│  2. Calls AdapterRegistry.AdaptStrategies(base, "grpc")    │
│  3. GRPCAdapter wraps strategies with adapters             │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  CompositeTranslator.Translate() - UNCHANGED               │
│  Phase 1: GRPCDeploymentAdapter.GenerateClusters()         │
│           → Adds HTTP/2 options to clusters                │
│  Phase 2: LoadBalancing (unchanged)                        │
│  Phase 3: GRPCRouteMatchAdapter.CreateMatcher()            │
│           → Adds content-type: application/grpc header     │
│  Phase 4: GRPCRetryAdapter.ConfigureRetry()                │
│           → Uses gRPC retry conditions                     │
│  ...                                                       │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
   xDS Resources with gRPC-specific configuration
```

## Summary: What Each Adapter Changes

| API Type | Cluster Changes | Route Changes | Retry Changes | LB Changes |
|----------|-----------------|---------------|---------------|------------|
| **REST** | None (base) | None (base) | None (base) | None (base) |
| **gRPC** | +HTTP/2 options, gRPC health check | +content-type header | gRPC retry codes | None |
| **WebSocket** | +Long idle timeout, circuit breakers | +upgrade header | Disabled (NoOp) | Sticky preferred |
| **SSE** | +Streaming timeout | None | Disabled (NoOp) | None |
| **GraphQL** | None | Single /graphql path | None | None |

## Benefits

1. **Single CompositeTranslator** - No need for separate translator implementations
2. **Composable** - Adapters wrap strategies, can be mixed and matched
3. **Extensible** - Add new API types by implementing `APITypeAdapter`
4. **Non-breaking** - REST APIs work exactly as before (no adapter applied)
5. **Testable** - Each adapter can be unit tested independently
6. **Clear separation** - Protocol concerns isolated from deployment/routing concerns

## When to Use Separate Translators Instead

Separate translators make sense when the **orchestration flow itself differs**, not just the strategy behavior:

| Scenario | Separate Translator? | Why |
|----------|---------------------|-----|
| gRPC API | No | Same flow, different cluster/route settings |
| WebSocket API | No | Same flow, different upgrade config |
| Edge Gateway deployment | **Maybe** | Different listener setup, TLS termination, WAF integration |
| External policy engine | **Yes** | Completely different flow (delegate to external service) |
| Multi-cluster deployment | **Maybe** | Might need to generate resources for multiple control planes |
| Sidecar injection | **Maybe** | Might generate different resource types entirely |

## Future Considerations

1. **Custom Adapters**: Allow users to register custom adapters for specialized API types
2. **Adapter Composition**: Support chaining multiple adapters (e.g., gRPC + custom security)
3. **Adapter Configuration**: Allow per-deployment adapter configuration in `flowc.yaml`
4. **Listener Adapters**: Extend pattern to listener configuration (TLS, access logging)
