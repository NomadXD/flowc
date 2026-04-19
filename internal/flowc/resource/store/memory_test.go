package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func makeGateway(name string) *StoredResource {
	spec := map[string]string{"nodeId": "node-" + name}
	specJSON, _ := json.Marshal(spec)
	return &StoredResource{
		Meta: StoreMeta{
			Kind: "Gateway",
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
	if !errors.Is(err, ErrRevisionConflict) {
		t.Errorf("expected ErrRevisionConflict, got %v", err)
	}
}

func TestPut_OwnershipStrict_Conflict(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	res.Meta.ManagedBy = "cli"
	res.Meta.ConflictPolicy = ConflictStrict
	s.Put(ctx, res, PutOptions{ManagedBy: "cli"})

	// Different writer, strict policy
	_, err := s.Put(ctx, res, PutOptions{ManagedBy: "k8s-operator"})
	if err == nil {
		t.Fatal("expected ownership conflict")
	}
	if !errors.Is(err, ErrOwnershipConflict) {
		t.Errorf("expected ErrOwnershipConflict, got %v", err)
	}
}

func TestPut_OwnershipTakeover_OK(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	res.Meta.ManagedBy = "cli"
	res.Meta.ConflictPolicy = ConflictTakeover
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

	got, err := s.Get(ctx, ResourceKey{Kind: "Gateway", Name: "gw-a"})
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

	_, err := s.Get(ctx, ResourceKey{Kind: "Gateway", Name: "nonexistent"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_Exists(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	res := makeGateway("gw-a")
	s.Put(ctx, res, PutOptions{})

	err := s.Delete(ctx, ResourceKey{Kind: "Gateway", Name: "gw-a"}, DeleteOptions{})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = s.Get(ctx, ResourceKey{Kind: "Gateway", Name: "gw-a"})
	if !errors.Is(err, ErrNotFound) {
		t.Error("resource should be gone after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.Delete(ctx, ResourceKey{Kind: "Gateway", Name: "nonexistent"}, DeleteOptions{})
	if !errors.Is(err, ErrNotFound) {
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
	items, err := s.List(ctx, ListFilter{Kind: "Gateway"})
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

	ch, err := s.Watch(ctx, WatchFilter{Kind: "Gateway"})
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
	s.Delete(ctx, ResourceKey{Kind: "Gateway", Name: "gw-a"}, DeleteOptions{})

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

	ch, _ := s.Watch(ctx, WatchFilter{Kind: "Listener"})

	// Put a gateway — should not match
	s.Put(ctx, makeGateway("gw-a"), PutOptions{})

	// Put a listener — should match
	listenerSpec, _ := json.Marshal(map[string]interface{}{"gatewayRef": "gw-a", "port": 8080})
	listener := &StoredResource{
		Meta:     StoreMeta{Kind: "Listener", Name: "http"},
		SpecJSON: listenerSpec,
	}
	s.Put(ctx, listener, PutOptions{})

	select {
	case event := <-ch:
		if event.Resource.Meta.Kind != "Listener" {
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
			s.Get(ctx, ResourceKey{Kind: "Gateway", Name: name})
			s.List(ctx, ListFilter{Kind: "Gateway"})
		}(i)
	}
	wg.Wait()

	// Verify consistency
	got, err := s.Get(ctx, ResourceKey{Kind: "Gateway", Name: "gw-a"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Meta.Revision < 1 {
		t.Errorf("expected revision >= 1, got %d", got.Meta.Revision)
	}
}
