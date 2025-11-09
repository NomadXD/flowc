package bundle

import (
	"testing"

	"github.com/flowc-labs/flowc/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestCreateZip(t *testing.T) {
	flowcYAML := []byte(`name: test-api
version: v1.0.0
context: test
gateway:
  mediation: {}
upstream:
  host: localhost
  port: 8080
`)

	openapiYAML := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
`)

	zipData, err := CreateZip(flowcYAML, openapiYAML)
	if err != nil {
		t.Fatalf("CreateZip failed: %v", err)
	}

	if len(zipData) == 0 {
		t.Fatal("CreateZip returned empty data")
	}

	// Verify ZIP signature
	if len(zipData) < 4 || string(zipData[:4]) != "PK\x03\x04" {
		t.Fatal("CreateZip did not create valid ZIP file")
	}
}

func TestValidateZip(t *testing.T) {
	tests := []struct {
		name    string
		zipData []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			zipData: []byte{},
			wantErr: true,
		},
		{
			name:    "invalid zip signature",
			zipData: []byte("not a zip file"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateZip(tt.zipData)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateZip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractFiles(t *testing.T) {
	// Create a valid ZIP
	flowcYAML := []byte(`name: test-api
version: v1.0.0
context: test
gateway:
  mediation: {}
upstream:
  host: localhost
  port: 8080
`)

	openapiYAML := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
`)

	zipData, err := CreateZip(flowcYAML, openapiYAML)
	if err != nil {
		t.Fatalf("CreateZip failed: %v", err)
	}

	// Extract files
	extractedFlowC, extractedOpenAPI, err := ExtractFiles(zipData)
	if err != nil {
		t.Fatalf("ExtractFiles failed: %v", err)
	}

	if string(extractedFlowC) != string(flowcYAML) {
		t.Errorf("Extracted flowc.yaml does not match original")
	}

	if string(extractedOpenAPI) != string(openapiYAML) {
		t.Errorf("Extracted openapi.yaml does not match original")
	}
}

func TestCreateZipFromBundle(t *testing.T) {
	metadata := &types.FlowCMetadata{
		Name:    "test-api",
		Version: "v1.0.0",
		Context: "test",
		Gateway: types.GatewayConfig{},
		Upstream: types.UpstreamConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	openapiData := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
`)

	bundle := NewBundle(metadata, openapiData)
	zipData, err := CreateZipFromBundle(bundle)
	if err != nil {
		t.Fatalf("CreateZipFromBundle failed: %v", err)
	}

	// Validate the created ZIP
	if err := ValidateZip(zipData); err != nil {
		t.Fatalf("Created ZIP is invalid: %v", err)
	}

	// Extract and verify
	flowcYAML, extractedOpenAPI, err := ExtractFiles(zipData)
	if err != nil {
		t.Fatalf("ExtractFiles failed: %v", err)
	}

	// Unmarshal and compare metadata
	var extractedMetadata types.FlowCMetadata
	if err := yaml.Unmarshal(flowcYAML, &extractedMetadata); err != nil {
		t.Fatalf("Failed to unmarshal extracted flowc.yaml: %v", err)
	}

	if extractedMetadata.Name != metadata.Name {
		t.Errorf("Name mismatch: got %s, want %s", extractedMetadata.Name, metadata.Name)
	}

	if string(extractedOpenAPI) != string(openapiData) {
		t.Errorf("OpenAPI data mismatch")
	}
}

func TestListFiles(t *testing.T) {
	flowcYAML := []byte(`name: test-api
version: v1.0.0
context: test
`)

	openapiYAML := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
`)

	zipData, err := CreateZip(flowcYAML, openapiYAML)
	if err != nil {
		t.Fatalf("CreateZip failed: %v", err)
	}

	files, err := ListFiles(zipData)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	expectedFiles := map[string]bool{
		FlowCFileName:   false,
		OpenAPIFileName: false,
	}

	for _, file := range files {
		if _, exists := expectedFiles[file]; exists {
			expectedFiles[file] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found in bundle", file)
		}
	}
}
