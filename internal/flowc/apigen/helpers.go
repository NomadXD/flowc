package apigen

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// schemaRef returns a $ref to a component schema.
func schemaRef(name string) *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("#/components/schemas/"+name, nil)
}

// errorResponseRef returns a ResponseRef pointing to the ErrorResponse schema.
func errorResponseRef() *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: openapi3.NewResponse().
			WithDescription("Error response").
			WithJSONSchemaRef(schemaRef("ErrorResponse")),
	}
}

// ifMatchParam returns the If-Match header parameter for optimistic concurrency.
func ifMatchParam() *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:        "If-Match",
			In:          "header",
			Description: "Resource revision for optimistic concurrency control",
			Required:    false,
			Schema:      openapi3.NewSchemaRef("", openapi3.NewInt64Schema()),
		},
	}
}

// managedByParam returns the X-Managed-By header parameter for ownership tracking.
func managedByParam() *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:        "X-Managed-By",
			In:          "header",
			Description: "Identifier of the managing entity for ownership tracking",
			Required:    false,
			Schema:      openapi3.NewSchemaRef("", openapi3.NewStringSchema()),
		},
	}
}

// capitalize returns a string with first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// stringMapSchema returns a schema for map[string]string.
func stringMapSchema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", &openapi3.Schema{
		Type: &openapi3.Types{openapi3.TypeObject},
		AdditionalProperties: openapi3.AdditionalProperties{
			Schema: openapi3.NewSchemaRef("", openapi3.NewStringSchema()),
		},
	})
}

// freeformObjectSchema returns a schema for map[string]interface{}.
func freeformObjectSchema() *openapi3.SchemaRef {
	t := true
	return openapi3.NewSchemaRef("", &openapi3.Schema{
		Type: &openapi3.Types{openapi3.TypeObject},
		AdditionalProperties: openapi3.AdditionalProperties{Has: &t},
	})
}

// newObjectSchema creates an object schema with the given required fields and properties.
func newObjectSchema(required []string, props openapi3.Schemas) *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", &openapi3.Schema{
		Type:       &openapi3.Types{openapi3.TypeObject},
		Required:   required,
		Properties: props,
	})
}

// newArraySchema creates an array schema with the given items ref.
func newArraySchema(itemRef *openapi3.SchemaRef) *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", &openapi3.Schema{
		Type:  &openapi3.Types{openapi3.TypeArray},
		Items: itemRef,
	})
}

// strSchema returns an inline string schema ref.
func strSchema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", openapi3.NewStringSchema())
}

// intSchema returns an inline integer schema ref.
func intSchema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", openapi3.NewIntegerSchema())
}

// int64Schema returns an inline int64 schema ref.
func int64Schema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", openapi3.NewInt64Schema())
}

// uint32Schema returns an inline uint32 (integer) schema ref.
func uint32Schema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", &openapi3.Schema{
		Type:   &openapi3.Types{openapi3.TypeInteger},
		Format: "int32",
		Min:    ptrFloat64(0),
	})
}

// boolSchema returns an inline boolean schema ref.
func boolSchema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", openapi3.NewBoolSchema())
}

// float64Schema returns an inline number (double) schema ref.
func float64Schema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", &openapi3.Schema{
		Type:   &openapi3.Types{openapi3.TypeNumber},
		Format: "double",
	})
}

// dateTimeSchema returns an inline date-time string schema ref.
func dateTimeSchema() *openapi3.SchemaRef {
	return openapi3.NewSchemaRef("", openapi3.NewDateTimeSchema())
}

// ptrFloat64 returns a pointer to a float64.
func ptrFloat64(v float64) *float64 {
	return &v
}
