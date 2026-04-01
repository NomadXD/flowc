package resource

import (
	"fmt"
	"time"
)

// ResourceKind identifies the type of a resource.
type ResourceKind string

const (
	KindGateway        ResourceKind = "Gateway"
	KindGatewayProfile ResourceKind = "GatewayProfile"
	KindListener       ResourceKind = "Listener"
	KindVirtualHost    ResourceKind = "VirtualHost"
	KindAPI            ResourceKind = "API"
	KindDeployment     ResourceKind = "Deployment"
)

// ValidKinds returns all valid resource kinds.
func ValidKinds() []ResourceKind {
	return []ResourceKind{KindGateway, KindGatewayProfile, KindListener, KindVirtualHost, KindAPI, KindDeployment}
}

// IsValidKind checks if a kind string is valid.
func IsValidKind(k ResourceKind) bool {
	for _, valid := range ValidKinds() {
		if k == valid {
			return true
		}
	}
	return false
}

// ConflictPolicy determines how ownership conflicts are handled.
type ConflictPolicy string

const (
	// ConflictStrict rejects writes from a different manager.
	ConflictStrict ConflictPolicy = "strict"
	// ConflictWarn allows the write but logs a warning.
	ConflictWarn ConflictPolicy = "warn"
	// ConflictTakeover transfers ownership to the new writer.
	ConflictTakeover ConflictPolicy = "takeover"
)

// ResourceMeta is the metadata envelope common to all resources.
type ResourceMeta struct {
	Kind           ResourceKind      `json:"kind" yaml:"kind"`
	Name           string            `json:"name" yaml:"name"`
	Revision       int64             `json:"revision" yaml:"revision"`
	ManagedBy      string            `json:"managedBy,omitempty" yaml:"managedBy,omitempty"`
	ConflictPolicy ConflictPolicy    `json:"conflictPolicy,omitempty" yaml:"conflictPolicy,omitempty"`
	Labels         map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt      time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

// ResourceKey is the unique identity of a resource: (Kind, Name).
type ResourceKey struct {
	Kind ResourceKind `json:"kind" yaml:"kind"`
	Name string       `json:"name" yaml:"name"`
}

// String returns a human-readable key representation.
func (k ResourceKey) String() string {
	return fmt.Sprintf("%s/%s", k.Kind, k.Name)
}

// Key returns the ResourceKey for a ResourceMeta.
func (m *ResourceMeta) Key() ResourceKey {
	return ResourceKey{Kind: m.Kind, Name: m.Name}
}

// Condition describes the status of a resource aspect, K8s-style.
type Condition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
}

// SetCondition adds or updates a condition in a slice, returning the new slice.
func SetCondition(conditions []Condition, c Condition) []Condition {
	for i, existing := range conditions {
		if existing.Type == c.Type {
			if existing.Status != c.Status {
				c.LastTransitionTime = time.Now()
			} else {
				c.LastTransitionTime = existing.LastTransitionTime
			}
			conditions[i] = c
			return conditions
		}
	}
	if c.LastTransitionTime.IsZero() {
		c.LastTransitionTime = time.Now()
	}
	return append(conditions, c)
}
