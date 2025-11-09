package bundle

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/flowc-labs/flowc/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	// Standard filenames in FlowC bundles
	FlowCFileName   = "flowc.yaml"
	OpenAPIFileName = "openapi.yaml"

	// MaxBundleSize is the maximum size of a bundle (100MB)
	MaxBundleSize = 100 * 1024 * 1024
)

// Bundle represents a FlowC API bundle containing flowc.yaml and openapi.yaml
type Bundle struct {
	FlowCMetadata *types.FlowCMetadata
	OpenAPIData   []byte // Raw OpenAPI YAML data
}

// NewBundle creates a new bundle from metadata and OpenAPI data
func NewBundle(metadata *types.FlowCMetadata, openapiData []byte) *Bundle {
	return &Bundle{
		FlowCMetadata: metadata,
		OpenAPIData:   openapiData,
	}
}

// CreateZip creates a ZIP file containing flowc.yaml and openapi.yaml
func CreateZip(flowcYAML, openapiYAML []byte) ([]byte, error) {
	if len(flowcYAML) == 0 {
		return nil, fmt.Errorf("flowc.yaml content is empty")
	}
	if len(openapiYAML) == 0 {
		return nil, fmt.Errorf("openapi.yaml content is empty")
	}

	// Create a buffer to write the ZIP to
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add flowc.yaml
	if err := addFileToZip(zipWriter, FlowCFileName, flowcYAML); err != nil {
		zipWriter.Close()
		return nil, fmt.Errorf("failed to add flowc.yaml: %w", err)
	}

	// Add openapi.yaml
	if err := addFileToZip(zipWriter, OpenAPIFileName, openapiYAML); err != nil {
		zipWriter.Close()
		return nil, fmt.Errorf("failed to add openapi.yaml: %w", err)
	}

	// Close the ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// CreateZipFromBundle creates a ZIP file from a Bundle
func CreateZipFromBundle(bundle *Bundle) ([]byte, error) {
	// Marshal FlowC metadata to YAML
	flowcYAML, err := yaml.Marshal(bundle.FlowCMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal flowc metadata: %w", err)
	}

	return CreateZip(flowcYAML, bundle.OpenAPIData)
}

// ValidateZip checks if a ZIP file contains the required files
func ValidateZip(zipData []byte) error {
	if len(zipData) == 0 {
		return fmt.Errorf("zip data is empty")
	}

	if len(zipData) > MaxBundleSize {
		return fmt.Errorf("bundle size exceeds maximum allowed size of %d bytes", MaxBundleSize)
	}

	// Check ZIP signature
	if len(zipData) < 4 || !bytes.HasPrefix(zipData, []byte("PK\x03\x04")) {
		return fmt.Errorf("invalid ZIP file: missing ZIP signature")
	}

	// Create a reader from the ZIP data
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to read zip file: %w", err)
	}

	// Check for required files
	hasFlowC := false
	hasOpenAPI := false

	for _, file := range reader.File {
		fileName := filepath.Base(file.Name)

		switch fileName {
		case FlowCFileName, "flowc.yml":
			hasFlowC = true
		case OpenAPIFileName, "openapi.yml", "swagger.yaml", "swagger.yml":
			hasOpenAPI = true
		}
	}

	if !hasFlowC {
		return fmt.Errorf("bundle missing required file: %s", FlowCFileName)
	}
	if !hasOpenAPI {
		return fmt.Errorf("bundle missing required file: %s (or swagger.yaml)", OpenAPIFileName)
	}

	return nil
}

// ExtractFiles extracts flowc.yaml and openapi.yaml from a ZIP file
func ExtractFiles(zipData []byte) (flowcYAML, openapiYAML []byte, err error) {
	// Validate first
	if err := ValidateZip(zipData); err != nil {
		return nil, nil, err
	}

	// Create a reader from the ZIP data
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read zip file: %w", err)
	}

	// Extract files
	for _, file := range reader.File {
		fileName := filepath.Base(file.Name)

		switch fileName {
		case FlowCFileName, "flowc.yml":
			flowcYAML, err = extractFile(file)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to extract %s: %w", fileName, err)
			}
		case OpenAPIFileName, "openapi.yml", "swagger.yaml", "swagger.yml":
			openapiYAML, err = extractFile(file)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to extract %s: %w", fileName, err)
			}
		}
	}

	if flowcYAML == nil {
		return nil, nil, fmt.Errorf("flowc.yaml not found in bundle")
	}
	if openapiYAML == nil {
		return nil, nil, fmt.Errorf("openapi.yaml not found in bundle")
	}

	return flowcYAML, openapiYAML, nil
}

// ListFiles returns a list of all files in the ZIP bundle
func ListFiles(zipData []byte) ([]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}

	files := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		files = append(files, file.Name)
	}

	return files, nil
}

// addFileToZip adds a file to a ZIP archive
func addFileToZip(zipWriter *zip.Writer, filename string, data []byte) error {
	fileWriter, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file in zip: %w", err)
	}

	_, err = fileWriter.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	return nil
}

// extractFile extracts a single file from the ZIP archive
func extractFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}
