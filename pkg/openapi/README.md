# OpenAPI Integration with kin-openapi

This package provides OpenAPI specification parsing, validation, and data model integration using the [kin-openapi](https://github.com/getkin/kin-openapi) library.

## Features

- **OpenAPI Specification Loading**: Load and validate OpenAPI 3.0 specifications from files, data, or URIs
- **Request/Response Validation**: Validate HTTP requests and responses against OpenAPI specifications
- **Route Extraction**: Extract API routes from OpenAPI path definitions
- **Schema Validation**: Validate data against OpenAPI schema definitions
- **Router Creation**: Create routers for request routing and validation

## Usage

### Basic OpenAPI Manager

```go
import "github.com/flowc-labs/flowc/internal/api/openapi"

// Create a new OpenAPI manager
manager := openapi.NewOpenAPIManager()

// Load OpenAPI specification from file
ctx := context.Background()
doc, err := manager.LoadFromFile(ctx, "openapi.yaml")
if err != nil {
    log.Fatal(err)
}

// Extract routes from the specification
routes, err := manager.ExtractRoutes(doc)
if err != nil {
    log.Fatal(err)
}
```

### Request Validation

```go
// Create router for validation
router, err := manager.CreateRouter(doc)
if err != nil {
    log.Fatal(err)
}

// Validate HTTP request
err = manager.ValidateRequest(ctx, httpRequest, router)
if err != nil {
    // Handle validation error
}
```

### Integration with Deployment Service

The deployment service automatically creates OpenAPI routers for each deployed API:

```go
// Get OpenAPI router for a deployment
router, exists := deploymentService.GetOpenAPIRouter(deploymentID)
if exists {
    // Use router for validation
}

// Validate request for specific deployment
err := deploymentService.ValidateRequestForDeployment(ctx, deploymentID, httpRequest)
```

### Middleware Integration

Use the OpenAPI validation middleware to automatically validate requests:

```go
import "github.com/flowc-labs/flowc/internal/api/middleware"

// Create middleware
middleware := middleware.NewOpenAPIValidationMiddleware(logger)

// Set router for validation
middleware.SetRouter(router)

// Use middleware in HTTP handler chain
http.Handle("/api/", middleware.ValidateRequest(yourHandler))
```

## Data Models

The integration updates the existing models to use kin-openapi structures:

- `OpenAPISpec` is now an alias to `openapi3.T`
- `APIRoute` includes an `Operation` field with `*openapi3.Operation`
- Full compatibility with kin-openapi validation and parsing

## Error Handling

The library provides detailed error messages for:
- Invalid OpenAPI specifications
- Request validation failures
- Response validation failures
- Schema validation errors

## Dependencies

- `github.com/getkin/kin-openapi/openapi3` - Core OpenAPI 3.0 support
- `github.com/getkin/kin-openapi/openapi3filter` - Request/response validation
- `github.com/getkin/kin-openapi/routers` - Router interfaces
- `github.com/getkin/kin-openapi/routers/gorillamux` - Gorilla Mux router implementation
