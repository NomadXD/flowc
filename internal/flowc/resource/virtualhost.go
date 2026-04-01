package resource

import "github.com/flowc-labs/flowc/pkg/types"

// VirtualHostResource represents a virtual host within a listener.
// Virtual hosts use hostname-based SNI for filter chain matching.
type VirtualHostResource struct {
	Meta   ResourceMeta      `json:"metadata" yaml:"metadata"`
	Spec   VirtualHostSpec   `json:"spec" yaml:"spec"`
	Status VirtualHostStatus `json:"status" yaml:"status"`
}

// VirtualHostSpec defines the desired state of a virtual host.
type VirtualHostSpec struct {
	// GatewayRef is the name of the parent Gateway resource.
	GatewayRef string `json:"gatewayRef" yaml:"gatewayRef"`

	// ListenerRef is the name of the parent Listener resource.
	ListenerRef string `json:"listenerRef" yaml:"listenerRef"`

	// Hostname is the SNI hostname; must be unique within the listener.
	Hostname string `json:"hostname" yaml:"hostname"`

	// HTTPFilters are optional per-virtual-host HTTP filters.
	HTTPFilters []types.HTTPFilter `json:"httpFilters,omitempty" yaml:"httpFilters,omitempty"`
}

// VirtualHostStatus is the observed state of a virtual host.
type VirtualHostStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

func (r *VirtualHostResource) GetMeta() *ResourceMeta { return &r.Meta }
