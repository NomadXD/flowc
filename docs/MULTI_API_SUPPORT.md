# Multi-API Type Support in FlowC

## Quick Start Guide

FlowC now supports multiple API types through a unified Intermediate Representation (IR) layer. This guide will help you get started with different API types.

## Supported API Types

| API Type | Status | Specification Format | Use Case |
|----------|--------|---------------------|----------|
| **REST/HTTP** | ✅ Fully Supported | OpenAPI 2.0/3.x | Traditional REST APIs |
| **gRPC** | 🔜 Coming Soon | Protocol Buffers (.proto) | High-performance RPC |
| **GraphQL** | 🔜 Coming Soon | GraphQL SDL (.graphql) | Flexible query APIs |
| **WebSocket** | 🔜 Coming Soon | AsyncAPI | Real-time bidirectional |
| **SSE** | 🔜 Coming Soon | AsyncAPI | Server push notifications |

## Creating an API Bundle

### Step 1: Prepare Your Files

Create a directory with two files:

1. **`flowc.yaml`** - FlowC configuration and metadata
2. **Specification file** - Your API specification (OpenAPI, Protobuf, GraphQL, or AsyncAPI)

### Step 2: Configure flowc.yaml

#### For REST APIs (OpenAPI)

```yaml
name: "my-rest-api"
version: "1.0.0"
description: "My REST API"
context: "api/v1"

# Specify API type (optional, auto-detected from openapi.yaml)
api_type: "rest"

# Specify spec file name (optional, defaults to openapi.yaml)
spec_file: "openapi.yaml"

# Gateway configuration
gateway:
  node_id: "gateway-node-1"
  listener: "http"
  virtual_host:
    name: "api"
    domains: ["*"]

# Upstream service configuration
upstream:
  host: "my-backend-service.local"
  port: 8080
  scheme: "http"
  timeout: "30s"

# Optional: Strategy configuration
strategies:
  deployment:
    type: "basic"
  route_matching:
    type: "prefix"
  load_balancing:
    type: "round-robin"
  retry:
    type: "conservative"
```

#### For gRPC APIs (Coming Soon)

```yaml
name: "my-grpc-api"
version: "1.0.0"
description: "My gRPC API"
context: "grpc/v1"

# Specify API type
api_type: "grpc"

# Specify protobuf file
spec_file: "service.proto"

gateway:
  node_id: "gateway-node-1"
  listener: "http2"  # gRPC uses HTTP/2
  virtual_host:
    name: "grpc-api"
    domains: ["grpc.example.com"]

upstream:
  host: "my-grpc-service.local"
  port: 50051
  scheme: "http2"
  timeout: "30s"
```

#### For GraphQL APIs (Coming Soon)

```yaml
name: "my-graphql-api"
version: "1.0.0"
description: "My GraphQL API"
context: "graphql"

# Specify API type
api_type: "graphql"

# Specify GraphQL schema file
spec_file: "schema.graphql"

gateway:
  node_id: "gateway-node-1"
  listener: "http"
  virtual_host:
    name: "graphql-api"
    domains: ["api.example.com"]

upstream:
  host: "my-graphql-service.local"
  port: 4000
  scheme: "http"
  timeout: "60s"

# GraphQL-specific configuration
extensions:
  graphql:
    query_depth_limit: 10
    query_complexity_limit: 100
```

#### For WebSocket APIs (Coming Soon)

```yaml
name: "my-websocket-api"
version: "1.0.0"
description: "My WebSocket API"
context: "ws"

# Specify API type
api_type: "websocket"

# Specify AsyncAPI file
spec_file: "asyncapi.yaml"

gateway:
  node_id: "gateway-node-1"
  listener: "http"  # WebSocket upgrades from HTTP
  virtual_host:
    name: "ws-api"
    domains: ["ws.example.com"]

upstream:
  host: "my-websocket-service.local"
  port: 8080
  scheme: "http"
  timeout: "300s"  # Long timeout for persistent connections

# WebSocket-specific configuration
extensions:
  websocket:
    max_frame_size: 65536
    idle_timeout: "600s"
```

### Step 3: Create Your API Specification

#### REST - OpenAPI Specification

Create `openapi.yaml`:

```yaml
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0

paths:
  /hello:
    get:
      operationId: sayHello
      summary: Say hello
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: string
```

#### gRPC - Protocol Buffers (Coming Soon)

Create `service.proto`:

```protobuf
syntax = "proto3";

package myapi.v1;

service MyService {
  rpc SayHello (HelloRequest) returns (HelloResponse);
}

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}
```

#### GraphQL - Schema (Coming Soon)

Create `schema.graphql`:

```graphql
type Query {
  hello(name: String!): String!
}

type Mutation {
  updateGreeting(message: String!): Boolean!
}
```

#### WebSocket - AsyncAPI (Coming Soon)

Create `asyncapi.yaml`:

```yaml
asyncapi: 2.6.0
info:
  title: My WebSocket API
  version: 1.0.0

channels:
  /messages:
    subscribe:
      message:
        payload:
          type: object
          properties:
            text:
              type: string
```

### Step 4: Create the Bundle

Package your files into a ZIP:

```bash
zip api-bundle.zip flowc.yaml openapi.yaml
```

Or for other API types:

```bash
# gRPC
zip api-bundle.zip flowc.yaml service.proto

# GraphQL
zip api-bundle.zip flowc.yaml schema.graphql

# WebSocket
zip api-bundle.zip flowc.yaml asyncapi.yaml
```

### Step 5: Deploy

Deploy using the FlowC API:

```bash
curl -X POST http://flowc-server:8080/api/v1/deployments \
  -H "Content-Type: multipart/form-data" \
  -F "file=@api-bundle.zip" \
  -F "description=My API deployment"
```

## Auto-Detection

FlowC can automatically detect your API type based on the specification file in your bundle:

| File Found | Detected API Type |
|------------|-------------------|
| `openapi.yaml` or `swagger.yaml` | REST |
| `*.proto` | gRPC |
| `*.graphql` or `*.gql` | GraphQL |
| `asyncapi.yaml` | WebSocket/SSE |

**Example without explicit `api_type`:**

```yaml
name: "my-api"
version: "1.0.0"
context: "api"
# api_type not specified - will be auto-detected

gateway:
  # ... gateway config ...

upstream:
  # ... upstream config ...
```

If you include `openapi.yaml` in your bundle, FlowC will automatically detect it as a REST API.

## Advanced Configuration

### Specifying Custom Spec File Names

If your specification file has a non-standard name:

```yaml
name: "my-api"
api_type: "rest"
spec_file: "custom-api-spec.yaml"  # Custom name
```

### Multiple Endpoints

You can deploy the same API to multiple gateways by creating multiple deployments with different gateway configurations.

### Environment-Specific Configuration

Create different bundles for different environments:

```
my-api-dev.zip
  ├── flowc.yaml (dev configuration)
  └── openapi.yaml

my-api-prod.zip
  ├── flowc.yaml (prod configuration)
  └── openapi.yaml
```

## Migration Guide

### From OpenAPI-Only to Multi-API

If you have existing OpenAPI deployments:

✅ **No changes required** - Your existing bundles continue to work  
✅ **Auto-detection** - OpenAPI files are automatically detected  
✅ **Optional explicit type** - Add `api_type: "rest"` if desired  

**Example existing bundle:**
```
api-bundle.zip
  ├── flowc.yaml (no api_type field)
  └── openapi.yaml
```

**Still works!** FlowC will:
1. Detect `openapi.yaml` in the bundle
2. Automatically set API type to `rest`
3. Parse using OpenAPI parser
4. Generate xDS resources as before

## Validation

FlowC validates your API specification during deployment:

### REST APIs
- OpenAPI specification is validated against OpenAPI 3.x schema
- Required fields are checked
- Schema references are validated

### Future API Types
- gRPC: Protobuf syntax validation
- GraphQL: Schema validation
- WebSocket: AsyncAPI specification validation

## Troubleshooting

### "No supported API specification file found"

**Problem:** FlowC couldn't find or identify your specification file.

**Solutions:**
1. Ensure your bundle includes a specification file
2. Use standard file names (`openapi.yaml`, `service.proto`, etc.)
3. Or explicitly specify `api_type` and `spec_file` in `flowc.yaml`

### "Failed to parse specification"

**Problem:** Your specification file has syntax errors or is invalid.

**Solutions:**
1. Validate your OpenAPI spec using tools like [Swagger Editor](https://editor.swagger.io/)
2. Check for syntax errors in your YAML/JSON
3. Ensure all `$ref` references are valid
4. Verify required fields are present

### "API type not supported"

**Problem:** You specified an API type that isn't implemented yet.

**Solutions:**
1. Check the supported API types list (REST is currently fully supported)
2. For gRPC, GraphQL, WebSocket - these are coming soon
3. Use REST/OpenAPI for now

## Examples

See the [examples/ir](../examples/ir/) directory for complete working examples:

- **REST API Example** - Complete OpenAPI-based user management API
- **gRPC Example** - Design for future gRPC support
- **GraphQL Example** - Design for future GraphQL support
- **WebSocket Example** - Design for future WebSocket support

## Best Practices

1. **Always specify versions** - Include version in both `flowc.yaml` and your spec
2. **Use semantic versioning** - Follow semver (1.0.0, 1.1.0, 2.0.0)
3. **Document your APIs** - Use descriptions and summaries in your spec
4. **Validate before deploying** - Test your specs with validators
5. **Use tags** - Organize endpoints with tags (OpenAPI) or groups
6. **Include examples** - Add example requests/responses to your spec
7. **Specify security** - Define authentication/authorization clearly

## Architecture

For deep technical details, see:
- [IR Package README](../internal/flowc/ir/README.md)

## Roadmap

### Current (Phase 1) - ✅ Complete
- ✅ IR foundation
- ✅ OpenAPI/REST support
- ✅ Auto-detection
- ✅ Backward compatibility

### Next (Phase 2) - 🔄 In Progress
- 🔄 Translator updates to use IR
- 🔄 Protocol detection in strategies

### Coming Soon (Phase 3)
- 🔜 gRPC parser implementation
- 🔜 GraphQL parser implementation
- 🔜 AsyncAPI parser implementation
- 🔜 End-to-end tests for each type

### Future (Phase 4+)
- 🔮 SOAP/WSDL support
- 🔮 Apache Thrift support
- 🔮 Custom protocol definitions
- 🔮 Multi-spec bundling
- 🔮 Client SDK generation

## Getting Help

- **Documentation:** Check the comprehensive docs in `/docs`
- **Examples:** Review working examples in `/examples/ir`
- **Issues:** Report issues on GitHub
- **Community:** Join discussions for support

## Contributing

Want to add support for a new API type?

1. Implement the `Parser` interface in `internal/flowc/ir`
2. Add your parser to the registry
3. Create tests
4. Add examples and documentation
5. Submit a pull request

See [IR Package README](../internal/flowc/ir/README.md) for details.

## Summary

FlowC's multi-API support makes it easy to deploy any type of API through a unified workflow:

1. **Create** your API specification (OpenAPI, Protobuf, GraphQL, AsyncAPI)
2. **Configure** FlowC with `flowc.yaml`
3. **Bundle** both files into a ZIP
4. **Deploy** through FlowC's API
5. **Manage** through the same interface regardless of API type

The IR layer handles the complexity of different formats, giving you a consistent experience across all API types.

---

**Ready to get started?** Check out the [REST example](../examples/ir/rest-example/) for a complete working example!

