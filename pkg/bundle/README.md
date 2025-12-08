# FlowC Bundle Package

The `bundle` package provides utilities for creating, validating, and extracting FlowC API bundles - ZIP files containing `flowc.yaml` configuration and API specifications for REST, gRPC, GraphQL, WebSocket, and SSE APIs.

## Overview

A FlowC bundle is a ZIP archive containing:
- `flowc.yaml` - FlowC metadata and configuration
- API Specification - One of:
  - `openapi.yaml` - OpenAPI specification for REST APIs
  - `*.proto` - Protocol Buffer files for gRPC APIs
  - `*.graphql` or `*.gql` - GraphQL schema for GraphQL APIs
  - `asyncapi.yaml` - AsyncAPI specification for WebSocket/SSE APIs

This package is used by:
- **FlowC Server** - To extract and parse uploaded bundles
- **FlowC CLI (`flowctl`)** - To create bundles from local files
- **FlowC Kubernetes Operator** - To create bundles from CRDs

## Installation

```go
import "github.com/flowc-labs/flowc/pkg/bundle"
```

## Supported API Types

| API Type | Specification Format | Detection Pattern | Status |
|----------|---------------------|-------------------|---------|
| **REST/HTTP** | OpenAPI 2.0/3.x | `openapi.yaml`, `swagger.yaml`, `*.json` | ✅ Fully Supported |
| **gRPC** | Protocol Buffers | `*.proto` | ✅ Supported (parser in progress) |
| **GraphQL** | GraphQL SDL | `*.graphql`, `*.gql` | ✅ Supported (parser in progress) |
| **WebSocket** | AsyncAPI | `asyncapi.yaml` | ✅ Supported (parser in progress) |
| **SSE** | AsyncAPI | `asyncapi.yaml` | ✅ Supported (parser in progress) |

## Usage

### Creating REST API Bundles

```go
package main

import (
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read YAML files
    flowcYAML, _ := os.ReadFile("flowc.yaml")
    openapiYAML, _ := os.ReadFile("openapi.yaml")
    
    // Create ZIP bundle
    zipData, err := bundle.CreateZip(flowcYAML, openapiYAML, "openapi.yaml")
    if err != nil {
        panic(err)
    }
    
    // Save to file
    os.WriteFile("rest-api-bundle.zip", zipData, 0644)
}
```

### Creating gRPC API Bundles

```go
package main

import (
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read YAML files
    flowcYAML, _ := os.ReadFile("flowc.yaml")
    protoData, _ := os.ReadFile("user_service.proto")
    
    // Create ZIP bundle
    zipData, err := bundle.CreateZip(flowcYAML, protoData, "user_service.proto")
    if err != nil {
        panic(err)
    }
    
    // Save to file
    os.WriteFile("grpc-bundle.zip", zipData, 0644)
}
```

### Creating GraphQL API Bundles

```go
package main

import (
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read YAML files
    flowcYAML, _ := os.ReadFile("flowc.yaml")
    schemaData, _ := os.ReadFile("schema.graphql")
    
    // Create ZIP bundle
    zipData, err := bundle.CreateZip(flowcYAML, schemaData, "schema.graphql")
    if err != nil {
        panic(err)
    }
    
    // Save to file
    os.WriteFile("graphql-bundle.zip", zipData, 0644)
}
```

### Creating WebSocket API Bundles

```go
package main

import (
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read YAML files
    flowcYAML, _ := os.ReadFile("flowc.yaml")
    asyncapiData, _ := os.ReadFile("asyncapi.yaml")
    
    // Create ZIP bundle
    zipData, err := bundle.CreateZip(flowcYAML, asyncapiData, "asyncapi.yaml")
    if err != nil {
        panic(err)
    }
    
    // Save to file
    os.WriteFile("websocket-bundle.zip", zipData, 0644)
}
```

### Creating Bundles from Metadata (Alternative Approach)

If you're programmatically building metadata (e.g., from a CRD or database), you can use the `NewBundle()` + `CreateZipFromBundle()` pattern:

```go
package main

import (
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
    "github.com/flowc-labs/flowc/pkg/types"
)

func main() {
    // Build metadata programmatically
    metadata := &types.FlowCMetadata{
        Name:    "user-service",
        Version: "v1.0.0",
        Context: "grpc/v1",
        APIType: "grpc",
        Gateway: types.GatewayConfig{
            NodeID:   "gateway-1",
            Listener: "http2",
            VirtualHost: types.VirtualHostConfig{
                Domains: []string{"grpc.example.com"},
            },
        },
        Upstream: types.UpstreamConfig{
            Host:   "user-service.local",
            Port:   50051,
            Scheme: "http2",
        },
    }
    
    // Read spec file
    protoData, _ := os.ReadFile("user_service.proto")
    
    // Create bundle struct
    b := bundle.NewBundle(metadata, protoData, "user_service.proto", "grpc")
    
    // Create ZIP from bundle
    zipData, err := bundle.CreateZipFromBundle(b)
    if err != nil {
        panic(err)
    }
    
    // Save bundle
    os.WriteFile("grpc-bundle.zip", zipData, 0644)
}
```

**When to use each approach:**

| Approach | Use When |
|----------|----------|
| `CreateZip()` | You have `flowc.yaml` as bytes (reading from file, HTTP request, etc.) |
| `NewBundle()` + `CreateZipFromBundle()` | You're building metadata programmatically (from CRD, database, etc.) |

Both approaches work for **all API types** equally!

### Validating a Bundle

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read ZIP file
    zipData, _ := os.ReadFile("api-bundle.zip")
    
    // Validate bundle (supports all API types)
    if err := bundle.ValidateZip(zipData); err != nil {
        fmt.Printf("Invalid bundle: %v\n", err)
        return
    }
    
    fmt.Println("Bundle is valid!")
}
```

### Extracting Files from a Bundle

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read ZIP file
    zipData, _ := os.ReadFile("api-bundle.zip")
    
    // Extract files (works for all API types)
    flowcYAML, specInfo, err := bundle.ExtractFiles(zipData, "")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("FlowC YAML:\n%s\n\n", string(flowcYAML))
    fmt.Printf("API Type: %s\n", specInfo.APIType)
    fmt.Printf("Spec File: %s\n", specInfo.FileName)
    fmt.Printf("Spec Data:\n%s\n", string(specInfo.Data))
}
```

### Detecting API Type

```go
package main

import (
    "fmt"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Detect API type from filename
    apiType := bundle.DetectAPIType("openapi.yaml")
    fmt.Printf("openapi.yaml -> %s\n", apiType) // Output: rest
    
    apiType = bundle.DetectAPIType("service.proto")
    fmt.Printf("service.proto -> %s\n", apiType) // Output: grpc
    
    apiType = bundle.DetectAPIType("schema.graphql")
    fmt.Printf("schema.graphql -> %s\n", apiType) // Output: graphql
    
    apiType = bundle.DetectAPIType("asyncapi.yaml")
    fmt.Printf("asyncapi.yaml -> %s\n", apiType) // Output: asyncapi
}
```

### Checking if File is a Spec File

```go
package main

import (
    "fmt"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Check if file is any supported spec file
    isSpec := bundle.IsSpecFile("openapi.yaml")
    fmt.Printf("openapi.yaml is spec: %v\n", isSpec) // true
    
    isSpec = bundle.IsSpecFile("readme.md")
    fmt.Printf("readme.md is spec: %v\n", isSpec) // false
    
    // Check specific API types
    isREST := bundle.IsRESTSpecFile("openapi.yaml")
    isGRPC := bundle.IsGRPCSpecFile("service.proto")
    isGraphQL := bundle.IsGraphQLSpecFile("schema.graphql")
    isAsyncAPI := bundle.IsAsyncAPISpecFile("asyncapi.yaml")
}
```

### Getting Spec File Information

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    zipData, _ := os.ReadFile("api-bundle.zip")
    
    // Get spec file info (auto-detect)
    specInfo, err := bundle.GetSpecFileInfo(zipData, "")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Detected API Type: %s\n", specInfo.APIType)
    fmt.Printf("Spec File Name: %s\n", specInfo.FileName)
    fmt.Printf("Spec Data Size: %d bytes\n", len(specInfo.Data))
    
    // Get specific spec file (if bundle has multiple)
    specInfo, err = bundle.GetSpecFileInfo(zipData, "openapi.yaml")
    if err != nil {
        panic(err)
    }
}
```

### Listing Files in a Bundle

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    zipData, _ := os.ReadFile("api-bundle.zip")
    
    files, err := bundle.ListFiles(zipData)
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Files in bundle:")
    for _, file := range files {
        fmt.Printf("  - %s\n", file)
    }
}
```

## API Reference

### Constants

```go
const (
    FlowCFileName = "flowc.yaml"    // Standard FlowC config filename
    MaxBundleSize = 100 * 1024 * 1024 // Maximum bundle size (100MB)
)
```

### Spec File Patterns

```go
var (
    // REST/OpenAPI specification files
    RESTSpecFiles = []string{"openapi.yaml", "openapi.yml", "swagger.yaml", "swagger.yml", "openapi.json", "swagger.json"}

    // gRPC Protocol Buffer files
    GRPCSpecExtensions = []string{".proto"}

    // GraphQL schema files
    GraphQLSpecExtensions = []string{".graphql", ".gql"}

    // AsyncAPI specification files (WebSocket, SSE)
    AsyncAPISpecFiles = []string{"asyncapi.yaml", "asyncapi.yml", "asyncapi.json"}
)
```

### Types

#### Bundle

```go
type Bundle struct {
    FlowCMetadata *types.FlowCMetadata
    SpecData      []byte // Raw API specification data
    SpecFileName  string // Name of the specification file
    APIType       string // Detected or specified API type
}
```

Represents a FlowC API bundle with multi-API type support.

#### SpecFileInfo

```go
type SpecFileInfo struct {
    FileName string // Name of the spec file in the bundle
    APIType  string // Detected API type based on file pattern
    Data     []byte // Raw file content
}
```

Contains information about a detected specification file.

### Functions

#### CreateZip

```go
func CreateZip(flowcYAML, specData []byte, specFileName string) ([]byte, error)
```

Creates a ZIP bundle with any API specification type.

**Parameters:**
- `flowcYAML` - FlowC configuration as YAML bytes
- `specData` - API specification as bytes (OpenAPI, Proto, GraphQL, AsyncAPI)
- `specFileName` - Name for the specification file in the bundle

**Returns:**
- ZIP file as bytes
- Error if creation fails

**Example:**
```go
zipData, err := bundle.CreateZip(flowcYAML, openapiYAML, "openapi.yaml")
zipData, err := bundle.CreateZip(flowcYAML, protoData, "service.proto")
zipData, err := bundle.CreateZip(flowcYAML, graphqlSchema, "schema.graphql")
```

#### CreateZipFromBundle

```go
func CreateZipFromBundle(bundle *Bundle) ([]byte, error)
```

Creates a ZIP bundle from a Bundle struct. Supports all API types.

**Example:**
```go
bundle := bundle.NewBundle(metadata, specData, "openapi.yaml", "rest")
zipData, err := bundle.CreateZipFromBundle(bundle)
```

#### ValidateZip

```go
func ValidateZip(zipData []byte) error
```

Validates that a ZIP file is a valid FlowC bundle.

**Checks:**
- ZIP signature is valid
- Bundle size is within limits
- Contains `flowc.yaml` (or `flowc.yml`)
- Contains at least one supported API specification file

**Example:**
```go
if err := bundle.ValidateZip(zipData); err != nil {
    log.Fatalf("Invalid bundle: %v", err)
}
```

#### ExtractFiles

```go
func ExtractFiles(zipData []byte, preferredSpecFile string) (flowcYAML []byte, specInfo *SpecFileInfo, err error)
```

Extracts flowc.yaml and the API specification file from a ZIP bundle.

**Parameters:**
- `zipData` - ZIP file as bytes
- `preferredSpecFile` - Optional: Specific spec file to extract if multiple are present (use empty string "" for auto-detection)

**Returns:**
- `flowcYAML` - FlowC configuration as bytes
- `specInfo` - Information about the detected spec file
- `err` - Error if extraction fails

**Example:**
```go
// Auto-detect spec file
flowcYAML, specInfo, err := bundle.ExtractFiles(zipData, "")

// Extract specific spec file
flowcYAML, specInfo, err := bundle.ExtractFiles(zipData, "openapi.yaml")
```

#### GetSpecFileInfo

```go
func GetSpecFileInfo(zipData []byte, preferredSpecFile string) (*SpecFileInfo, error)
```

Extracts information about the API specification file in a bundle.

**Parameters:**
- `zipData` - ZIP file as bytes
- `preferredSpecFile` - Optional: Specific spec file to get info for (use empty string "" for auto-detection)

**Returns:**
- Spec file information (name, type, data)
- Error if not found

**Example:**
```go
specInfo, err := bundle.GetSpecFileInfo(zipData, "")
fmt.Printf("Found %s API spec: %s\n", specInfo.APIType, specInfo.FileName)
```

#### DetectAPIType

```go
func DetectAPIType(fileName string) string
```

Detects the API type based on the specification file name.

**Returns:** `"rest"`, `"grpc"`, `"graphql"`, `"asyncapi"`, or `""` if unknown

**Example:**
```go
apiType := bundle.DetectAPIType("openapi.yaml")  // "rest"
apiType := bundle.DetectAPIType("service.proto") // "grpc"
```

#### IsSpecFile / IsRESTSpecFile / IsGRPCSpecFile / IsGraphQLSpecFile / IsAsyncAPISpecFile

```go
func IsSpecFile(fileName string) bool
func IsRESTSpecFile(fileName string) bool
func IsGRPCSpecFile(fileName string) bool
func IsGraphQLSpecFile(fileName string) bool
func IsAsyncAPISpecFile(fileName string) bool
```

Check if a file matches a specific API specification pattern.

**Example:**
```go
if bundle.IsRESTSpecFile("openapi.yaml") {
    // Handle REST API
}

if bundle.IsGRPCSpecFile("service.proto") {
    // Handle gRPC API
}
```

#### NewBundle

```go
func NewBundle(metadata *types.FlowCMetadata, specData []byte, specFileName, apiType string) *Bundle
```

Creates a new Bundle from metadata and specification data for any API type.

**Parameters:**
- `metadata` - FlowC metadata
- `specData` - API specification as bytes
- `specFileName` - Name of the specification file
- `apiType` - API type (`"rest"`, `"grpc"`, `"graphql"`, `"websocket"`, `"sse"`)

**Example:**
```go
bundle := bundle.NewBundle(metadata, openapiData, "openapi.yaml", "rest")
bundle := bundle.NewBundle(metadata, protoData, "service.proto", "grpc")
bundle := bundle.NewBundle(metadata, graphqlSchema, "schema.graphql", "graphql")
```

#### ListFiles

```go
func ListFiles(zipData []byte) ([]string, error)
```

Lists all files in a bundle.

**Example:**
```go
files, err := bundle.ListFiles(zipData)
for _, file := range files {
    fmt.Println(file)
}
```

## Error Handling

The package returns descriptive errors for common issues:

- `"flowc.yaml content is empty"` - FlowC YAML is empty
- `"specification file content is empty"` - Spec file is empty
- `"specification file name is empty"` - Spec file name not provided
- `"zip data is empty"` - No ZIP data provided
- `"bundle size exceeds maximum allowed size"` - Bundle too large (>100MB)
- `"invalid ZIP file: missing ZIP signature"` - Not a valid ZIP file
- `"bundle missing required file: flowc.yaml"` - flowc.yaml not found
- `"bundle missing API specification file"` - No supported spec file found
- `"no API specification file found in bundle"` - No spec files detected
- `"specified spec file X not found in bundle"` - Requested spec file doesn't exist

## Testing

Run tests:

```bash
go test ./pkg/bundle/...
```

Run tests with coverage:

```bash
go test -cover ./pkg/bundle/...
```

Run tests verbosely:

```bash
go test -v ./pkg/bundle/...
```

## Best Practices

1. **Always validate bundles** before processing them
2. **Specify API type explicitly** in `flowc.yaml` and when creating bundles
3. **Use descriptive spec file names** that indicate the API type
4. **Check file sizes** to prevent memory issues
5. **Handle errors gracefully** with descriptive messages
6. **Support alternative filenames** (`.yml` vs `.yaml`, `.gql` vs `.graphql`)
7. **Limit maximum bundle size** to prevent abuse (100MB default)
8. **Handle multiple spec files** gracefully with `preferredSpecFile` parameter

## Examples

See the [examples/ir](../../examples/ir/) directory for complete working examples of different API types.

## Related Documentation

- [Multi-API Support Guide](../../docs/MULTI_API_SUPPORT.md) - User guide for different API types
- [IR Architecture](../../docs/IR_ARCHITECTURE.md) - Intermediate Representation design
- [IR Package README](../../internal/flowc/ir/README.md) - Parser implementation details

## License

See the LICENSE file in the repository root.
