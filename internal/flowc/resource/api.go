package resource

import "github.com/flowc-labs/flowc/pkg/types"

// APIResource represents a first-class API entity.
type APIResource struct {
	Meta   ResourceMeta `json:"metadata" yaml:"metadata"`
	Spec   APISpec      `json:"spec" yaml:"spec"`
	Status APIStatus    `json:"status" yaml:"status"`
}

// APISpec defines the desired state of an API.
type APISpec struct {
	// Version is the semver version of this API.
	Version string `json:"version" yaml:"version"`

	// DisplayName is a human-friendly display name.
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`

	// Description is a human-readable description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Context is the base path where this API is exposed (e.g., "/petstore").
	Context string `json:"context" yaml:"context"`

	// APIType is the specification type: rest, grpc, graphql, websocket, sse.
	// Auto-detected from SpecContent if empty.
	APIType string `json:"apiType,omitempty" yaml:"apiType,omitempty"`

	// SpecContent holds the full API specification as a string (OpenAPI YAML, proto, etc.).
	SpecContent string `json:"specContent,omitempty" yaml:"specContent,omitempty"`

	// Upstream defines the backend service.
	Upstream types.UpstreamConfig `json:"upstream" yaml:"upstream"`

	// Routing defines route matching behavior.
	Routing RoutingConfig `json:"routing,omitempty" yaml:"routing,omitempty"`

	// PolicyChain is an ordered list of policy instances.
	PolicyChain []PolicyInstance `json:"policyChain,omitempty" yaml:"policyChain,omitempty"`
}

// RoutingConfig defines routing behavior for an API.
type RoutingConfig struct {
	MatchType     string `json:"matchType,omitempty" yaml:"matchType,omitempty"`
	CaseSensitive bool   `json:"caseSensitive,omitempty" yaml:"caseSensitive,omitempty"`
	LoadBalancing string `json:"loadBalancing,omitempty" yaml:"loadBalancing,omitempty"`
}

// PolicyInstance represents an attached policy with configuration.
type PolicyInstance struct {
	ID              string                 `json:"id" yaml:"id"`
	PolicyType      string                 `json:"policyType" yaml:"policyType"`
	Order           int                    `json:"order" yaml:"order"`
	Enabled         bool                   `json:"enabled" yaml:"enabled"`
	InheritanceMode string                 `json:"inheritanceMode,omitempty" yaml:"inheritanceMode,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// ParsedInfo contains lightweight metadata extracted from a parsed specification.
type ParsedInfo struct {
	Title   string   `json:"title,omitempty" yaml:"title,omitempty"`
	Version string   `json:"version,omitempty" yaml:"version,omitempty"`
	Paths   []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	Servers []string `json:"servers,omitempty" yaml:"servers,omitempty"`
}

// APIStatus is the observed state of an API.
type APIStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	ParsedInfo *ParsedInfo `json:"parsedInfo,omitempty" yaml:"parsedInfo,omitempty"`
}

func (r *APIResource) GetMeta() *ResourceMeta { return &r.Meta }
