package translator

import (
	"github.com/flowc-labs/flowc/internal/flowc/ir"
	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
)

// DeploymentModel represents the internal FlowC deployment representation
// This is the canonical model that all xDS translators work with
type DeploymentModel struct {
	// Metadata contains all FlowC configuration
	Metadata *types.FlowCMetadata

	// OpenAPISpec contains the API specification (DEPRECATED: use IR instead)
	// Kept for backward compatibility with existing translators
	OpenAPISpec *openapi3.T

	// IR contains the unified intermediate representation of the API
	// This should be used by translators instead of OpenAPISpec
	IR *ir.API

	// DeploymentID is a unique identifier for this deployment
	DeploymentID string

	// Additional context for advanced strategies
	Context *DeploymentContext

	// Strategy configuration for xDS generation
	StrategyConfig *types.StrategyConfig
}

// DeploymentContext provides additional context for xDS generation
type DeploymentContext struct {
	// NodeID is the target Envoy node ID
	NodeID string

	// Namespace for resource isolation (optional)
	Namespace string

	// Labels for resource tagging and organization
	Labels map[string]string

	// TrafficStrategy defines how traffic should be routed (canary, blue-green, etc.)
	TrafficStrategy *TrafficStrategy

	// Advanced configurations
	CustomConfig map[string]interface{}
}

// TrafficStrategy defines traffic routing strategies
type TrafficStrategy struct {
	// Type of strategy: "basic", "canary", "blue-green", "a-b-test"
	Type string

	// Weight-based routing (for canary)
	// Key: version/deployment, Value: weight percentage (0-100)
	Weights map[string]int

	// Blue-Green settings
	BlueGreen *BlueGreenConfig

	// Canary settings
	Canary *CanaryConfig
}

// BlueGreenConfig defines blue-green deployment configuration
type BlueGreenConfig struct {
	// ActiveVersion is the currently serving version
	ActiveVersion string

	// StandbyVersion is the version ready to switch to
	StandbyVersion string

	// AutoPromote indicates if automatic promotion is enabled
	AutoPromote bool
}

// CanaryConfig defines canary deployment configuration
type CanaryConfig struct {
	// BaselineVersion is the stable version
	BaselineVersion string

	// CanaryVersion is the new version being tested
	CanaryVersion string

	// CanaryWeight is the percentage of traffic to canary (0-100)
	CanaryWeight int

	// MatchCriteria for header-based routing
	MatchCriteria *MatchCriteria
}

// MatchCriteria defines advanced traffic matching
type MatchCriteria struct {
	// Headers to match for routing
	Headers map[string]string

	// QueryParams to match
	QueryParams map[string]string

	// SourceLabels to match (for service mesh)
	SourceLabels map[string]string
}

// NewDeploymentModel creates a new deployment model from metadata and spec
// DEPRECATED: Use NewDeploymentModelWithIR instead for new implementations
func NewDeploymentModel(metadata *types.FlowCMetadata, spec *openapi3.T, deploymentID string) *DeploymentModel {
	return &DeploymentModel{
		Metadata:     metadata,
		OpenAPISpec:  spec,
		DeploymentID: deploymentID,
		Context: &DeploymentContext{
			Labels: make(map[string]string),
		},
	}
}

// NewDeploymentModelWithIR creates a new deployment model with IR representation
// This is the preferred method for creating deployment models
func NewDeploymentModelWithIR(metadata *types.FlowCMetadata, irAPI *ir.API, deploymentID string) *DeploymentModel {
	return &DeploymentModel{
		Metadata:     metadata,
		IR:           irAPI,
		DeploymentID: deploymentID,
		Context: &DeploymentContext{
			Labels: make(map[string]string),
		},
	}
}

// NewDeploymentModelComplete creates a deployment model with both OpenAPI and IR
// Used during transition period for backward compatibility
func NewDeploymentModelComplete(metadata *types.FlowCMetadata, spec *openapi3.T, irAPI *ir.API, deploymentID string) *DeploymentModel {
	return &DeploymentModel{
		Metadata:     metadata,
		OpenAPISpec:  spec,
		IR:           irAPI,
		DeploymentID: deploymentID,
		Context: &DeploymentContext{
			Labels: make(map[string]string),
		},
	}
}

// WithNodeID sets the target node ID
func (m *DeploymentModel) WithNodeID(nodeID string) *DeploymentModel {
	m.Context.NodeID = nodeID
	return m
}

// WithNamespace sets the namespace
func (m *DeploymentModel) WithNamespace(namespace string) *DeploymentModel {
	m.Context.Namespace = namespace
	return m
}

// WithTrafficStrategy sets the traffic strategy
func (m *DeploymentModel) WithTrafficStrategy(strategy *TrafficStrategy) *DeploymentModel {
	m.Context.TrafficStrategy = strategy
	return m
}

// WithLabels sets labels
func (m *DeploymentModel) WithLabels(labels map[string]string) *DeploymentModel {
	m.Context.Labels = labels
	return m
}

// WithCustomConfig sets custom configuration
func (m *DeploymentModel) WithCustomConfig(config map[string]interface{}) *DeploymentModel {
	m.Context.CustomConfig = config
	return m
}

// WithStrategyConfig sets the strategy configuration
func (m *DeploymentModel) WithStrategyConfig(config *types.StrategyConfig) *DeploymentModel {
	m.StrategyConfig = config
	return m
}

// WithIR sets the IR representation
func (m *DeploymentModel) WithIR(irAPI *ir.API) *DeploymentModel {
	m.IR = irAPI
	return m
}

// GetAPIType returns the API type from the IR or defaults to REST
func (m *DeploymentModel) GetAPIType() ir.APIType {
	if m.IR != nil {
		return m.IR.Metadata.Type
	}
	// Fallback to metadata or default
	if m.Metadata != nil && m.Metadata.APIType != "" {
		return ir.APIType(m.Metadata.APIType)
	}
	return ir.APITypeREST // Default for backward compatibility
}

// IsRESTAPI checks if this is a REST/HTTP API
func (m *DeploymentModel) IsRESTAPI() bool {
	return m.GetAPIType() == ir.APITypeREST
}

// IsGRPCAPI checks if this is a gRPC API
func (m *DeploymentModel) IsGRPCAPI() bool {
	return m.GetAPIType() == ir.APITypeGRPC
}

// IsGraphQLAPI checks if this is a GraphQL API
func (m *DeploymentModel) IsGraphQLAPI() bool {
	return m.GetAPIType() == ir.APITypeGraphQL
}

// IsWebSocketAPI checks if this is a WebSocket API
func (m *DeploymentModel) IsWebSocketAPI() bool {
	return m.GetAPIType() == ir.APITypeWebSocket
}

// IsSSEAPI checks if this is a Server-Sent Events API
func (m *DeploymentModel) IsSSEAPI() bool {
	return m.GetAPIType() == ir.APITypeSSE
}

// GetBasePath returns the gateway base path for this API
// This is a unified concept that works across all API types:
// - REST: Base path prefix for all HTTP routes
// - gRPC: Base path for gRPC services
// - GraphQL: Base path for GraphQL endpoint
// - WebSocket: Base path for WebSocket connections
// - SSE: Base path for Server-Sent Events
// Falls back to metadata.Context if IR is not available (for backward compatibility)
func (m *DeploymentModel) GetBasePath() string {
	if m.IR != nil && m.IR.Metadata.BasePath != "" {
		return m.IR.Metadata.BasePath
	}
	// Fallback to metadata.Context for backward compatibility
	if m.Metadata != nil && m.Metadata.Context != "" {
		// Normalize the path
		path := m.Metadata.Context
		if path[0] != '/' {
			path = "/" + path
		}
		if len(path) > 1 && path[len(path)-1] == '/' {
			path = path[:len(path)-1]
		}
		return path
	}
	return "/" // Default to root
}
