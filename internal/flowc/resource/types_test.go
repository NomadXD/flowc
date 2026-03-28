package resource

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResourceKey_String(t *testing.T) {
	key := ResourceKey{Kind: KindGateway, Project: "default", Name: "my-gw"}
	want := "Gateway/default/my-gw"
	if got := key.String(); got != want {
		t.Errorf("ResourceKey.String() = %q, want %q", got, want)
	}
}

func TestResourceMeta_Key(t *testing.T) {
	m := ResourceMeta{Kind: KindAPI, Project: "prod", Name: "petstore"}
	key := m.Key()
	if key.Kind != KindAPI || key.Project != "prod" || key.Name != "petstore" {
		t.Errorf("unexpected key: %+v", key)
	}
}

func TestIsValidKind(t *testing.T) {
	tests := []struct {
		kind ResourceKind
		want bool
	}{
		{KindGateway, true},
		{KindListener, true},
		{KindEnvironment, true},
		{KindAPI, true},
		{KindDeployment, true},
		{"Invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsValidKind(tt.kind); got != tt.want {
			t.Errorf("IsValidKind(%q) = %v, want %v", tt.kind, got, tt.want)
		}
	}
}

func TestResourceMeta_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	meta := ResourceMeta{
		Kind:           KindGateway,
		Name:           "test-gw",
		Project:        "default",
		Revision:       3,
		ManagedBy:      "cli",
		ConflictPolicy: ConflictStrict,
		Labels:         map[string]string{"env": "prod"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ResourceMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Kind != meta.Kind || got.Name != meta.Name || got.Revision != meta.Revision {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
	if got.Labels["env"] != "prod" {
		t.Errorf("labels lost: %v", got.Labels)
	}
}

func TestSetCondition_AddNew(t *testing.T) {
	var conditions []Condition
	c := Condition{Type: "Ready", Status: "True", Reason: "OK"}
	conditions = SetCondition(conditions, c)
	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conditions))
	}
	if conditions[0].Type != "Ready" || conditions[0].Status != "True" {
		t.Errorf("unexpected condition: %+v", conditions[0])
	}
}

func TestSetCondition_Update(t *testing.T) {
	conditions := []Condition{
		{Type: "Ready", Status: "False", Reason: "Pending", LastTransitionTime: time.Now().Add(-time.Minute)},
	}
	c := Condition{Type: "Ready", Status: "True", Reason: "Deployed"}
	conditions = SetCondition(conditions, c)
	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conditions))
	}
	if conditions[0].Status != "True" || conditions[0].Reason != "Deployed" {
		t.Errorf("condition not updated: %+v", conditions[0])
	}
}
