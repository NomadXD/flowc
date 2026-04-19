package apigen

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// SchemaRegistry builds OpenAPI component schemas from Go types via reflection.
type SchemaRegistry struct {
	schemas  openapi3.Schemas
	seen     map[reflect.Type]string
	enums    map[reflect.Type][]interface{}
	pending  []pendingType // types registered but not yet built
}

type pendingType struct {
	name string
	typ  reflect.Type
}

// NewSchemaRegistry creates a new registry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		schemas: make(openapi3.Schemas),
		seen:    make(map[reflect.Type]string),
		enums:   make(map[reflect.Type][]interface{}),
	}
}

// RegisterEnum registers enum values for a named string type.
// Creates a named component schema and registers the type for $ref resolution.
func (r *SchemaRegistry) RegisterEnum(name string, typ reflect.Type, values []interface{}) {
	r.enums[typ] = values
	r.seen[typ] = name
	r.schemas[name] = openapi3.NewSchemaRef("", &openapi3.Schema{
		Type: &openapi3.Types{openapi3.TypeString},
		Enum: values,
	})
}

// Register queues a named type for schema generation.
// The type is immediately added to the $ref lookup table so that other types
// referencing it will produce $ref links regardless of registration order.
// Actual schema building is deferred to BuildAll().
func (r *SchemaRegistry) Register(name string, typ reflect.Type) {
	r.seen[typ] = name
	r.pending = append(r.pending, pendingType{name, typ})
}

// BuildAll builds schemas for all registered types. Must be called after all
// Register() calls so that cross-references between types resolve to $ref.
func (r *SchemaRegistry) BuildAll() {
	for _, p := range r.pending {
		r.schemas[p.name] = r.buildSchema(p.typ)
	}
	r.pending = nil
}

// Schemas returns all registered component schemas.
func (r *SchemaRegistry) Schemas() openapi3.Schemas {
	return r.schemas
}

// schemaFor returns a SchemaRef for a type. Named types already in the
// registry return a $ref; others are built inline.
func (r *SchemaRegistry) schemaFor(t reflect.Type) *openapi3.SchemaRef {
	// Dereference pointer
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Named type in registry (enum or struct) → $ref
	if name, ok := r.seen[t]; ok {
		return schemaRef(name)
	}

	// Well-known types
	switch t {
	case reflect.TypeOf(time.Time{}):
		return openapi3.NewSchemaRef("", openapi3.NewDateTimeSchema())
	case reflect.TypeOf(json.RawMessage{}):
		tr := true
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type:                 &openapi3.Types{openapi3.TypeObject},
			AdditionalProperties: openapi3.AdditionalProperties{Has: &tr},
		})
	}

	// Primitives
	switch t.Kind() {
	case reflect.String:
		return openapi3.NewSchemaRef("", openapi3.NewStringSchema())
	case reflect.Bool:
		return openapi3.NewSchemaRef("", openapi3.NewBoolSchema())
	case reflect.Int, reflect.Int64:
		return openapi3.NewSchemaRef("", openapi3.NewInt64Schema())
	case reflect.Int32, reflect.Uint32:
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type:   &openapi3.Types{openapi3.TypeInteger},
			Format: "int32",
			Min:    ptrFloat64(0),
		})
	case reflect.Float64:
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type:   &openapi3.Types{openapi3.TypeNumber},
			Format: "double",
		})
	case reflect.Int8, reflect.Int16, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint64:
		return openapi3.NewSchemaRef("", openapi3.NewIntegerSchema())
	case reflect.Float32:
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type:   &openapi3.Types{openapi3.TypeNumber},
			Format: "float",
		})
	}

	// Slice
	if t.Kind() == reflect.Slice {
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type:  &openapi3.Types{openapi3.TypeArray},
			Items: r.schemaFor(t.Elem()),
		})
	}

	// Map
	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		valType := t.Elem()
		if valType.Kind() == reflect.Interface {
			tr := true
			return openapi3.NewSchemaRef("", &openapi3.Schema{
				Type:                 &openapi3.Types{openapi3.TypeObject},
				AdditionalProperties: openapi3.AdditionalProperties{Has: &tr},
			})
		}
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
			AdditionalProperties: openapi3.AdditionalProperties{
				Schema: r.schemaFor(valType),
			},
		})
	}

	// Struct not in registry → build inline
	if t.Kind() == reflect.Struct {
		return r.buildSchema(t)
	}

	// Fallback
	return openapi3.NewSchemaRef("", openapi3.NewStringSchema())
}

// buildSchema builds an object schema from a struct type's fields.
func (r *SchemaRegistry) buildSchema(t reflect.Type) *openapi3.SchemaRef {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return r.schemaFor(t)
	}

	props := make(openapi3.Schemas)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name, opts := parseJSONTag(jsonTag)
		if name == "" {
			name = field.Name
		}

		omitempty := strings.Contains(opts, "omitempty")
		isPointer := field.Type.Kind() == reflect.Ptr

		// A field is required unless it's a pointer or has omitempty.
		if !omitempty && !isPointer {
			required = append(required, name)
		}

		props[name] = r.schemaFor(field.Type)
	}

	schema := &openapi3.Schema{
		Type:       &openapi3.Types{openapi3.TypeObject},
		Properties: props,
	}
	if len(required) > 0 {
		schema.Required = required
	}
	return openapi3.NewSchemaRef("", schema)
}

// parseJSONTag splits a json tag into name and options.
func parseJSONTag(tag string) (string, string) {
	if tag == "" {
		return "", ""
	}
	parts := strings.SplitN(tag, ",", 2)
	name := parts[0]
	opts := ""
	if len(parts) > 1 {
		opts = parts[1]
	}
	return name, opts
}
