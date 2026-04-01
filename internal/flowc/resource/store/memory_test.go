package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
)

func makeGateway(name string) *StoredResource {
	spec := resource.GatewaySpec{NodeID: "node-" + name}
	specJSON, _ := json.Marshal(spec)
	return &StoredResource{
		Meta: resource.ResourceMeta{
			Kind: resource.KindGateway,
			Name: name,
		},
		SpecJSON: specJSON,
	}
}

func TestPut_New_RevisionIsOne(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	out, err := s.Put(ctx, res, PutOptions{})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if out.Meta.Revision != 1 {
		t.Errorf("expected revision 1, got %d", out.Meta.Revision)
	}
	if out.Meta.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestPut_Existing_RevisionIncrements(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	out1, _ := s.Put(ctx, res, PutOptions{})

	res.SpecJSON = json.RawMessage(`{"nodeId":"node-updated"}`)
	out2, err := s.Put(ctx, res, PutOptions{})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if out2.Meta.Revision != out1.Meta.Revision+1 {
		t.Errorf("expected revision %d, got %d", out1.Meta.Revision+1, out2.Meta.Revision)
	}
	// CreatedAt should be preserved
	if !out2.Meta.CreatedAt.Equal(out1.Meta.CreatedAt) {
		t.Error("CreatedAt should not change on update")
	}
}

func TestPut_StaleRevision_Conflict(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	s.Put(ctx, res, PutOptions{})

	// Try to update with stale revision
	_, err := s.Put(ctx, res, PutOptions{ExpectedRevision: 999})
	if err == nil {
		t.Fatal("expected revision conflict error")
	}
	if !errors.Is(err, resource.ErrRevisionConflict) {
		t.Errorf("expected ErrRevisionConflict, got %v", err)
	}
}

func TestPut_OwnershipStrict_Conflict(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	res.Meta.ManagedBy = "cli"
	res.Meta.ConflictPolicy = resource.ConflictStrict
	s.Put(ctx, res, PutOptions{ManagedBy: "cli"})

	// Different writer, strict policy
	_, err := s.Put(ctx, res, PutOptions{ManagedBy: "k8s-operator"})
	if err == nil {
		t.Fatal("expected ownership conflict")
	}
	if !errors.Is(err, resource.ErrOwnershipConflict) {
		t.Errorf("expected ErrOwnershipConflict, got %v", err)
	}
}

func TestPut_OwnershipTakeover_OK(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	res.Meta.ManagedBy = "cli"
	res.Meta.ConflictPolicy = resource.ConflictTakeover
	s.Put(ctx, res, PutOptions{ManagedBy: "cli"})

	// Different writer, takeover policy
	out, err := s.Put(ctx, res, PutOptions{ManagedBy: "k8s-operator"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Meta.ManagedBy != "k8s-operator" {
		t.Errorf("expected managedBy=k8s-operator, got %s", out.Meta.ManagedBy)
	}
}

func TestGet_Exists(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	s.Put(ctx, res, PutOptions{})

	got, err := s.Get(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "gw-a"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Meta.Name != "gw-a" {
		t.Errorf("expected name gw-a, got %s", got.Meta.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.Get(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "nonexistent"})
	if !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_Exists(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	s.Put(ctx, res, PutOptions{})

	err := s.Delete(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "gw-a"}, DeleteOptions{})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = s.Get(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "gw-a"})
	if !errors.Is(err, resource.ErrNotFound) {
		t.Error("resource should be gone after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.Delete(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "nonexistent"}, DeleteOptions{})
	if !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestList_KindFilter(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	s.Put(ctx, makeGateway("gw-a"), PutOptions{})
	s.Put(ctx, makeGateway("gw-b"), PutOptions{})
	s.Put(ctx, makeGateway("gw-c"), PutOptions{})

	// List by kind
	items, err := s.List(ctx, ListFilter{Kind: resource.KindGateway})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestList_LabelFilter(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	gw := makeGateway("gw-a")
	gw.Meta.Labels = map[string]string{"env": "prod"}
	s.Put(ctx, gw, PutOptions{})

	gw2 := makeGateway("gw-b")
	gw2.Meta.Labels = map[string]string{"env": "staging"}
	s.Put(ctx, gw2, PutOptions{})

	items, _ := s.List(ctx, ListFilter{Labels: map[string]string{"env": "prod"}})
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestWatch_ReceivesPutAndDelete(t *testing.T) {
	s := NewMemoryStore()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := s.Watch(ctx, WatchFilter{Kind: resource.KindGateway})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	// Put
	gw := makeGateway("gw-a")
	s.Put(ctx, gw, PutOptions{})

	select {
	case event := <-ch:
		if event.Type != WatchEventPut {
			t.Errorf("expected PUT event, got %s", event.Type)
		}
		if event.Resource.Meta.Name != "gw-a" {
			t.Errorf("expected gw-a, got %s", event.Resource.Meta.Name)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for PUT event")
	}

	// Delete
	s.Delete(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "gw-a"}, DeleteOptions{})

	select {
	case event := <-ch:
		if event.Type != WatchEventDelete {
			t.Errorf("expected DELETE event, got %s", event.Type)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for DELETE event")
	}
}

func TestWatch_FilterByKind(t *testing.T) {
	s := NewMemoryStore()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, _ := s.Watch(ctx, WatchFilter{Kind: resource.KindListener})

	// Put a gateway — should not match
	s.Put(ctx, makeGateway("gw-a"), PutOptions{})

	// Put a listener — should match
	listenerSpec, _ := json.Marshal(resource.ListenerSpec{GatewayRef: "gw-a", Port: 8080})
	listener := &StoredResource{
		Meta:     resource.ResourceMeta{Kind: resource.KindListener, Name: "http"},
		SpecJSON: listenerSpec,
	}
	s.Put(ctx, listener, PutOptions{})

	select {
	case event := <-ch:
		if event.Resource.Meta.Kind != resource.KindListener {
			t.Errorf("expected Listener event, got %s", event.Resource.Meta.Kind)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for Listener event")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := "gw-a" // same resource, concurrent writes
			res := makeGateway(name)
			s.Put(ctx, res, PutOptions{})
			s.Get(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: name})
			s.List(ctx, ListFilter{Kind: resource.KindGateway})
		}(i)
	}
	wg.Wait()

	// Verify consistency
	got, err := s.Get(ctx, resource.ResourceKey{Kind: resource.KindGateway, Name: "gw-a"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Meta.Revision < 1 {
		t.Errorf("expected revision >= 1, got %d", got.Meta.Revision)
	}
}

func TestTypedStore_GatewayRoundTrip(t *testing.T) {
	s := NewMemoryStore()
	ts := NewTypedStore(s)
	ctx := context.Background()

	gw := &resource.GatewayResource{
		Meta: resource.ResourceMeta{Kind: resource.KindGateway, Name: "my-gw"},
		Spec: resource.GatewaySpec{NodeID: "envoy-1"},
	}

	out, err := ts.PutGateway(ctx, gw, PutOptions{})
	if err != nil {
		t.Fatalf("PutGateway: %v", err)
	}
	if out.Spec.NodeID != "envoy-1" {
		t.Errorf("expected nodeId=envoy-1, got %s", out.Spec.NodeID)
	}

	got, err := ts.GetGateway(ctx, "my-gw")
	if err != nil {
		t.Fatalf("GetGateway: %v", err)
	}
	if got.Spec.NodeID != "envoy-1" {
		t.Errorf("expected nodeId=envoy-1, got %s", got.Spec.NodeID)
	}

	list, err := ts.ListGateways(ctx)
	if err != nil {
		t.Fatalf("ListGateways: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 gateway, got %d", len(list))
	}
}

func TestTypedStore_DeploymentRoundTrip(t *testing.T) {
	s := NewMemoryStore()
	ts := NewTypedStore(s)
	ctx := context.Background()

	dep := &resource.DeploymentResource{
		Meta: resource.ResourceMeta{Kind: resource.KindDeployment, Name: "petstore-prod"},
		Spec: resource.DeploymentSpec{
			APIRef: "petstore",
			Gateway: resource.DeploymentGatewayRef{
				Name:        "my-gw",
				Listener:    "http",
				VirtualHost: "production",
			},
		},
	}

	out, err := ts.PutDeployment(ctx, dep, PutOptions{})
	if err != nil {
		t.Fatalf("PutDeployment: %v", err)
	}
	if out.Spec.APIRef != "petstore" {
		t.Errorf("expected apiRef=petstore, got %s", out.Spec.APIRef)
	}

	got, err := ts.GetDeployment(ctx, "petstore-prod")
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if got.Spec.Gateway.Name != "my-gw" {
		t.Errorf("expected gateway.name=my-gw, got %s", got.Spec.Gateway.Name)
	}
}
