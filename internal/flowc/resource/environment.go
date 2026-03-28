package resource

import "github.com/flowc-labs/flowc/pkg/types"

// EnvironmentResource represents a virtual environment within a listener.
// Environments use hostname-based SNI for filter chain matching.
type EnvironmentResource struct {
	Meta   ResourceMeta      `json:"metadata" yaml:"metadata"`
	Spec   EnvironmentSpec   `json:"spec" yaml:"spec"`
	Status EnvironmentStatus `json:"status" yaml:"status"`
}

// EnvironmentSpec defines the desired state of an environment.
type EnvironmentSpec struct {
	// GatewayRef is the name of the parent Gateway resource.
	GatewayRef string `json:"gatewayRef" yaml:"gatewayRef"`

	// ListenerRef is the name of the parent Listener resource.
	ListenerRef string `json:"listenerRef" yaml:"listenerRef"`

	// Hostname is the SNI hostname; must be unique within the listener.
	Hostname string `json:"hostname" yaml:"hostname"`

	// HTTPFilters are optional per-environment HTTP filters.
	HTTPFilters []types.HTTPFilter `json:"httpFilters,omitempty" yaml:"httpFilters,omitempty"`
}

// EnvironmentStatus is the observed state of an environment.
type EnvironmentStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

func (r *EnvironmentResource) GetMeta() *ResourceMeta { return &r.Meta }
