package openapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// OpenAPIManager handles OpenAPI specification parsing, validation, and routing
type OpenAPIManager struct {
	loader *openapi3.Loader
}

// NewOpenAPIManager creates a new OpenAPI manager
func NewOpenAPIManager() *OpenAPIManager {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	return &OpenAPIManager{
		loader: loader,
	}
}

// LoadFromData loads an OpenAPI specification from byte data
func (m *OpenAPIManager) LoadFromData(ctx context.Context, data []byte) (*openapi3.T, error) {
	doc, err := m.loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Validate the loaded specification
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	return doc, nil
}

// LoadFromFile loads an OpenAPI specification from a file
func (m *OpenAPIManager) LoadFromFile(ctx context.Context, filename string) (*openapi3.T, error) {
	doc, err := m.loader.LoadFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec from file: %w", err)
	}

	// Validate the loaded specification
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	return doc, nil
}

// LoadFromURI loads an OpenAPI specification from a URI
func (m *OpenAPIManager) LoadFromURI(ctx context.Context, uri *url.URL) (*openapi3.T, error) {
	doc, err := m.loader.LoadFromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec from URI: %w", err)
	}

	// Validate the loaded specification
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	return doc, nil
}

// ExtractRoutes extracts API routes from an OpenAPI specification
func (m *OpenAPIManager) ExtractRoutes(doc *openapi3.T) ([]*openapi3.PathItem, error) {
	var routes []*openapi3.PathItem

	if doc.Paths == nil {
		return routes, nil
	}

	for _, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}

		// Handle different HTTP methods
		operations := map[string]*openapi3.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
			"TRACE":   pathItem.Trace,
		}

		for _, operation := range operations {
			if operation == nil {
				continue
			}

			routes = append(routes, pathItem)
		}
	}

	return routes, nil
}

// CreateRouter creates a router from an OpenAPI specification for request validation
func (m *OpenAPIManager) CreateRouter(doc *openapi3.T) (routers.Router, error) {
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	return router, nil
}

// ValidateRequest validates an HTTP request against the OpenAPI specification
func (m *OpenAPIManager) ValidateRequest(ctx context.Context, req *http.Request, router routers.Router) error {
	// Find the route for this request
	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("route not found: %w", err)
	}

	// Create validation input
	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	// Validate the request
	if err := openapi3filter.ValidateRequest(ctx, requestValidationInput); err != nil {
		return fmt.Errorf("request validation failed: %w", err)
	}

	return nil
}

// ValidateResponse validates an HTTP response against the OpenAPI specification
func (m *OpenAPIManager) ValidateResponse(ctx context.Context, req *http.Request, resp *http.Response, router routers.Router) error {
	// Find the route for this request
	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("route not found: %w", err)
	}

	// Create validation input
	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
	}

	// Validate the response
	if err := openapi3filter.ValidateResponse(ctx, responseValidationInput); err != nil {
		return fmt.Errorf("response validation failed: %w", err)
	}

	return nil
}

// GetOperationByID finds an operation by its operationId
func (m *OpenAPIManager) GetOperationByID(doc *openapi3.T, operationID string) (*openapi3.Operation, string, string, error) {
	if doc.Paths == nil {
		return nil, "", "", fmt.Errorf("no paths defined in OpenAPI spec")
	}

	for path, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}

		operations := map[string]*openapi3.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
			"TRACE":   pathItem.Trace,
		}

		for method, operation := range operations {
			if operation != nil && operation.OperationID == operationID {
				return operation, path, method, nil
			}
		}
	}

	return nil, "", "", fmt.Errorf("operation with ID '%s' not found", operationID)
}

// GetSchemaByRef gets a schema by its reference
func (m *OpenAPIManager) GetSchemaByRef(doc *openapi3.T, ref string) (*openapi3.Schema, error) {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return nil, fmt.Errorf("no components/schemas defined in OpenAPI spec")
	}

	// Remove the "#/components/schemas/" prefix if present
	schemaName := ref
	if len(ref) > 21 && ref[:21] == "#/components/schemas/" {
		schemaName = ref[21:]
	}

	schemaRef, exists := doc.Components.Schemas[schemaName]
	if !exists {
		return nil, fmt.Errorf("schema '%s' not found", schemaName)
	}

	if schemaRef.Value == nil {
		return nil, fmt.Errorf("schema '%s' has no value", schemaName)
	}

	return schemaRef.Value, nil
}

// ValidateData validates data against a schema
func (m *OpenAPIManager) ValidateData(ctx context.Context, schema *openapi3.Schema, data interface{}) error {
	if err := schema.VisitJSON(data); err != nil {
		return fmt.Errorf("data validation failed: %w", err)
	}
	return nil
}
