package types

import (
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// FlowCMetadata represents the metadata from flowc.yaml
type FlowCMetadata struct {
	// Name of the API
	Name string `yaml:"name" json:"name"`

	// Version of the API
	Version string `yaml:"version" json:"version"`

	// Description of the API
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Context of the API (URL path context exposed from the gateway)
	Context string `yaml:"context" json:"context"`

	// Gateway configuration
	Gateway GatewayConfig `yaml:"gateway" json:"gateway"`

	// Upstream configuration
	Upstream UpstreamConfig `yaml:"upstream" json:"upstream"`

	// Labels for the API
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// VirtualHostConfig represents virtual host settings
type VirtualHostConfig struct {
	// Name of the virtual host (auto-generated if not provided)
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Domains this virtual host should match
	Domains []string `yaml:"domains,omitempty" json:"domains,omitempty"`

	// Use existing virtual host by name (for grouping APIs)
	UseExisting string `yaml:"use_existing,omitempty" json:"use_existing,omitempty"`
}

// GatewayConfig represents gateway configuration
type GatewayConfig struct {
	// Node ID of the gateway
	NodeID string `yaml:"node_id" json:"node_id"`

	// Listener of the gateway
	Listener string `yaml:"listener" json:"listener"`

	// Virtual host configuration
	VirtualHost VirtualHostConfig `yaml:"virtual_host,omitempty" json:"virtual_host,omitempty"`

	// HTTP filters to apply to the gateway
	HTTPFilters []HTTPFilter `yaml:"http_filters,omitempty" json:"http_filters,omitempty"`
}

// UpstreamConfig represents upstream service configuration
type UpstreamConfig struct {
	// Host of the upstream service
	Host string `yaml:"host" json:"host"`

	// Port of the upstream service
	Port uint32 `yaml:"port" json:"port"`

	// Scheme of the upstream service
	Scheme string `yaml:"scheme,omitempty" json:"scheme,omitempty"`

	// Timeout of the upstream service
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// HTTPFilter represents an HTTP filter to apply to the gateway
type HTTPFilter struct {
	// Name of the HTTP filter
	Name string `yaml:"name" json:"name"`

	// Configuration of the HTTP filter
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// APIDeployment represents a complete API deployment
type APIDeploymentInfo struct {
	ID        string    `yaml:"id" json:"id"`
	Name      string    `yaml:"name" json:"name"`
	Version   string    `yaml:"version" json:"version"`
	Context   string    `yaml:"context" json:"context"`
	Status    string    `yaml:"status" json:"status"`
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
}

// APIRoute represents a route extracted from OpenAPI paths
type APIRoute struct {
	Path        string              `yaml:"path" json:"path"`
	Method      string              `yaml:"method" json:"method"`
	Operation   *openapi3.Operation `yaml:"operation,omitempty" json:"operation,omitempty"`
	OperationID string              `yaml:"operation_id,omitempty" json:"operation_id,omitempty"`
	Summary     string              `yaml:"summary,omitempty" json:"summary,omitempty"`
	Tags        []string            `yaml:"tags,omitempty" json:"tags,omitempty"`
}
