package resource

import "github.com/flowc-labs/flowc/pkg/types"

// DeploymentResource represents an API deployment to a gateway.
type DeploymentResource struct {
	Meta   ResourceMeta     `json:"metadata" yaml:"metadata"`
	Spec   DeploymentSpec   `json:"spec" yaml:"spec"`
	Status DeploymentStatus `json:"status" yaml:"status"`
}

// DeploymentSpec defines the desired state of a deployment.
type DeploymentSpec struct {
	// APIRef is the name of the API resource to deploy.
	APIRef string `json:"apiRef" yaml:"apiRef"`

	// Gateway specifies the target gateway and optionally the listener and virtual host.
	Gateway DeploymentGatewayRef `json:"gateway" yaml:"gateway"`

	// Strategy overrides API/gateway defaults for this deployment.
	Strategy *types.StrategyConfig `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// DeploymentGatewayRef identifies the target gateway, listener, and virtual host for a deployment.
type DeploymentGatewayRef struct {
	// Name is the name of the target Gateway (required).
	Name string `json:"name" yaml:"name"`

	// Listener is the name of the target Listener (optional; auto-resolved if gateway has exactly one).
	Listener string `json:"listener,omitempty" yaml:"listener,omitempty"`

	// VirtualHost is the name of the target VirtualHost (optional; auto-resolved if listener has exactly one).
	VirtualHost string `json:"virtualHost,omitempty" yaml:"virtualHost,omitempty"`
}

// DeploymentStatus is the observed state of a deployment.
type DeploymentStatus struct {
	Phase              string      `json:"phase" yaml:"phase"` // Pending, Deploying, Deployed, Failed
	Conditions         []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	XDSSnapshotVersion string     `json:"xdsSnapshotVersion,omitempty" yaml:"xdsSnapshotVersion,omitempty"`
}

func (r *DeploymentResource) GetMeta() *ResourceMeta { return &r.Meta }
