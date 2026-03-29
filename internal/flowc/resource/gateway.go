package resource

import "github.com/flowc-labs/flowc/pkg/types"

// GatewayResource represents a registered gateway (Envoy proxy instance).
type GatewayResource struct {
	Meta   ResourceMeta  `json:"metadata" yaml:"metadata"`
	Spec   GatewaySpec   `json:"spec" yaml:"spec"`
	Status GatewayStatus `json:"status" yaml:"status"`
}

// GatewaySpec defines the desired state of a gateway.
type GatewaySpec struct {
	// NodeID is the Envoy node ID for xDS; must be unique across gateways.
	NodeID string `json:"nodeId" yaml:"nodeId"`

	// ProfileRef optionally references a GatewayProfile resource by name.
	// When set, the profile's defaults are applied between builtin defaults
	// and this gateway's own defaults in the strategy precedence chain.
	ProfileRef string `json:"profileRef,omitempty" yaml:"profileRef,omitempty"`

	// Defaults are optional strategy defaults for APIs deployed to this gateway.
	Defaults *types.StrategyConfig `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

// GatewayStatus is the observed state of a gateway.
type GatewayStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

func (r *GatewayResource) GetMeta() *ResourceMeta { return &r.Meta }
