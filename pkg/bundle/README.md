# FlowC Bundle Package

The `bundle` package provides utilities for creating, validating, and extracting FlowC API bundles (ZIP files containing `flowc.yaml` and `openapi.yaml`).

## Overview

A FlowC bundle is a ZIP archive containing:
- `flowc.yaml` - FlowC metadata and configuration
- `openapi.yaml` - OpenAPI specification for the API

This package is used by:
- **FlowC Server** - To extract and parse uploaded bundles
- **FlowC CLI (`flowctl`)** - To create bundles from local files
- **FlowC Kubernetes Operator** - To create bundles from CRDs

## Installation

```go
import "github.com/flowc-labs/flowc/pkg/bundle"
```

## Usage

### Creating a Bundle from YAML Bytes

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
)

func main() {
    // Read YAML files
    flowcYAML, _ := os.ReadFile("flowc.yaml")
    openapiYAML, _ := os.ReadFile("openapi.yaml")
    
    // Create ZIP bundle
    zipData, err := bundle.CreateZip(flowcYAML, openapiYAML)
    if err != nil {
        panic(err)
    }
    
    // Save to file
    os.WriteFile("api-bundle.zip", zipData, 0644)
    fmt.Println("Bundle created successfully!")
}
```

### Creating a Bundle from FlowC Metadata

```go
package main

import (
    "os"
    
    "github.com/flowc-labs/flowc/pkg/bundle"
    "github.com/flowc-labs/flowc/pkg/types"
)

func main() {
    // Create metadata
    metadata := &types.FlowCMetadata{
        Name:    "petstore-api",
        Version: "v1.0.0",
        Context: "petstore",
        Gateway: types.GatewayConfig{
            Mediation: &types.MediationConfig{
                JWT: &types.JWTConfig{
                    Enabled: true,
                    Providers: map[string]types.JWTProvider{
                        "auth0": {
                            Issuer: "https://auth.petstore.com",
                            Audiences: []string{"petstore-api"},
                        },
                    },
                },
            },
        },
        Upstream: types.UpstreamConfig{
            Host: "petstore-service",
            Port: 8080,
        },
    }
    
    // Read OpenAPI spec
    openapiYAML, _ := os.ReadFile("openapi.yaml")
    
    // Create bundle
    b := bundle.NewBundle(metadata, openapiYAML)
    zipData, err := bundle.CreateZipFromBundle(b)
    if err != nil {
        panic(err)
    }
    
    // Save bundle
    os.WriteFile("petstore-bundle.zip", zipData, 0644)
}
```

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
    
    // Validate bundle
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
    
    // Extract files
    flowcYAML, openapiYAML, err := bundle.ExtractFiles(zipData)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("FlowC YAML:\n%s\n\n", string(flowcYAML))
    fmt.Printf("OpenAPI YAML:\n%s\n", string(openapiYAML))
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
    FlowCFileName   = "flowc.yaml"    // Standard FlowC config filename
    OpenAPIFileName = "openapi.yaml"  // Standard OpenAPI spec filename
    MaxBundleSize   = 100 * 1024 * 1024 // Maximum bundle size (100MB)
)
```

### Types

#### Bundle

```go
type Bundle struct {
    FlowCMetadata *types.FlowCMetadata
    OpenAPIData   []byte
}
```

Represents a FlowC API bundle.

### Functions

#### CreateZip

```go
func CreateZip(flowcYAML, openapiYAML []byte) ([]byte, error)
```

Creates a ZIP bundle from raw YAML bytes.

**Parameters:**
- `flowcYAML` - FlowC configuration as YAML bytes
- `openapiYAML` - OpenAPI specification as YAML bytes

**Returns:**
- ZIP file as bytes
- Error if creation fails

#### CreateZipFromBundle

```go
func CreateZipFromBundle(bundle *Bundle) ([]byte, error)
```

Creates a ZIP bundle from a Bundle struct.

**Parameters:**
- `bundle` - Bundle containing metadata and OpenAPI data

**Returns:**
- ZIP file as bytes
- Error if creation fails

#### ValidateZip

```go
func ValidateZip(zipData []byte) error
```

Validates that a ZIP file is a valid FlowC bundle.

**Checks:**
- ZIP signature is valid
- Bundle size is within limits
- Contains `flowc.yaml` (or `flowc.yml`)
- Contains `openapi.yaml` (or `openapi.yml`, `swagger.yaml`, `swagger.yml`)

**Parameters:**
- `zipData` - ZIP file as bytes

**Returns:**
- Error if validation fails, nil if valid

#### ExtractFiles

```go
func ExtractFiles(zipData []byte) (flowcYAML, openapiYAML []byte, err error)
```

Extracts `flowc.yaml` and `openapi.yaml` from a bundle.

**Parameters:**
- `zipData` - ZIP file as bytes

**Returns:**
- `flowcYAML` - FlowC configuration as bytes
- `openapiYAML` - OpenAPI specification as bytes
- `err` - Error if extraction fails

#### ListFiles

```go
func ListFiles(zipData []byte) ([]string, error)
```

Lists all files in a bundle.

**Parameters:**
- `zipData` - ZIP file as bytes

**Returns:**
- List of filenames
- Error if listing fails

#### NewBundle

```go
func NewBundle(metadata *types.FlowCMetadata, openapiData []byte) *Bundle
```

Creates a new Bundle from metadata and OpenAPI data.

**Parameters:**
- `metadata` - FlowC metadata
- `openapiData` - OpenAPI specification as bytes

**Returns:**
- New Bundle instance

## Usage in FlowC Components

### FlowC CLI (flowctl)

```go
// flowctl deploy command
func deployCommand(zipFile string) error {
    zipData, _ := os.ReadFile(zipFile)
    
    // Validate before sending
    if err := bundle.ValidateZip(zipData); err != nil {
        return fmt.Errorf("invalid bundle: %w", err)
    }
    
    // Send to FlowC server
    client := flowc.NewClient("http://localhost:8080")
    deployment, err := client.DeployAPI(zipData)
    return err
}
```

### FlowC Kubernetes Operator

```go
// In operator reconcile loop
func (r *Reconciler) createBundle(crd *flowcv1.APIDeployment) ([]byte, error) {
    // Convert CRD to FlowC metadata
    metadata := &types.FlowCMetadata{
        Name:     crd.Spec.Name,
        Version:  crd.Spec.Version,
        Context:  crd.Spec.Context,
        Gateway:  convertGatewayConfig(crd.Spec.Gateway),
        Upstream: convertUpstreamConfig(crd.Spec.Upstream),
    }
    
    // Get OpenAPI from ConfigMap
    openapiYAML := getOpenAPIFromConfigMap(crd.Spec.OpenAPIRef)
    
    // Create bundle
    b := bundle.NewBundle(metadata, openapiYAML)
    return bundle.CreateZipFromBundle(b)
}
```

### FlowC Server

```go
// In deployment handler
func (h *Handler) DeployAPI(w http.ResponseWriter, r *http.Request) {
    zipData, _ := io.ReadAll(r.Body)
    
    // Extract files
    flowcYAML, openapiYAML, err := bundle.ExtractFiles(zipData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Parse and deploy...
}
```

## Error Handling

The package returns descriptive errors for common issues:

- `"flowc.yaml content is empty"` - FlowC YAML is empty
- `"openapi.yaml content is empty"` - OpenAPI YAML is empty
- `"zip data is empty"` - No ZIP data provided
- `"bundle size exceeds maximum allowed size"` - Bundle too large (>100MB)
- `"invalid ZIP file: missing ZIP signature"` - Not a valid ZIP file
- `"bundle missing required file: flowc.yaml"` - flowc.yaml not found
- `"bundle missing required file: openapi.yaml"` - openapi.yaml not found

## Testing

Run tests:

```bash
go test ./pkg/bundle/...
```

Run tests with coverage:

```bash
go test -cover ./pkg/bundle/...
```

## Best Practices

1. **Always validate bundles** before processing them
2. **Check file sizes** to prevent memory issues
3. **Use descriptive error messages** when bundle creation fails
4. **Support alternative filenames** (`.yml` vs `.yaml`)
5. **Limit maximum bundle size** to prevent abuse

## License

See the LICENSE file in the repository root.



