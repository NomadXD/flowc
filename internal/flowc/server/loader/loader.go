package loader

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/pkg/openapi"
	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// ZipParser handles parsing of zip files containing API specifications
type BundleLoader struct {
	openAPIManager *openapi.OpenAPIManager
}

// NewZipParser creates a new zip parser instance
func NewBundleLoader() *BundleLoader {
	return &BundleLoader{
		openAPIManager: openapi.NewOpenAPIManager(),
	}
}

// ParseResult contains the parsed results from a zip file
type DeploymentBundle struct {
	FlowCMetadata *types.FlowCMetadata
	OpenAPISpec   *openapi3.T
	Routes        []*openapi3.PathItem
}

// LoadBundle loads a bundle from a zip file
func (l *BundleLoader) LoadBundle(zipData []byte) (*DeploymentBundle, error) {
	// Create a reader from the zip data
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}

	var flowcData []byte
	var openapiData []byte

	// Extract files from zip
	for _, file := range reader.File {
		fileName := filepath.Base(file.Name)

		switch fileName {
		case "flowc.yaml", "flowc.yml":
			flowcData, err = l.extractFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to extract flowc.yaml: %w", err)
			}
		case "openapi.yaml", "openapi.yml", "swagger.yaml", "swagger.yml":
			openapiData, err = l.extractFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to extract openapi.yaml: %w", err)
			}
		}
	}

	// Validate required files
	if flowcData == nil {
		return nil, fmt.Errorf("flowc.yaml not found in zip file")
	}
	if openapiData == nil {
		return nil, fmt.Errorf("openapi.yaml not found in zip file")
	}

	// Load FlowC metadata
	flowcMetadata, err := l.loadFlowCMetadata(flowcData)
	if err != nil {
		return nil, fmt.Errorf("failed to load flowc.yaml: %w", err)
	}

	// Load OpenAPI specification
	openAPISpec, err := l.loadOpenAPISpec(openapiData)
	if err != nil {
		return nil, fmt.Errorf("failed to load openapi.yaml: %w", err)
	}

	// Extract routes from OpenAPI spec
	routes, err := l.extractRoutes(openAPISpec)
	if err != nil {
		return nil, fmt.Errorf("failed to extract routes from OpenAPI spec: %w", err)
	}

	return &DeploymentBundle{
		FlowCMetadata: flowcMetadata,
		OpenAPISpec:   openAPISpec,
		Routes:        routes,
	}, nil
}

// extractFile extracts a single file from the zip archive
func (l *BundleLoader) extractFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// loadFlowCMetadata loads the FlowC metadata from YAML
func (l *BundleLoader) loadFlowCMetadata(data []byte) (*types.FlowCMetadata, error) {
	var metadata types.FlowCMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flowc.yaml: %w", err)
	}

	// Validate required fields
	if metadata.Name == "" {
		return nil, fmt.Errorf("name is required in flowc.yaml")
	}
	if metadata.Version == "" {
		return nil, fmt.Errorf("version is required in flowc.yaml")
	}
	if metadata.Context == "" {
		return nil, fmt.Errorf("context is required in flowc.yaml")
	}
	// if metadata.Gateway.Host == "" {
	// 	return nil, fmt.Errorf("gateway.host is required in flowc.yaml")
	// }
	// if metadata.Gateway.Port == 0 {
	// 	return nil, fmt.Errorf("gateway.port is required in flowc.yaml")
	// }
	if metadata.Upstream.Host == "" {
		return nil, fmt.Errorf("upstream.host is required in flowc.yaml")
	}
	if metadata.Upstream.Port == 0 {
		return nil, fmt.Errorf("upstream.port is required in flowc.yaml")
	}

	// Set defaults
	if metadata.Upstream.Scheme == "" {
		metadata.Upstream.Scheme = "http"
	}
	if metadata.Upstream.Timeout == "" {
		metadata.Upstream.Timeout = "30s"
	}

	return &metadata, nil
}

// loadOpenAPISpec loads the OpenAPI specification from YAML using kin-openapi
func (l *BundleLoader) loadOpenAPISpec(data []byte) (*models.OpenAPISpec, error) {
	ctx := context.Background()

	// Use kin-openapi to load and validate the specification
	spec, err := l.openAPIManager.LoadFromData(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse and validate openapi.yaml: %w", err)
	}

	return spec, nil
}

// extractRoutes extracts route information from OpenAPI paths using kin-openapi
func (l *BundleLoader) extractRoutes(spec *openapi3.T) ([]*openapi3.PathItem, error) {
	// Use the OpenAPI manager to extract routes
	routes, err := l.openAPIManager.ExtractRoutes(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to extract routes: %w", err)
	}

	return routes, nil
}
