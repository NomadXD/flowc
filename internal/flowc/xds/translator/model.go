package translator

import (
	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
)

// DeploymentModel represents the internal FlowC deployment representation
// This is the canonical model that all xDS translators work with
type DeploymentModel struct {
	// Metadata contains all FlowC configuration
	Metadata *types.FlowCMetadata

	// OpenAPISpec contains the API specification
	OpenAPISpec *openapi3.T

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
