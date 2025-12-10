# xDS Control Plane Package

This package implements FlowC's Envoy xDS (Discovery Service) control plane, providing dynamic configuration management for Envoy proxies. It translates high-level API deployment specifications into Envoy xDS resources (clusters, routes, listeners, endpoints).

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Package Structure](#package-structure)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [Quick Start](#quick-start)
- [Configuration Management](#configuration-management)

## Overview

The xDS package is the **core control plane** of FlowC, responsible for:

- **Dynamic Configuration**: Generate Envoy xDS resources from API deployments
- **Strategy-Based Translation**: Support multiple deployment patterns (basic, canary, blue-green)
- **Resource Management**: Create and manage clusters, routes, listeners, and endpoints
- **Cache Management**: Maintain versioned configuration snapshots per Envoy node
- **gRPC Server**: Serve xDS protocol to connected Envoy proxies

**Key Features:**
- 📦 **Modular Architecture** - Separate concerns (translation, caching, serving, resource generation)
- 🎯 **Strategy Pattern** - Pluggable translation strategies for different deployment patterns
- 🔄 **Atomic Updates** - Snapshot-based configuration updates ensure consistency
- 🌐 **Multi-Node Support** - Independent configuration per Envoy node ID
- 🚀 **Production-Ready** - Built on Envoy's official go-control-plane library

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         API Deployment                               │
│                    (Metadata + IR + Spec)                            │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      Translator Package                              │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │         CompositeTranslator (Strategy Orchestration)         │   │
│  │  • Deployment Strategy  • Route Match Strategy               │   │
│  │  • Load Balancing      • Retry Strategy                      │   │
│  │  • Rate Limiting       • Observability                       │   │
│  └──────────────────────────────────────────────────────────────┘   │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             ▼
                      XDSResources
             (Clusters, Routes, Listeners, Endpoints)
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                       Resources Package                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │  Cluster   │  │   Route    │  │  Listener  │  │  Endpoint  │   │
│  │  Builders  │  │  Builders  │  │  Builders  │  │  Builders  │   │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘   │
│  (Helper functions to create Envoy protobuf messages)               │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        Cache Package                                 │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    ConfigManager                             │   │
│  │  • Snapshot versioning    • Per-node isolation               │   │
│  │  • Atomic updates         • Consistency validation           │   │
│  │  • Bulk operations        • Node lifecycle                   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                    (Envoy SnapshotCache)                             │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        Server Package                                │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     XDSServer                                │   │
│  │  • gRPC server (port 18000)                                  │   │
│  │  • ADS (Aggregated Discovery Service)                        │   │
│  │  • Connection management                                     │   │
│  │  • Keepalive configuration                                   │   │
│  └──────────────────────────────────────────────────────────────┘   │
└────────────────────────────┬─────────────────────────────────────────┘
                             │
                             ▼
                      ┌──────────────┐
                      │ Envoy Proxy  │
                      │  (Clients)   │
                      └──────────────┘
```

## Package Structure

```
internal/flowc/xds/
├── cache/              # Configuration snapshot management
│   └── cache.go        # ConfigManager for xDS resource versioning
│
├── resources/          # xDS resource builders
│   ├── cluster/        # Cluster resource creation
│   │   └── cluster.go  # CreateCluster(), CreateClusterWithScheme()
│   ├── route/          # Route configuration creation
│   │   └── route.go    # CreateRoute(), CreateRouteForOperation()
│   ├── listener/       # Listener resource creation
│   │   └── listener.go # CreateListener()
│   └── endpoint/       # Endpoint resource creation
│       └── endpoint.go # CreateLbEndpoint()
│
├── server/             # xDS gRPC server
│   └── server.go       # XDSServer implementation
│
└── translator/         # Strategy-based xDS generation
    ├── translator.go   # Core Translator interface
    ├── composite.go    # CompositeTranslator orchestration
    ├── strategies.go   # Strategy interfaces
    ├── strategy_*.go   # Strategy implementations
    ├── resolver.go     # Configuration resolution & factory
    ├── config.go       # Configuration types
    └── README.md       # Detailed translator documentation
```

## Core Components

### 1. Translator (`translator/`)

**Purpose:** Converts API deployments into xDS resources using composable strategies.

**Key Types:**
- `Translator` interface - Core translation contract
- `CompositeTranslator` - Orchestrates multiple strategies
- `StrategySet` - Collection of strategies for different concerns

**Strategies:**
- **Deployment Strategy** - Cluster generation (basic, canary, blue-green)
- **Route Match Strategy** - Path matching (prefix, exact, regex, header-versioned)
- **Load Balancing Strategy** - LB policies (round-robin, least-request, consistent-hash)
- **Retry Strategy** - Retry configuration (none, conservative, aggressive, custom)
- **Rate Limit Strategy** - Rate limiting policies
- **Observability Strategy** - Tracing, metrics, logging

**Translation Phases:**
1. Generate clusters (DeploymentStrategy)
2. Apply load balancing (LoadBalancingStrategy)
3. Generate routes from IR (RouteMatchStrategy)
4. Apply retry policies (RetryStrategy)
5. Generate listeners (if needed)
6. Apply rate limiting (RateLimitStrategy)
7. Apply observability (ObservabilityStrategy)

> 📖 **Detailed Documentation:** See [translator/README.md](./translator/README.md) for comprehensive translator architecture and strategy documentation.

---

### 2. Resources (`resources/`)

**Purpose:** Helper functions to create Envoy protobuf resources.

#### Cluster (`resources/cluster/`)
Creates upstream service clusters with connection settings.

```go
// Create HTTP cluster
cluster := cluster.CreateCluster("my-api-v1-cluster", "backend.internal", 8080)

// Create HTTPS cluster with TLS
cluster := cluster.CreateClusterWithScheme("secure-api-cluster", "api.example.com", 443, "https")
```

**Features:**
- LOGICAL_DNS discovery type for hostname resolution
- Configurable connection timeout (default 5s)
- TLS/HTTPS support with system CA validation
- Round-robin load balancing by default

#### Route (`resources/route/`)
Creates route configurations for path matching and routing.

```go
// Simple prefix-based route
route := route.CreateRoute("api-route", "my-cluster", "/api/v1")

// Operation-specific route with method matching
route := route.CreateRouteForOperation("/users/{id}", "GET", "user-cluster")
```

**Features:**
- Prefix, exact, and regex path matching
- OpenAPI path parameter support (`{param}` → regex)
- HTTP method matching
- Cluster routing configuration

#### Listener (`resources/listener/`)
Creates listeners that accept incoming traffic.

```go
// Create listener with RDS (Route Discovery Service)
listener := listener.CreateListener("my-listener", "my-route-config", 9095)
```

**Features:**
- HTTP connection manager configuration
- RDS integration for dynamic routing
- Configurable listen port and address
- HTTP/2 codec support

#### Endpoint (`resources/endpoint/`)
Creates endpoint assignments for static clusters.

```go
// Create endpoint for a specific backend
endpoint := endpoint.CreateLbEndpoint("my-cluster", "10.0.0.1", 8080)
```

**Note:** Endpoints are typically not needed for LOGICAL_DNS clusters as Envoy resolves hostnames directly.

---

### 3. Cache (`cache/`)

**Purpose:** Manages versioned xDS configuration snapshots per Envoy node.

**ConfigManager** - Core cache management API:

```go
type ConfigManager struct {
    cache  cachev3.SnapshotCache
    logger *logger.EnvoyLogger
}
```

**Key Methods:**

```go
// Atomic API deployment
func (cm *ConfigManager) DeployAPI(nodeID string, deployment *APIDeployment) error

// Individual resource operations
func (cm *ConfigManager) AddCluster(nodeID, clusterName string, cluster *clusterv3.Cluster) error
func (cm *ConfigManager) AddRoute(nodeID, routeName string, route *routev3.RouteConfiguration) error
func (cm *ConfigManager) AddListener(nodeID, listenerName string, listener *listenerv3.Listener) error
func (cm *ConfigManager) AddEndpoint(nodeID, clusterName string, endpoint *endpointv3.ClusterLoadAssignment) error

// Bulk operations
func (cm *ConfigManager) BulkUpdate(nodeID string, update *BulkResourceUpdate) error

// Snapshot management
func (cm *ConfigManager) GetSnapshot(nodeID string) (*cachev3.Snapshot, error)
func (cm *ConfigManager) UpdateSnapshot(nodeID string, snapshot *cachev3.Snapshot) error

// Node lifecycle
func (cm *ConfigManager) RemoveNode(nodeID string)
func (cm *ConfigManager) ListNodes() []string
```

**Features:**
- **Atomic Updates** - All resources updated together in a single snapshot
- **Versioning** - Automatic snapshot version management
- **Consistency Validation** - Ensures snapshot coherence before deployment
- **Per-Node Isolation** - Each Envoy node has independent configuration
- **Bulk Operations** - Efficient multi-resource updates

**Snapshot Architecture:**
- Each node ID has its own snapshot
- Snapshots contain 4 resource types: Clusters, Routes, Listeners, Endpoints
- Version increments trigger Envoy updates via xDS protocol

---

### 4. Server (`server/`)

**Purpose:** gRPC server implementing Envoy xDS protocol.

**XDSServer** - Control plane gRPC server:

```go
type XDSServer struct {
    grpcServer *grpc.Server
    cache      cachev3.SnapshotCache
    server     serverv3.Server
    logger     *logger.EnvoyLogger
    port       int
}
```

**Key Methods:**

```go
// Lifecycle
func NewXDSServer(port int, keepaliveTime, keepaliveTimeout, keepaliveMinTime time.Duration, keepalivePermitWithoutStream bool, logger *logger.EnvoyLogger) *XDSServer
func (s *XDSServer) Start() error
func (s *XDSServer) Stop()

// Configuration
func (s *XDSServer) InitializeDefaultListener(nodeID string, listenerPort uint32) error
func (s *XDSServer) GetCache() cachev3.SnapshotCache
```

**Features:**
- **ADS (Aggregated Discovery Service)** - Single stream for all resource types
- **gRPC Keepalive** - Configurable connection health checks
- **Default Listener Initialization** - Shared listener for all APIs
- **Graceful Shutdown** - Clean connection termination

**Default Configuration:**
- **Port:** 18000 (configurable)
- **Protocol:** gRPC
- **Service:** ADS (Aggregated Discovery Service)
- **Keepalive:** Configurable timeouts and intervals

---

## Data Flow

### 1. API Deployment Flow

```
1. User uploads API bundle (ZIP)
   └─→ REST API Server (port 8080)

2. Bundle loaded and parsed
   └─→ BundleLoader extracts spec + metadata + IR

3. Deployment created
   └─→ DeploymentService.DeployAPI()

4. Configuration resolution
   └─→ ConfigResolver merges (built-in → gateway → API config)

5. Strategy set creation
   └─→ StrategyFactory creates strategies from resolved config

6. xDS translation
   └─→ CompositeTranslator.Translate(deployment, ir, nodeID)
   ├─→ Phase 1: Generate Clusters
   ├─→ Phase 2: Apply Load Balancing
   ├─→ Phase 3: Generate Routes
   ├─→ Phase 4: Apply Retry Policies
   ├─→ Phase 5: Generate Listeners
   ├─→ Phase 6: Apply Rate Limiting
   └─→ Phase 7: Apply Observability

7. Cache update
   └─→ ConfigManager.DeployAPI(nodeID, xdsResources)
   └─→ New snapshot created with incremented version

8. Envoy notification
   └─→ gRPC stream update pushed to connected Envoy proxies

9. Envoy applies configuration
   └─→ API is live and ready to serve traffic
```

### 2. Envoy Connection Flow

```
1. Envoy starts with bootstrap config
   └─→ Points to FlowC xDS server (localhost:18000)

2. Envoy connects via gRPC
   └─→ XDSServer accepts connection

3. Envoy subscribes to resources
   └─→ ADS stream established

4. Initial configuration
   └─→ XDSServer.InitializeDefaultListener() creates base listener

5. API deployments
   └─→ Each deployment adds clusters + routes to snapshot

6. Incremental updates
   └─→ Envoy receives only changed resources

7. Configuration applied
   └─→ Envoy updates runtime config without restart
```

### 3. Resource Lifecycle

```
Deployment Created
    ↓
Translator Generates XDSResources
    ↓
ConfigManager Creates Snapshot (version N)
    ├─ Clusters:  [cluster1, cluster2, ...]
    ├─ Routes:    [route1, route2, ...]
    ├─ Listeners: [default-listener, ...]
    └─ Endpoints: [endpoint1, ...]
    ↓
Snapshot Validated for Consistency
    ↓
Cache Updated (atomically)
    ↓
Envoy Receives Update via gRPC Stream
    ↓
Envoy Applies Configuration
    ↓
API Live ✓
```

## Quick Start

### Basic Usage Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/flowc-labs/flowc/internal/flowc/xds/cache"
    "github.com/flowc-labs/flowc/internal/flowc/xds/server"
    "github.com/flowc-labs/flowc/internal/flowc/xds/translator"
    "github.com/flowc-labs/flowc/pkg/logger"
)

func main() {
    // 1. Create logger
    log := logger.NewEnvoyLogger(logger.INFO)
    
    // 2. Create xDS server
    xdsServer := server.NewXDSServer(
        18000,                  // port
        30*time.Second,         // keepalive time
        5*time.Second,          // keepalive timeout
        5*time.Second,          // keepalive min time
        true,                   // permit without stream
        log,
    )
    
    // 3. Initialize default listener for node
    nodeID := "envoy-node-1"
    err := xdsServer.InitializeDefaultListener(nodeID, 9095)
    if err != nil {
        panic(err)
    }
    
    // 4. Create config manager
    configManager := cache.NewConfigManager(xdsServer.GetCache(), log)
    
    // 5. Deploy an API
    deployment := getAPIDeployment() // Your deployment
    irAPI := getIR()                  // Your IR
    
    // 6. Translate deployment to xDS
    resolver := translator.NewConfigResolver(nil, log)
    config := resolver.Resolve(deployment.Metadata.Strategies)
    
    factory := translator.NewStrategyFactory(nil, log)
    strategies, _ := factory.CreateStrategySet(config, deployment)
    
    compositeTranslator, _ := translator.NewCompositeTranslator(strategies, nil, log)
    xdsResources, _ := compositeTranslator.Translate(context.Background(), deployment, irAPI, nodeID)
    
    // 7. Deploy to cache
    apiDeployment := &cache.APIDeployment{
        Clusters:  xdsResources.Clusters,
        Endpoints: xdsResources.Endpoints,
        Listeners: xdsResources.Listeners,
        Routes:    xdsResources.Routes,
    }
    
    err = configManager.DeployAPI(nodeID, apiDeployment)
    if err != nil {
        panic(err)
    }
    
    // 8. Start xDS server (blocks)
    log.Info("Starting xDS control plane")
    xdsServer.Start()
}
```

### Creating Resources Directly

```go
import (
    "github.com/flowc-labs/flowc/internal/flowc/xds/cache"
    "github.com/flowc-labs/flowc/internal/flowc/xds/resources/cluster"
    "github.com/flowc-labs/flowc/internal/flowc/xds/resources/route"
)

// Create cluster
myCluster := cluster.CreateClusterWithScheme(
    "my-api-cluster",
    "backend.example.com",
    8080,
    "http",
)

// Create route
myRoute := route.CreateRoute(
    "my-api-route",
    "my-api-cluster",
    "/api/v1",
)

// Add to cache
configManager.AddCluster(nodeID, "my-api-cluster", myCluster)
configManager.AddRoute(nodeID, "my-api-route", myRoute)
```

## Configuration Management

### Snapshot Versioning

Snapshots are versioned to enable Envoy to track configuration changes:

```go
// Version increments automatically
v0 → Initial snapshot (default listener)
v1 → First API deployed (+ clusters, routes)
v2 → Second API deployed (+ more clusters, routes)
v3 → Configuration updated
```

### Multi-Node Support

Each Envoy node has independent configuration:

```go
// Node 1 - Production
configManager.DeployAPI("envoy-prod-1", prodDeployment)

// Node 2 - Staging
configManager.DeployAPI("envoy-staging-1", stagingDeployment)

// Both nodes have different configurations
```

### Atomic Updates

All resource changes happen atomically via snapshots:

```go
// All resources updated together
apiDeployment := &cache.APIDeployment{
    Clusters:  []*clusterv3.Cluster{cluster1, cluster2},
    Routes:    []*routev3.RouteConfiguration{route1},
    Listeners: []*listenerv3.Listener{listener1},
}

// Atomic deployment - all or nothing
configManager.DeployAPI(nodeID, apiDeployment)
```

### Consistency Validation

Snapshots are validated before deployment:

```go
// ConfigManager automatically validates
// - All route cluster references exist in clusters
// - All listener route references exist in routes
// - Resource names are unique
// - Required fields are set

err := configManager.DeployAPI(nodeID, deployment)
if err != nil {
    // Snapshot inconsistent - not deployed
}
```

## Integration Points

### With REST API Server

The xDS package is used by the REST API server to deploy APIs:

```go
// In internal/flowc/server/services/deployments.go

func (s *DeploymentService) DeployAPI(ctx context.Context, req *DeployAPIRequest) error {
    // 1. Load bundle
    bundle := s.bundleLoader.LoadBundle(req.File)
    
    // 2. Create deployment
    deployment := s.CreateDeployment(bundle)
    
    // 3. Translate to xDS
    xdsResources := s.translator.Translate(ctx, deployment, bundle.GetIR(), req.NodeID)
    
    // 4. Deploy to cache
    return s.configManager.DeployAPI(req.NodeID, xdsResources)
}
```

### With Envoy

Envoy connects to the xDS server via bootstrap configuration:

```yaml
# envoy-bootstrap.yaml
node:
  id: envoy-node-1
  cluster: flowc-cluster

dynamic_resources:
  ads_config:
    api_type: GRPC
    transport_api_version: V3
    grpc_services:
      - envoy_grpc:
          cluster_name: xds_cluster

static_resources:
  clusters:
    - name: xds_cluster
      type: STATIC
      connect_timeout: 1s
      load_assignment:
        cluster_name: xds_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: localhost
                      port_value: 18000  # FlowC xDS server
```

## Performance Considerations

### Snapshot Size
- Snapshots grow with each deployment
- Consider periodic cleanup of unused resources
- Future: Implement snapshot compaction

### Update Frequency
- Atomic updates are efficient but complete
- Bulk operations preferred over individual adds
- Envoy only fetches changed resources

### Memory Usage
- One snapshot per node ID in memory
- Snapshots contain all resources for that node
- Consider memory limits for large deployments

## Debugging

### Enable Debug Logging

```go
log := logger.NewEnvoyLogger(logger.DEBUG)
```

### Check Snapshot Contents

```go
snapshot, err := configManager.GetSnapshot(nodeID)
if err != nil {
    log.Error(err)
}

clusters := snapshot.GetResources(resourcev3.ClusterType)
routes := snapshot.GetResources(resourcev3.RouteType)
listeners := snapshot.GetResources(resourcev3.ListenerType)

log.Infof("Clusters: %d, Routes: %d, Listeners: %d", 
    len(clusters), len(routes), len(listeners))
```

### Verify Envoy Connection

Check Envoy admin interface:

```bash
# Envoy config dump
curl http://localhost:9901/config_dump

# Connected control plane
curl http://localhost:9901/clusters | grep xds_cluster
```

## Related Documentation

- **Translator Package:** [translator/README.md](./translator/README.md) - Detailed strategy architecture
- **Configuration System:** `../../config/README.md` - Control plane configuration
- **IR Package:** `../../ir/README.md` - Intermediate Representation for multi-API support
- **Server Package:** `../../server/README.md` - REST API server and deployment services

## Summary

The xDS package is the **heart of FlowC's control plane**, providing:

✅ **Dynamic Configuration** - Translate API deployments to Envoy config  
✅ **Strategy-Based** - Flexible, pluggable deployment patterns  
✅ **Atomic Updates** - Consistent snapshot-based configuration  
✅ **Multi-Node** - Independent config per Envoy instance  
✅ **Production-Ready** - Built on Envoy's official libraries  

This architecture enables FlowC to dynamically configure Envoy proxies for any API deployment pattern while maintaining consistency, reliability, and extensibility.

