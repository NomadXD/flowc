package resource

import "github.com/flowc-labs/flowc/pkg/types"

// GatewayProfileResource represents a reusable gateway profile that encodes
// the personality of a gateway type (edge, mediation, sidecar, egress, AI).
// Profiles carry sensible strategy defaults, listener presets, and bootstrap
// configuration. A Gateway references a profile via spec.profileRef.
type GatewayProfileResource struct {
	Meta   ResourceMeta         `json:"metadata" yaml:"metadata"`
	Spec   GatewayProfileSpec   `json:"spec" yaml:"spec"`
	Status GatewayProfileStatus `json:"status" yaml:"status"`
}

// GatewayProfileSpec defines the desired state of a gateway profile.
type GatewayProfileSpec struct {
	// DisplayName is a human-friendly name for the profile.
	DisplayName string `json:"displayName" yaml:"displayName"`

	// Description describes the profile's intended use case.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// ProfileType is the canonical type: edge, mediation, sidecar, egress, ai, custom.
	ProfileType string `json:"profileType" yaml:"profileType"`

	// DefaultStrategy provides strategy defaults for gateways using this profile.
	// Precedence: API config > Gateway defaults > Profile defaults > Builtin defaults.
	DefaultStrategy *types.StrategyConfig `json:"defaultStrategy,omitempty" yaml:"defaultStrategy,omitempty"`

	// ListenerPresets defines recommended listener configurations for this profile.
	ListenerPresets []ListenerPreset `json:"listenerPresets,omitempty" yaml:"listenerPresets,omitempty"`

	// DefaultHTTPFilters are filters applied by default to environments on this profile.
	DefaultHTTPFilters []types.HTTPFilter `json:"defaultHttpFilters,omitempty" yaml:"defaultHttpFilters,omitempty"`

	// AllowedPolicies constrains which policy types can be used.
	// Empty means all policies are allowed.
	AllowedPolicies []string `json:"allowedPolicies,omitempty" yaml:"allowedPolicies,omitempty"`

	// Bootstrap contains the Envoy bootstrap template configuration.
	Bootstrap *BootstrapConfig `json:"bootstrap,omitempty" yaml:"bootstrap,omitempty"`

	// EnvoyImage is the recommended Envoy Docker image for this profile.
	EnvoyImage string `json:"envoyImage,omitempty" yaml:"envoyImage,omitempty"`
}

// ListenerPreset is a recommended listener configuration for a profile.
type ListenerPreset struct {
	Name    string     `json:"name" yaml:"name"`
	Port    uint32     `json:"port" yaml:"port"`
	TLS     *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
	HTTP2   bool       `json:"http2,omitempty" yaml:"http2,omitempty"`
	Address string     `json:"address,omitempty" yaml:"address,omitempty"`
}

// BootstrapConfig contains Envoy bootstrap template configuration.
type BootstrapConfig struct {
	// AdminPort is the Envoy admin interface port.
	AdminPort uint32 `json:"adminPort,omitempty" yaml:"adminPort,omitempty"`

	// XDSClusterName is the cluster name used to connect to the control plane.
	XDSClusterName string `json:"xdsClusterName,omitempty" yaml:"xdsClusterName,omitempty"`

	// StaticResources allows arbitrary static resource config in the bootstrap.
	StaticResources map[string]interface{} `json:"staticResources,omitempty" yaml:"staticResources,omitempty"`
}

// GatewayProfileStatus is the observed state of a gateway profile.
type GatewayProfileStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	// GatewayCount tracks how many gateways reference this profile.
	GatewayCount int `json:"gatewayCount,omitempty" yaml:"gatewayCount,omitempty"`
}

func (r *GatewayProfileResource) GetMeta() *ResourceMeta { return &r.Meta }
