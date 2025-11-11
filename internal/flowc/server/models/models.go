package models

import (
	"time"

	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
)

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
	NodeID      string            `yaml:"node_id" json:"node_id"`
	Listener    string            `yaml:"listener" json:"listener"`
	VirtualHost VirtualHostConfig `yaml:"virtual_host,omitempty" json:"virtual_host,omitempty"`
	HTTPFilters []HTTPFilter      `yaml:"http_filters,omitempty" json:"http_filters,omitempty"`
}

// UpstreamConfig represents upstream service configuration
type UpstreamConfig struct {
	Host    string `yaml:"host" json:"host"`
	Port    uint32 `yaml:"port" json:"port"`
	Scheme  string `yaml:"scheme,omitempty" json:"scheme,omitempty"`
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

type HTTPFilter struct {
	Name   string                 `yaml:"name" json:"name"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// OpenAPISpec is now an alias to the kin-openapi T type for better compatibility
type OpenAPISpec = openapi3.T

// APIDeployment represents a complete API deployment
type APIDeployment struct {
	ID          string              `yaml:"id" json:"id"`
	Name        string              `json:"name"`
	Version     string              `yaml:"version" json:"version"`
	Context     string              `yaml:"context" json:"context"`
	Status      string              `yaml:"status" json:"status"`
	CreatedAt   time.Time           `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time           `yaml:"updated_at" json:"updated_at"`
	Metadata    types.FlowCMetadata `yaml:"metadata" json:"metadata"`
	OpenAPISpec OpenAPISpec         `yaml:"openapi_spec" json:"openapi_spec"`
}

// DeploymentStatus represents the status of an API deployment
type DeploymentStatus string

const (
	StatusPending   DeploymentStatus = "pending"
	StatusDeploying DeploymentStatus = "deploying"
	StatusDeployed  DeploymentStatus = "deployed"
	StatusFailed    DeploymentStatus = "failed"
	StatusUpdating  DeploymentStatus = "updating"
	StatusDeleting  DeploymentStatus = "deleting"
	StatusDeleted   DeploymentStatus = "deleted"
)

// APIRoute represents a route extracted from OpenAPI paths
type APIRoute struct {
	Path        string              `yaml:"path" json:"path"`
	Method      string              `yaml:"method" json:"method"`
	Operation   *openapi3.Operation `yaml:"operation,omitempty" json:"operation,omitempty"`
	OperationID string              `yaml:"operation_id,omitempty" json:"operation_id,omitempty"`
	Summary     string              `yaml:"summary,omitempty" json:"summary,omitempty"`
	Tags        []string            `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// DeploymentRequest represents the request payload for API deployment
type DeploymentRequest struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// DeploymentResponse represents the response for API deployment
type DeploymentResponse struct {
	Success    bool           `yaml:"success" json:"success"`
	Message    string         `yaml:"message" json:"message"`
	Deployment *APIDeployment `yaml:"deployment,omitempty" json:"deployment,omitempty"`
	Error      string         `yaml:"error,omitempty" json:"error,omitempty"`
}

// ListDeploymentsResponse represents the response for listing deployments
type ListDeploymentsResponse struct {
	Success     bool             `yaml:"success" json:"success"`
	Deployments []*APIDeployment `yaml:"deployments" json:"deployments"`
	Total       int              `yaml:"total" json:"total"`
}

// GetDeploymentResponse represents the response for getting a specific deployment
type GetDeploymentResponse struct {
	Success    bool           `yaml:"success" json:"success"`
	Deployment *APIDeployment `yaml:"deployment,omitempty" json:"deployment,omitempty"`
	Error      string         `yaml:"error,omitempty" json:"error,omitempty"`
}

// DeleteDeploymentResponse represents the response for deleting a deployment
type DeleteDeploymentResponse struct {
	Success bool   `yaml:"success" json:"success"`
	Message string `yaml:"message" json:"message"`
	Error   string `yaml:"error,omitempty" json:"error,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `yaml:"status" json:"status"`
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`
	Version   string    `yaml:"version" json:"version"`
	Uptime    string    `yaml:"uptime" json:"uptime"`
}
