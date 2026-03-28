package resource

import "github.com/flowc-labs/flowc/pkg/types"

// DeploymentResource represents an API deployment to a specific environment.
type DeploymentResource struct {
	Meta   ResourceMeta     `json:"metadata" yaml:"metadata"`
	Spec   DeploymentSpec   `json:"spec" yaml:"spec"`
	Status DeploymentStatus `json:"status" yaml:"status"`
}

// DeploymentSpec defines the desired state of a deployment.
type DeploymentSpec struct {
	// APIRef is the name of the API resource to deploy.
	APIRef string `json:"apiRef" yaml:"apiRef"`

	// GatewayRef is the name of the target Gateway.
	GatewayRef string `json:"gatewayRef" yaml:"gatewayRef"`

	// ListenerRef is the name of the target Listener.
	ListenerRef string `json:"listenerRef" yaml:"listenerRef"`

	// EnvironmentRef is the name of the target Environment.
	EnvironmentRef string `json:"environmentRef" yaml:"environmentRef"`

	// Strategy overrides API/gateway defaults for this deployment.
	Strategy *types.StrategyConfig `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// DeploymentStatus is the observed state of a deployment.
type DeploymentStatus struct {
	Phase              string      `json:"phase" yaml:"phase"` // Pending, Deploying, Deployed, Failed
	Conditions         []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	XDSSnapshotVersion string     `json:"xdsSnapshotVersion,omitempty" yaml:"xdsSnapshotVersion,omitempty"`
}

func (r *DeploymentResource) GetMeta() *ResourceMeta { return &r.Meta }
