# FlowC

FlowC is an Envoy xDS control plane implementation in Go that provides a REST API for deploying APIs via zip files containing OpenAPI and FlowC specifications.

## Features

- **REST API for API Deployment**: Deploy APIs by uploading zip files containing OpenAPI and FlowC specifications
- **xDS Control Plane**: Full Envoy xDS implementation with cluster, endpoint, listener, and route management
- **Intermediate Layer**: Manages API deployments and automatically generates xDS resources
- **File Parsing**: Supports parsing of `openapi.yaml` and `flowc.yaml` configuration files
- **Real-time Updates**: Automatically updates Envoy configuration when APIs are deployed or updated

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST API      │    │  Deployment     │    │   xDS Cache     │
│   (Port 8080)   │───▶│   Service       │───▶│   Manager       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   ZIP Parser    │    │  xDS Handlers   │
                       │ (OpenAPI/FlowC) │    │ (Envoy Config)  │
                       └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │  Envoy Proxy    │
                                              │  (Port 18000)   │
                                              └─────────────────┘
```

## Quick Start

### 1. Build and Run

```bash
# Clone the repository
git clone <repository-url>
cd flowc

# Build the server
go build ./cmd/server

# Run the server
./server
```

The server will start two services:
- **REST API Server**: `http://localhost:8080`
- **xDS Control Plane**: `localhost:18000` (gRPC)

### 2. Deploy an API

Create a zip file with your API specification:

```bash
cd examples/api-deployment
zip api-deployment.zip flowc.yaml openapi.yaml
```

Deploy using the REST API:

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -F "file=@api-deployment.zip" \
  -F "node_id=test-envoy-node" \
  -F "description=My API deployment"
```

### 3. Manage Deployments

```bash
# List all deployments
curl http://localhost:8080/api/v1/deployments

# Get specific deployment
curl http://localhost:8080/api/v1/deployments/{deployment-id}

# Update deployment
curl -X PUT http://localhost:8080/api/v1/deployments/{deployment-id} \
  -F "file=@updated-api-deployment.zip"

# Delete deployment
curl -X DELETE http://localhost:8080/api/v1/deployments/{deployment-id}

# Get deployment statistics
curl http://localhost:8080/api/v1/deployments/stats
```

## Configuration Files

### FlowC Metadata (`flowc.yaml`)

```yaml
name: "my-api"
version: "1.0.0"
description: "My API description"
context: "myapi"  # URL path context

gateway:
  host: "0.0.0.0"
  port: 8080
  tls: false

upstream:
  host: "backend.example.com"
  port: 443
  scheme: "https"
  timeout: "30s"

labels:
  environment: "production"
  team: "api-team"
```

### OpenAPI Specification (`openapi.yaml`)

Standard OpenAPI 3.0 specification file containing your API definition with paths, operations, schemas, etc.

## API Endpoints

### Deployment Management

- `POST /api/v1/deployments` - Deploy new API from zip file
- `GET /api/v1/deployments` - List all deployments
- `GET /api/v1/deployments/{id}` - Get specific deployment
- `PUT /api/v1/deployments/{id}` - Update existing deployment
- `DELETE /api/v1/deployments/{id}` - Delete deployment
- `GET /api/v1/deployments/stats` - Get deployment statistics

### Validation

- `POST /api/v1/validate` - Validate zip file without deploying

### Health & Monitoring

- `GET /health` - Health check endpoint

## Development

### Project Structure

```
flowc/
├── cmd/server/           # Main server application
├── internal/
│   ├── api/             # REST API implementation
│   │   ├── handlers/    # HTTP handlers
│   │   ├── models/      # Data models
│   │   ├── parsers/     # ZIP and YAML parsers
│   │   ├── server/      # API server
│   │   └── service/     # Business logic
│   └── xds/             # xDS control plane
│       ├── cache/       # xDS cache management
│       ├── handlers/    # xDS resource handlers
│       └── server/      # xDS gRPC server
├── pkg/logger/          # Logging utilities
└── examples/            # Example configurations
```

### Dependencies

- **Envoy Go Control Plane**: xDS implementation
- **Gin**: HTTP web framework
- **YAML v3**: YAML parsing
- **Google UUID**: Unique ID generation

### Building

```bash
# Install dependencies
go mod tidy

# Build
go build ./cmd/server

# Run tests
go test ./...
```

## Examples

See the `examples/api-deployment/` directory for:
- Sample `flowc.yaml` and `openapi.yaml` files
- Deployment script (`create-deployment.sh`)
- Usage documentation

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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.