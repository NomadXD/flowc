# FlowC XDS Control Plane

A basic XDS (eXtensible Discovery Service) control plane implementation for Envoy Proxy using Go.

## Overview

This project provides a foundation for building an XDS control plane that can serve configuration to Envoy proxy instances. It includes:

- gRPC server with XDS service registration
- Snapshot cache for managing configuration
- Basic resource handlers for CDS, EDS, LDS, and RDS
- Configuration management utilities

## Project Structure

```
flowc/
├── cmd/server/           # Main application entry point
├── internal/
│   └── xds/
│       ├── server/       # XDS server implementation
│       ├── cache/        # Configuration cache management
│       └── handlers/     # Basic XDS resource handlers
├── go.mod               # Go module dependencies
└── README.md            # This file
```

## Features

- **XDS Server**: Complete gRPC server with XDS service registration
- **Snapshot Cache**: Manages configuration snapshots for different Envoy nodes
- **Configuration Manager**: Utilities for adding/updating cluster, endpoint, listener, and route configurations
- **Basic Handlers**: Placeholder implementations for creating XDS resources
- **Graceful Shutdown**: Proper signal handling for clean server shutdown

## Getting Started

### Prerequisites

- Go 1.23.0 or later
- Envoy Proxy (for testing)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd flowc
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the server:
```bash
go build -o bin/xds-server ./cmd/server
```

4. Run the server:
```bash
./bin/xds-server
```

The server will start on port 18000 by default.

### Configuration

The XDS server can be configured by modifying the `main.go` file:

- **Port**: Change the port number in `NewXDSServer(18000)`
- **Logging**: Adjust log level in the logger configuration
- **Resources**: Add actual Envoy configuration resources using the handlers

### Example Usage

```go
// Create XDS server
xdsServer := server.NewXDSServer(18000)

// Create configuration manager
configManager := cache.NewConfigManager(xdsServer.GetCache(), xdsServer.GetLogger())

// Create handlers
handlers := handlers.NewXDSHandlers(xdsServer.GetLogger())

// Add a cluster
cluster := handlers.CreateBasicCluster("my-cluster", "my-service", 8080)
configManager.AddCluster("node-1", "my-cluster", cluster)

// Add an endpoint
endpoint := handlers.CreateBasicEndpoint("my-cluster", "127.0.0.1", 8080)
configManager.AddEndpoint("node-1", "my-cluster", endpoint)

// Start the server
xdsServer.Start()
```

## Envoy Configuration

To connect Envoy to this XDS server, use the following configuration:

```yaml
dynamic_resources:
  cds_config:
    resource_api_version: V3
    api_config_source:
      api_type: GRPC
      transport_api_version: V3
      grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster
  lds_config:
    resource_api_version: V3
    api_config_source:
      api_type: GRPC
      transport_api_version: V3
      grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster

static_resources:
  clusters:
  - name: xds_cluster
    connect_timeout: 0.25s
    type: LOGICAL_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: 18000
    http2_protocol_options: {}
```

## Development

### Adding New Resource Types

1. Create a new resource struct in `handlers/handlers.go`
2. Implement the `GetName()` and `GetType()` methods
3. Add a creation method in the `XDSHandlers` struct
4. Update the cache manager to handle the new resource type

### Testing

To test the XDS server:

1. Start the XDS server
2. Configure Envoy with the XDS server address
3. Send requests through Envoy and verify configuration updates

## Dependencies

- `github.com/envoyproxy/go-control-plane`: Envoy control plane Go library
- `google.golang.org/grpc`: gRPC implementation
- `log/slog`: Go's built-in structured logging (no external dependency)

## License

See LICENSE file for details.
