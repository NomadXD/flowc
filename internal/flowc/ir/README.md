# IR (Intermediate Representation) Package

## Overview

The IR package provides a unified intermediate representation for different API specification formats. This abstraction layer allows FlowC to work with various API types (REST, gRPC, GraphQL, WebSocket, SSE) in a consistent manner, making it easier to translate them into xDS configurations.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Specifications                        │
├──────────┬──────────┬──────────┬──────────┬─────────────────┤
│ OpenAPI  │ Protobuf │ GraphQL  │ AsyncAPI │  Future Types   │
│  (REST)  │  (gRPC)  │          │ (WS/SSE) │                 │
└─────┬────┴─────┬────┴─────┬────┴─────┬────┴────────┬────────┘
      │          │          │          │             │
      ▼          ▼          ▼          ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│                      IR Parsers                              │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐  │
│  │ OpenAPI  │   gRPC   │ GraphQL  │ AsyncAPI │  Custom  │  │
│  │ Parser   │  Parser  │  Parser  │  Parser  │  Parser  │  │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘  │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│              Unified IR (Intermediate Representation)        │
│                                                              │
│  ┌─────────────┐  ┌──────────┐  ┌───────────────┐         │
│  │   API       │  │ Endpoint │  │  DataModel    │         │
│  │  Metadata   │  │          │  │               │         │
│  └─────────────┘  └──────────┘  └───────────────┘         │
│                                                              │
│  ┌─────────────┐  ┌──────────┐  ┌───────────────┐         │
│  │  Security   │  │  Server  │  │  Extensions   │         │
│  │   Schemes   │  │          │  │               │         │
│  └─────────────┘  └──────────┘  └───────────────┘         │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                   xDS Translator Layer                       │
│          (Converts IR to Envoy xDS Resources)               │
└─────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Core Types (`types.go`)

The IR defines a unified type system that can represent any API type:

#### API Types
- **REST/HTTP**: Traditional REST APIs (OpenAPI)
- **gRPC**: RPC-based APIs using Protocol Buffers
- **GraphQL**: Query language APIs with schema
- **WebSocket**: Full-duplex communication channels
- **SSE**: Server-Sent Events for server push

#### Core Structures

**API**: Top-level representation containing:
- `Metadata`: API information (type, version, description)
- `Endpoints`: All operations/methods available
- `DataModels`: Schema definitions
- `Security`: Security schemes
- `Servers`: Server configurations
- `Extensions`: Custom/spec-specific features

**Endpoint**: Represents a single operation:
- Works across all API types (REST endpoints, gRPC methods, GraphQL queries, etc.)
- Contains request/response specifications
- Includes security, rate limiting, and timeout configurations
- Extensible for protocol-specific features

**DataModel**: Schema/type definitions:
- Unified representation for JSON Schema, Protobuf messages, GraphQL types
- Supports validation rules
- Handles nested and referenced types

### 2. Parser Interface (`parser.go`)

All parsers implement the `Parser` interface:

```go
type Parser interface {
    Parse(ctx context.Context, data []byte) (*API, error)
    SupportedType() APIType
    SupportedFormats() []string
    Validate(ctx context.Context, data []byte) error
}
```

**ParserRegistry**: Manages multiple parsers and routes to the appropriate one based on API type.

### 3. API-Specific Parsers

#### OpenAPI Parser (`openapi_parser.go`) - **IMPLEMENTED**
Converts OpenAPI 2.0 and 3.x specifications to IR:
- Extracts paths and operations → `Endpoint`
- Converts schemas → `DataModel`
- Maps security definitions → `SecurityScheme`
- Preserves validation rules and examples

**Status**: ✅ Fully implemented

#### gRPC Parser (`grpc_parser.go`) - **FUTURE**
Will convert Protobuf service definitions to IR:
- Service methods → `Endpoint` (with streaming types)
- Protobuf messages → `DataModel`
- Method options → Extensions

**Status**: 🔜 Stub implementation with design notes

#### GraphQL Parser (`graphql_parser.go`) - **FUTURE**
Will convert GraphQL schemas to IR:
- Query/Mutation/Subscription fields → `Endpoint`
- Object/Interface/Union types → `DataModel`
- Directives → Extensions

**Status**: 🔜 Stub implementation with design notes

#### AsyncAPI Parser (`asyncapi_parser.go`) - **FUTURE**
Will convert AsyncAPI specifications to IR:
- Channels and operations → `Endpoint`
- Message schemas → `DataModel`
- Protocol bindings → Extensions

**Status**: 🔜 Stub implementation with design notes

## Usage

### Basic Usage

```go
import "github.com/flowc-labs/flowc/internal/flowc/ir"

// Create a parser registry with all supported parsers
registry := ir.DefaultParserRegistry()

// Parse an OpenAPI specification
ctx := context.Background()
openapiData := []byte("...") // Your OpenAPI YAML/JSON

api, err := registry.Parse(ctx, ir.APITypeREST, openapiData)
if err != nil {
    // Handle error
}

// Now you have a unified IR representation
fmt.Printf("API: %s v%s\n", api.Metadata.Title, api.Metadata.Version)
fmt.Printf("Endpoints: %d\n", len(api.Endpoints))
```

### Custom Parser Options

```go
// Create a parser with custom options
parser := ir.NewOpenAPIParser().WithOptions(&ir.ParseOptions{
    Strict:            true,  // Fail on warnings
    Validate:          true,  // Validate spec before parsing
    IncludeExamples:   true,  // Include examples in IR
    IncludeExtensions: true,  // Include vendor extensions
})

api, err := parser.Parse(ctx, openapiData)
```

### Inspecting the IR

```go
// Iterate through endpoints
for _, endpoint := range api.Endpoints {
    fmt.Printf("Endpoint: %s %s %s\n", 
        endpoint.Method, 
        endpoint.Path.Pattern,
        endpoint.Description)
    
    // Check endpoint type
    switch endpoint.Type {
    case ir.EndpointTypeHTTP:
        fmt.Println("  Type: REST/HTTP")
    case ir.EndpointTypeGRPCUnary:
        fmt.Println("  Type: gRPC Unary")
    case ir.EndpointTypeGraphQLQuery:
        fmt.Println("  Type: GraphQL Query")
    case ir.EndpointTypeWebSocket:
        fmt.Println("  Type: WebSocket")
    }
    
    // Access request/response details
    if endpoint.Request != nil && endpoint.Request.Body != nil {
        fmt.Printf("  Request: %s\n", endpoint.Request.Body.Type.BaseType)
    }
    
    for _, response := range endpoint.Responses {
        fmt.Printf("  Response %d: %s\n", 
            response.StatusCode, 
            response.Description)
    }
}
```

### Data Models

```go
// Iterate through data models/schemas
for _, model := range api.DataModels {
    fmt.Printf("Model: %s\n", model.Name)
    
    // Properties for object types
    for _, prop := range model.Properties {
        fmt.Printf("  - %s: %s", prop.Name, prop.Type.BaseType)
        if prop.Required {
            fmt.Print(" (required)")
        }
        fmt.Println()
    }
}
```

## Integration with FlowC

The IR layer is integrated with FlowC's existing architecture:

### 1. Bundle Loading
The bundle loader (`internal/flowc/server/loader/loader.go`):
1. Auto-detects the API specification type from file extensions or `api_type` field in `flowc.yaml`
2. Uses the appropriate parser from the registry to convert to IR
3. Returns a `DeploymentBundle` containing both the original spec and IR representation
4. Extracts spec files: `openapi.yaml`, `*.proto`, `*.graphql`, `asyncapi.yaml`

### 2. Translation to xDS
The translator (`internal/flowc/xds/translator/`):
1. Implements the `Translator` interface with `Translate(ctx, deployment, ir, nodeID)` method
2. Accepts `APIDeployment` (persisted metadata) and `ir.API` (transient IR) as separate parameters
3. Generates xDS resources (`XDSResources`) containing clusters, endpoints, listeners, and routes
4. Uses IR `Endpoint` types to generate protocol-appropriate xDS configurations
5. Provides `Validate()` method to check deployment compatibility with the translator
6. Handles protocol-specific features via IR extensions and endpoint metadata

### 3. Bundle Structure
```
api-bundle.zip
├── flowc.yaml          # FlowC metadata and configuration
└── spec/
    ├── openapi.yaml    # For REST APIs
    ├── service.proto   # For gRPC APIs
    ├── schema.graphql  # For GraphQL APIs
    └── asyncapi.yaml   # For WebSocket/SSE APIs
```

The `flowc.yaml` includes an `api_type` field:
```yaml
name: "my-api"
version: "1.0.0"
api_type: "rest"  # or "grpc", "graphql", "websocket", "sse"
spec_file: "openapi.yaml"  # Points to the spec file
```

## Endpoint Type Mapping

| API Type | Source | IR Endpoint Type |
|----------|--------|------------------|
| REST | OpenAPI paths | `EndpointTypeHTTP` |
| gRPC Unary | Protobuf unary RPC | `EndpointTypeGRPCUnary` |
| gRPC Server Stream | Protobuf server streaming | `EndpointTypeGRPCServerStream` |
| gRPC Client Stream | Protobuf client streaming | `EndpointTypeGRPCClientStream` |
| gRPC Bidirectional | Protobuf bidirectional | `EndpointTypeGRPCBidirectional` |
| GraphQL Query | Query type fields | `EndpointTypeGraphQLQuery` |
| GraphQL Mutation | Mutation type fields | `EndpointTypeGraphQLMutation` |
| GraphQL Subscription | Subscription type fields | `EndpointTypeGraphQLSubscription` |
| WebSocket | AsyncAPI channels | `EndpointTypeWebSocket` |
| SSE | AsyncAPI with SSE binding | `EndpointTypeSSE` |

## Protocol Mapping

| IR Protocol | Envoy Configuration |
|-------------|-------------------|
| `ProtocolHTTP` | HTTP/1.1 listener, HTTP connection manager |
| `ProtocolHTTPS` | HTTP/1.1 with TLS, HTTP connection manager |
| `ProtocolHTTP2` | HTTP/2 listener, HTTP connection manager |
| `ProtocolGRPC` | HTTP/2 with gRPC codec |
| `ProtocolWebSocket` | HTTP/1.1 with WebSocket upgrade |

## Extension Points

The IR supports extensions for API-specific features that don't fit the common model:

```go
// Example: gRPC-specific options
endpoint.Extensions = map[string]interface{}{
    "grpc": map[string]interface{}{
        "timeout":      "30s",
        "retry_policy": map[string]interface{}{
            "max_attempts": 3,
        },
    },
}

// Example: GraphQL-specific features
endpoint.Extensions = map[string]interface{}{
    "graphql": map[string]interface{}{
        "complexity": 100,
        "depth_limit": 10,
    },
}
```

## Future Enhancements

1. **Complete Parser Implementations**
   - Implement gRPC/Protobuf parser
   - Implement GraphQL parser
   - Implement AsyncAPI parser

2. **Validation Framework**
   - Validate IR structure
   - Cross-reference validation (e.g., ensure all refs exist)
   - Protocol-specific validation rules

3. **Transformation Utilities**
   - IR optimization (merge similar endpoints, etc.)
   - IR diffing (compare two versions)
   - IR merging (combine multiple APIs)

4. **Code Generation**
   - Generate client SDKs from IR
   - Generate documentation from IR
   - Generate test suites from IR

5. **Additional API Types**
   - SOAP/WSDL support
   - Apache Thrift support
   - Custom protocol definitions

## Testing

```bash
# Run IR package tests
go test ./internal/flowc/ir/...

# Test with coverage
go test -cover ./internal/flowc/ir/...

# Test specific parser
go test -run TestOpenAPIParser ./internal/flowc/ir/
```

## Contributing

When adding support for a new API type:

1. Define the API type constant in `types.go`
2. Create a new parser file (e.g., `myapi_parser.go`)
3. Implement the `Parser` interface
4. Add the parser to `DefaultParserRegistry()` in `parser.go`
5. Add tests for the parser
6. Update this README with usage examples

## References

- [OpenAPI Specification](https://swagger.io/specification/)
- [gRPC & Protocol Buffers](https://grpc.io/docs/what-is-grpc/introduction/)
- [GraphQL Schema Definition](https://graphql.org/learn/schema/)
- [AsyncAPI Specification](https://www.asyncapi.com/docs/reference/specification/latest)
- [Envoy xDS Protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol)

