package store

import (
	"context"
	"encoding/json"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
)

// StoredResource is the kind-agnostic envelope stored in the store.
type StoredResource struct {
	Meta       resource.ResourceMeta `json:"metadata"`
	SpecJSON   json.RawMessage       `json:"spec"`
	StatusJSON json.RawMessage       `json:"status,omitempty"`
}

// Key returns the resource key for this stored resource.
func (s *StoredResource) Key() resource.ResourceKey {
	return s.Meta.Key()
}

// Clone returns a deep copy of the stored resource.
func (s *StoredResource) Clone() *StoredResource {
	c := &StoredResource{
		Meta: s.Meta,
	}
	if s.SpecJSON != nil {
		c.SpecJSON = make(json.RawMessage, len(s.SpecJSON))
		copy(c.SpecJSON, s.SpecJSON)
	}
	if s.StatusJSON != nil {
		c.StatusJSON = make(json.RawMessage, len(s.StatusJSON))
		copy(c.StatusJSON, s.StatusJSON)
	}
	// Deep copy labels
	if s.Meta.Labels != nil {
		c.Meta.Labels = make(map[string]string, len(s.Meta.Labels))
		for k, v := range s.Meta.Labels {
			c.Meta.Labels[k] = v
		}
	}
	return c
}

// PutOptions controls the behavior of Store.Put.
type PutOptions struct {
	// ExpectedRevision enables optimistic concurrency.
	// If non-zero, the put will fail with ErrRevisionConflict when the
	// stored revision doesn't match.
	ExpectedRevision int64

	// ManagedBy identifies the writer. Combined with ConflictPolicy on the
	// resource to enforce ownership.
	ManagedBy string
}

// DeleteOptions controls the behavior of Store.Delete.
type DeleteOptions struct {
	// ExpectedRevision for optimistic concurrency on delete.
	ExpectedRevision int64
}

// ListFilter selects which resources to return from Store.List.
type ListFilter struct {
	Kind   resource.ResourceKind
	Labels map[string]string // all must match
}

// WatchEventType indicates whether a resource was written or deleted.
type WatchEventType string

const (
	WatchEventPut    WatchEventType = "PUT"
	WatchEventDelete WatchEventType = "DELETE"
)

// WatchEvent represents a change to a stored resource.
type WatchEvent struct {
	Type        WatchEventType
	Resource    *StoredResource
	OldResource *StoredResource // nil for creates
}

// WatchFilter selects which events to receive.
type WatchFilter struct {
	Kind resource.ResourceKind // empty = all kinds
}

// Store is the desired-state store abstraction.
type Store interface {
	// Get retrieves a resource by key.
	Get(ctx context.Context, key resource.ResourceKey) (*StoredResource, error)

	// Put creates or updates a resource. Returns the stored resource with
	// updated metadata (revision, timestamps).
	Put(ctx context.Context, res *StoredResource, opts PutOptions) (*StoredResource, error)

	// Delete removes a resource by key.
	Delete(ctx context.Context, key resource.ResourceKey, opts DeleteOptions) error

	// List retrieves resources matching the filter.
	List(ctx context.Context, filter ListFilter) ([]*StoredResource, error)

	// Watch returns a channel that receives events matching the filter.
	// The channel is closed when the context is cancelled.
	Watch(ctx context.Context, filter WatchFilter) (<-chan WatchEvent, error)
}
