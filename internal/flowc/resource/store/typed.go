package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
)

// TypedStore provides typed convenience methods on top of Store.
type TypedStore struct {
	Store Store
}

// NewTypedStore wraps a Store with typed accessors.
func NewTypedStore(s Store) *TypedStore {
	return &TypedStore{Store: s}
}

// --- Helpers ---

func marshalSpec(v interface{}) (json.RawMessage, error) {
	return json.Marshal(v)
}

func marshalStatus(v interface{}) (json.RawMessage, error) {
	return json.Marshal(v)
}

// --- Gateway ---

func (t *TypedStore) GetGateway(ctx context.Context, name string) (*resource.GatewayResource, error) {
	sr, err := t.Store.Get(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: name})
	if err != nil {
		return nil, err
	}
	return unmarshalGateway(sr)
}

func (t *TypedStore) PutGateway(ctx context.Context, r *resource.GatewayResource, opts PutOptions) (*resource.GatewayResource, error) {
	sr, err := toStored(r.Meta, r.Spec, r.Status)
	if err != nil {
		return nil, err
	}
	out, err := t.Store.Put(ctx, sr, opts)
	if err != nil {
		return nil, err
	}
	return unmarshalGateway(out)
}

func (t *TypedStore) ListGateways(ctx context.Context) ([]*resource.GatewayResource, error) {
	items, err := t.Store.List(ctx, ListFilter{Kind: resource.KindGateway})
	if err != nil {
		return nil, err
	}
	result := make([]*resource.GatewayResource, 0, len(items))
	for _, item := range items {
		gw, err := unmarshalGateway(item)
		if err != nil {
			return nil, err
		}
		result = append(result, gw)
	}
	return result, nil
}

func unmarshalGateway(sr *StoredResource) (*resource.GatewayResource, error) {
	r := &resource.GatewayResource{Meta: sr.Meta}
	if err := json.Unmarshal(sr.SpecJSON, &r.Spec); err != nil {
		return nil, fmt.Errorf("unmarshal gateway spec: %w", err)
	}
	if len(sr.StatusJSON) > 0 {
		if err := json.Unmarshal(sr.StatusJSON, &r.Status); err != nil {
			return nil, fmt.Errorf("unmarshal gateway status: %w", err)
		}
	}
	return r, nil
}

// --- GatewayProfile ---

func (t *TypedStore) GetGatewayProfile(ctx context.Context, name string) (*resource.GatewayProfileResource, error) {
	sr, err := t.Store.Get(ctx, resource.ResourceKey{Kind: resource.KindGatewayProfile, Name: name})
	if err != nil {
		return nil, err
	}
	return unmarshalGatewayProfile(sr)
}

func (t *TypedStore) PutGatewayProfile(ctx context.Context, r *resource.GatewayProfileResource, opts PutOptions) (*resource.GatewayProfileResource, error) {
	sr, err := toStored(r.Meta, r.Spec, r.Status)
	if err != nil {
		return nil, err
	}
	out, err := t.Store.Put(ctx, sr, opts)
	if err != nil {
		return nil, err
	}
	return unmarshalGatewayProfile(out)
}

func (t *TypedStore) ListGatewayProfiles(ctx context.Context) ([]*resource.GatewayProfileResource, error) {
	items, err := t.Store.List(ctx, ListFilter{Kind: resource.KindGatewayProfile})
	if err != nil {
		return nil, err
	}
	result := make([]*resource.GatewayProfileResource, 0, len(items))
	for _, item := range items {
		p, err := unmarshalGatewayProfile(item)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func unmarshalGatewayProfile(sr *StoredResource) (*resource.GatewayProfileResource, error) {
	r := &resource.GatewayProfileResource{Meta: sr.Meta}
	if err := json.Unmarshal(sr.SpecJSON, &r.Spec); err != nil {
		return nil, fmt.Errorf("unmarshal gateway profile spec: %w", err)
	}
	if len(sr.StatusJSON) > 0 {
		if err := json.Unmarshal(sr.StatusJSON, &r.Status); err != nil {
			return nil, fmt.Errorf("unmarshal gateway profile status: %w", err)
		}
	}
	return r, nil
}

// --- Listener ---

func (t *TypedStore) GetListener(ctx context.Context, name string) (*resource.ListenerResource, error) {
	sr, err := t.Store.Get(ctx, resource.ResourceKey{Kind: resource.KindListener, Name: name})
	if err != nil {
		return nil, err
	}
	return unmarshalListener(sr)
}

func (t *TypedStore) PutListener(ctx context.Context, r *resource.ListenerResource, opts PutOptions) (*resource.ListenerResource, error) {
	sr, err := toStored(r.Meta, r.Spec, r.Status)
	if err != nil {
		return nil, err
	}
	out, err := t.Store.Put(ctx, sr, opts)
	if err != nil {
		return nil, err
	}
	return unmarshalListener(out)
}

func (t *TypedStore) ListListeners(ctx context.Context) ([]*resource.ListenerResource, error) {
	items, err := t.Store.List(ctx, ListFilter{Kind: resource.KindListener})
	if err != nil {
		return nil, err
	}
	result := make([]*resource.ListenerResource, 0, len(items))
	for _, item := range items {
		l, err := unmarshalListener(item)
		if err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

func unmarshalListener(sr *StoredResource) (*resource.ListenerResource, error) {
	r := &resource.ListenerResource{Meta: sr.Meta}
	if err := json.Unmarshal(sr.SpecJSON, &r.Spec); err != nil {
		return nil, fmt.Errorf("unmarshal listener spec: %w", err)
	}
	if len(sr.StatusJSON) > 0 {
		if err := json.Unmarshal(sr.StatusJSON, &r.Status); err != nil {
			return nil, fmt.Errorf("unmarshal listener status: %w", err)
		}
	}
	return r, nil
}

// --- Environment ---

func (t *TypedStore) GetEnvironment(ctx context.Context, name string) (*resource.EnvironmentResource, error) {
	sr, err := t.Store.Get(ctx, resource.ResourceKey{Kind: resource.KindEnvironment, Name: name})
	if err != nil {
		return nil, err
	}
	return unmarshalEnvironment(sr)
}

func (t *TypedStore) PutEnvironment(ctx context.Context, r *resource.EnvironmentResource, opts PutOptions) (*resource.EnvironmentResource, error) {
	sr, err := toStored(r.Meta, r.Spec, r.Status)
	if err != nil {
		return nil, err
	}
	out, err := t.Store.Put(ctx, sr, opts)
	if err != nil {
		return nil, err
	}
	return unmarshalEnvironment(out)
}

func (t *TypedStore) ListEnvironments(ctx context.Context) ([]*resource.EnvironmentResource, error) {
	items, err := t.Store.List(ctx, ListFilter{Kind: resource.KindEnvironment})
	if err != nil {
		return nil, err
	}
	result := make([]*resource.EnvironmentResource, 0, len(items))
	for _, item := range items {
		e, err := unmarshalEnvironment(item)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func unmarshalEnvironment(sr *StoredResource) (*resource.EnvironmentResource, error) {
	r := &resource.EnvironmentResource{Meta: sr.Meta}
	if err := json.Unmarshal(sr.SpecJSON, &r.Spec); err != nil {
		return nil, fmt.Errorf("unmarshal environment spec: %w", err)
	}
	if len(sr.StatusJSON) > 0 {
		if err := json.Unmarshal(sr.StatusJSON, &r.Status); err != nil {
			return nil, fmt.Errorf("unmarshal environment status: %w", err)
		}
	}
	return r, nil
}

// --- API ---

func (t *TypedStore) GetAPI(ctx context.Context, name string) (*resource.APIResource, error) {
	sr, err := t.Store.Get(ctx, resource.ResourceKey{Kind: resource.KindAPI, Name: name})
	if err != nil {
		return nil, err
	}
	return unmarshalAPI(sr)
}

func (t *TypedStore) PutAPI(ctx context.Context, r *resource.APIResource, opts PutOptions) (*resource.APIResource, error) {
	sr, err := toStored(r.Meta, r.Spec, r.Status)
	if err != nil {
		return nil, err
	}
	out, err := t.Store.Put(ctx, sr, opts)
	if err != nil {
		return nil, err
	}
	return unmarshalAPI(out)
}

func (t *TypedStore) ListAPIs(ctx context.Context) ([]*resource.APIResource, error) {
	items, err := t.Store.List(ctx, ListFilter{Kind: resource.KindAPI})
	if err != nil {
		return nil, err
	}
	result := make([]*resource.APIResource, 0, len(items))
	for _, item := range items {
		a, err := unmarshalAPI(item)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

func unmarshalAPI(sr *StoredResource) (*resource.APIResource, error) {
	r := &resource.APIResource{Meta: sr.Meta}
	if err := json.Unmarshal(sr.SpecJSON, &r.Spec); err != nil {
		return nil, fmt.Errorf("unmarshal api spec: %w", err)
	}
	if len(sr.StatusJSON) > 0 {
		if err := json.Unmarshal(sr.StatusJSON, &r.Status); err != nil {
			return nil, fmt.Errorf("unmarshal api status: %w", err)
		}
	}
	return r, nil
}

// --- Deployment ---

func (t *TypedStore) GetDeployment(ctx context.Context, name string) (*resource.DeploymentResource, error) {
	sr, err := t.Store.Get(ctx, resource.ResourceKey{Kind: resource.KindDeployment, Name: name})
	if err != nil {
		return nil, err
	}
	return unmarshalDeployment(sr)
}

func (t *TypedStore) PutDeployment(ctx context.Context, r *resource.DeploymentResource, opts PutOptions) (*resource.DeploymentResource, error) {
	sr, err := toStored(r.Meta, r.Spec, r.Status)
	if err != nil {
		return nil, err
	}
	out, err := t.Store.Put(ctx, sr, opts)
	if err != nil {
		return nil, err
	}
	return unmarshalDeployment(out)
}

func (t *TypedStore) ListDeployments(ctx context.Context) ([]*resource.DeploymentResource, error) {
	items, err := t.Store.List(ctx, ListFilter{Kind: resource.KindDeployment})
	if err != nil {
		return nil, err
	}
	result := make([]*resource.DeploymentResource, 0, len(items))
	for _, item := range items {
		d, err := unmarshalDeployment(item)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

func unmarshalDeployment(sr *StoredResource) (*resource.DeploymentResource, error) {
	r := &resource.DeploymentResource{Meta: sr.Meta}
	if err := json.Unmarshal(sr.SpecJSON, &r.Spec); err != nil {
		return nil, fmt.Errorf("unmarshal deployment spec: %w", err)
	}
	if len(sr.StatusJSON) > 0 {
		if err := json.Unmarshal(sr.StatusJSON, &r.Status); err != nil {
			return nil, fmt.Errorf("unmarshal deployment status: %w", err)
		}
	}
	return r, nil
}

// --- Generic helpers ---

func toStored(meta resource.ResourceMeta, spec interface{}, status interface{}) (*StoredResource, error) {
	specJSON, err := marshalSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec: %w", err)
	}
	statusJSON, err := marshalStatus(status)
	if err != nil {
		return nil, fmt.Errorf("marshal status: %w", err)
	}
	return &StoredResource{
		Meta:       meta,
		SpecJSON:   specJSON,
		StatusJSON: statusJSON,
	}, nil
}
