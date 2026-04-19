package apigen

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestSchemaRegistry_Primitives(t *testing.T) {
	type Simple struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Score   float64 `json:"score"`
		Active  bool    `json:"active"`
		Port    uint32  `json:"port"`
		Comment string  `json:"comment,omitempty"`
	}

	r := NewSchemaRegistry()
	r.Register("Simple", reflect.TypeOf(Simple{}))
	r.BuildAll()

	s := r.Schemas()
	schema, ok := s["Simple"]
	if !ok {
		t.Fatal("missing schema Simple")
	}

	props := schema.Value.Properties
	if props["name"] == nil || props["age"] == nil || props["score"] == nil || props["active"] == nil || props["port"] == nil || props["comment"] == nil {
		t.Error("missing expected properties")
	}

	// name, age, score, active, port are required (no omitempty)
	// comment is NOT required (omitempty)
	required := schema.Value.Required
	assertContains(t, required, "name")
	assertContains(t, required, "age")
	assertContains(t, required, "score")
	assertContains(t, required, "active")
	assertContains(t, required, "port")
	assertNotContains(t, required, "comment")
}

func TestSchemaRegistry_PointerFields(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Required string `json:"required"`
		Optional *Inner `json:"optional,omitempty"`
		Nullable *Inner `json:"nullable"`
	}

	r := NewSchemaRegistry()
	r.Register("Inner", reflect.TypeOf(Inner{}))
	r.Register("Outer", reflect.TypeOf(Outer{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Outer"]

	// "required" field is required; pointer fields are not
	assertContains(t, schema.Value.Required, "required")
	assertNotContains(t, schema.Value.Required, "optional")
	assertNotContains(t, schema.Value.Required, "nullable")

	// nullable should reference Inner via $ref
	ref := schema.Value.Properties["nullable"]
	if ref.Ref != "#/components/schemas/Inner" {
		t.Errorf("expected $ref to Inner, got %q", ref.Ref)
	}
}

func TestSchemaRegistry_NestedRef(t *testing.T) {
	type Child struct {
		Name string `json:"name"`
	}
	type Parent struct {
		Child Child `json:"child"`
	}

	r := NewSchemaRegistry()
	r.Register("Child", reflect.TypeOf(Child{}))
	r.Register("Parent", reflect.TypeOf(Parent{}))
	r.BuildAll()

	s := r.Schemas()
	parent := s["Parent"]
	childRef := parent.Value.Properties["child"]
	if childRef.Ref != "#/components/schemas/Child" {
		t.Errorf("expected $ref to Child, got %q", childRef.Ref)
	}
}

func TestSchemaRegistry_Enum(t *testing.T) {
	type Status string

	r := NewSchemaRegistry()
	r.RegisterEnum("Status", reflect.TypeOf(Status("")),
		[]interface{}{"active", "inactive", "pending"})

	s := r.Schemas()
	schema, ok := s["Status"]
	if !ok {
		t.Fatal("missing schema Status")
	}

	if len(schema.Value.Enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(schema.Value.Enum))
	}
}

func TestSchemaRegistry_EnumRef(t *testing.T) {
	type Kind string
	type Resource struct {
		Kind Kind   `json:"kind"`
		Name string `json:"name"`
	}

	r := NewSchemaRegistry()
	r.RegisterEnum("Kind", reflect.TypeOf(Kind("")),
		[]interface{}{"A", "B"})
	r.Register("Resource", reflect.TypeOf(Resource{}))
	r.BuildAll()

	s := r.Schemas()
	res := s["Resource"]
	kindRef := res.Value.Properties["kind"]
	if kindRef.Ref != "#/components/schemas/Kind" {
		t.Errorf("expected $ref to Kind, got %q", kindRef.Ref)
	}
}

func TestSchemaRegistry_Maps(t *testing.T) {
	type Config struct {
		Labels   map[string]string      `json:"labels"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	r := NewSchemaRegistry()
	r.Register("Config", reflect.TypeOf(Config{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Config"]

	labels := schema.Value.Properties["labels"]
	if labels.Value.Type.Slice()[0] != "object" {
		t.Error("labels should be object type")
	}
	if labels.Value.AdditionalProperties.Schema == nil {
		t.Error("labels should have additionalProperties schema")
	}

	metadata := schema.Value.Properties["metadata"]
	if metadata.Value.AdditionalProperties.Has == nil || !*metadata.Value.AdditionalProperties.Has {
		t.Error("metadata should be freeform object")
	}
}

func TestSchemaRegistry_Slices(t *testing.T) {
	type Item struct {
		Tags []string `json:"tags"`
	}

	r := NewSchemaRegistry()
	r.Register("Item", reflect.TypeOf(Item{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Item"]
	tags := schema.Value.Properties["tags"]
	if tags.Value.Type.Slice()[0] != "array" {
		t.Error("tags should be array type")
	}
}

func TestSchemaRegistry_DateTime(t *testing.T) {
	type Event struct {
		At time.Time `json:"at"`
	}

	r := NewSchemaRegistry()
	r.Register("Event", reflect.TypeOf(Event{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Event"]
	at := schema.Value.Properties["at"]
	if at.Value.Format != "date-time" {
		t.Errorf("expected date-time format, got %q", at.Value.Format)
	}
}

func TestSchemaRegistry_JSONRawMessage(t *testing.T) {
	type Envelope struct {
		Data json.RawMessage `json:"data"`
	}

	r := NewSchemaRegistry()
	r.Register("Envelope", reflect.TypeOf(Envelope{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Envelope"]
	data := schema.Value.Properties["data"]
	if data.Value.AdditionalProperties.Has == nil || !*data.Value.AdditionalProperties.Has {
		t.Error("RawMessage should be freeform object")
	}
}

func TestSchemaRegistry_NoJSONTag(t *testing.T) {
	type Legacy struct {
		FieldName string
		Active    bool
	}

	r := NewSchemaRegistry()
	r.Register("Legacy", reflect.TypeOf(Legacy{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Legacy"]
	if schema.Value.Properties["FieldName"] == nil {
		t.Error("expected Go field name as property name")
	}
	if schema.Value.Properties["Active"] == nil {
		t.Error("expected Go field name as property name")
	}
}

func TestSchemaRegistry_SkipJSONDash(t *testing.T) {
	type Hidden struct {
		Visible string `json:"visible"`
		Hidden  string `json:"-"`
	}

	r := NewSchemaRegistry()
	r.Register("Hidden", reflect.TypeOf(Hidden{}))
	r.BuildAll()

	s := r.Schemas()
	schema := s["Hidden"]
	if schema.Value.Properties["visible"] == nil {
		t.Error("visible should be present")
	}
	if schema.Value.Properties["Hidden"] != nil {
		t.Error("Hidden should be skipped")
	}
}

func TestSchemaRegistry_CrossRefOrder(t *testing.T) {
	// Register Parent before Child — should still produce $ref
	type Child struct {
		Name string `json:"name"`
	}
	type Parent struct {
		Child Child `json:"child"`
	}

	r := NewSchemaRegistry()
	r.Register("Parent", reflect.TypeOf(Parent{}))
	r.Register("Child", reflect.TypeOf(Child{}))
	r.BuildAll()

	s := r.Schemas()
	parent := s["Parent"]
	childRef := parent.Value.Properties["child"]
	if childRef.Ref != "#/components/schemas/Child" {
		t.Errorf("expected $ref to Child even with reversed registration order, got %q", childRef.Ref)
	}
}

// ─── Helpers ────────────────────────────────────────────────────────

func assertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("expected %q in %v", item, slice)
}

func assertNotContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			t.Errorf("did not expect %q in %v", item, slice)
			return
		}
	}
}
