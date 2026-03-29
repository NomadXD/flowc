package apigen

import (
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/server/handlers"
	"github.com/flowc-labs/flowc/pkg/types"
)

// buildSchemas returns all component schemas for the OpenAPI spec.
// Struct schemas are derived from Go types via reflection; composite
// API-layer schemas (responses, requests, envelopes) are defined manually.
func buildSchemas() openapi3.Schemas {
	r := NewSchemaRegistry()

	// ── Enums (can't be discovered via reflection) ──
	r.RegisterEnum("ResourceKind", reflect.TypeOf(resource.ResourceKind("")),
		[]interface{}{"Gateway", "GatewayProfile", "Listener", "Environment", "API", "Deployment"})
	r.RegisterEnum("ConflictPolicy", reflect.TypeOf(resource.ConflictPolicy("")),
		[]interface{}{"strict", "warn", "takeover"})

	// ── Core types ──
	r.Register("ResourceMeta", reflect.TypeOf(resource.ResourceMeta{}))
	r.Register("Condition", reflect.TypeOf(resource.Condition{}))

	// ── Envelope types ──
	r.Register("ErrorResponse", reflect.TypeOf(resource.ErrorResponse{}))
	r.Register("ApplyRequest", reflect.TypeOf(resource.ApplyRequest{}))
	r.Register("ApplyResult", reflect.TypeOf(resource.ApplyResult{}))
	r.Register("ApplyResultItem", reflect.TypeOf(resource.ApplyResultItem{}))

	// ── Resource specs & status ──
	r.Register("GatewayProfileSpec", reflect.TypeOf(resource.GatewayProfileSpec{}))
	r.Register("GatewayProfileStatus", reflect.TypeOf(resource.GatewayProfileStatus{}))
	r.Register("ListenerPreset", reflect.TypeOf(resource.ListenerPreset{}))
	r.Register("BootstrapConfig", reflect.TypeOf(resource.BootstrapConfig{}))
	r.Register("GatewaySpec", reflect.TypeOf(resource.GatewaySpec{}))
	r.Register("GatewayStatus", reflect.TypeOf(resource.GatewayStatus{}))
	r.Register("ListenerSpec", reflect.TypeOf(resource.ListenerSpec{}))
	r.Register("TLSConfig", reflect.TypeOf(resource.TLSConfig{}))
	r.Register("ListenerStatus", reflect.TypeOf(resource.ListenerStatus{}))
	r.Register("EnvironmentSpec", reflect.TypeOf(resource.EnvironmentSpec{}))
	r.Register("EnvironmentStatus", reflect.TypeOf(resource.EnvironmentStatus{}))
	r.Register("APISpec", reflect.TypeOf(resource.APISpec{}))
	r.Register("RoutingConfig", reflect.TypeOf(resource.RoutingConfig{}))
	r.Register("PolicyInstance", reflect.TypeOf(resource.PolicyInstance{}))
	r.Register("ParsedInfo", reflect.TypeOf(resource.ParsedInfo{}))
	r.Register("APIStatus", reflect.TypeOf(resource.APIStatus{}))
	r.Register("DeploymentSpec", reflect.TypeOf(resource.DeploymentSpec{}))
	r.Register("DeploymentStatus", reflect.TypeOf(resource.DeploymentStatus{}))

	// ── Shared types (pkg/types) ──
	r.Register("UpstreamConfig", reflect.TypeOf(types.UpstreamConfig{}))
	r.Register("HTTPFilter", reflect.TypeOf(types.HTTPFilter{}))
	r.Register("StrategyConfig", reflect.TypeOf(types.StrategyConfig{}))
	r.Register("DeploymentStrategyConfig", reflect.TypeOf(types.DeploymentStrategyConfig{}))
	r.Register("CanaryConfig", reflect.TypeOf(types.CanaryConfig{}))
	r.Register("BlueGreenConfig", reflect.TypeOf(types.BlueGreenConfig{}))
	r.Register("MatchCriteria", reflect.TypeOf(types.MatchCriteria{}))
	r.Register("RouteMatchStrategyConfig", reflect.TypeOf(types.RouteMatchStrategyConfig{}))
	r.Register("LoadBalancingStrategyConfig", reflect.TypeOf(types.LoadBalancingStrategyConfig{}))
	r.Register("HealthCheckConfig", reflect.TypeOf(types.HealthCheckConfig{}))
	r.Register("RetryStrategyConfig", reflect.TypeOf(types.RetryStrategyConfig{}))
	r.Register("RateLimitStrategyConfig", reflect.TypeOf(types.RateLimitStrategyConfig{}))
	r.Register("ObservabilityStrategyConfig", reflect.TypeOf(types.ObservabilityStrategyConfig{}))
	r.Register("TracingConfig", reflect.TypeOf(types.TracingConfig{}))
	r.Register("MetricsConfig", reflect.TypeOf(types.MetricsConfig{}))
	r.Register("AccessLogsConfig", reflect.TypeOf(types.AccessLogsConfig{}))

	// ── Deploy instruction types (server/handlers) ──
	r.Register("DeployInstructions", reflect.TypeOf(handlers.DeployInstructions{}))
	r.Register("GatewayInfo", reflect.TypeOf(handlers.GatewayInfo{}))
	r.Register("DockerInstructions", reflect.TypeOf(handlers.DockerInstructions{}))
	r.Register("K8sInstructions", reflect.TypeOf(handlers.K8sInstructions{}))

	// Build all registered struct schemas (after all Register calls
	// so cross-references resolve to $ref).
	r.BuildAll()

	// ── Composite API-layer schemas (not derivable from Go types) ──
	s := r.Schemas()

	// Typed response wrappers per resource kind
	type resCfg struct {
		name, spec, status string
	}
	for _, rc := range []resCfg{
		{"GatewayProfile", "GatewayProfileSpec", "GatewayProfileStatus"},
		{"Gateway", "GatewaySpec", "GatewayStatus"},
		{"Listener", "ListenerSpec", "ListenerStatus"},
		{"Environment", "EnvironmentSpec", "EnvironmentStatus"},
		{"API", "APISpec", "APIStatus"},
		{"Deployment", "DeploymentSpec", "DeploymentStatus"},
	} {
		s[rc.name+"Response"] = typedResponseSchema(rc.spec, rc.status)
		s[rc.name+"ListResponse"] = typedListResponseSchema(rc.name + "Response")
		s[rc.name+"PutRequest"] = putRequestSchema(rc.spec)
	}

	s["HealthResponse"] = healthResponseSchema()
	s["DeleteResponse"] = deleteResponseSchema()

	return s
}

// ─── Composite schema builders ──────────────────────────────────────

func typedResponseSchema(specRef, statusRef string) *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"kind", "metadata", "spec"},
		openapi3.Schemas{
			"kind":     schemaRef("ResourceKind"),
			"metadata": schemaRef("ResourceMeta"),
			"spec":     schemaRef(specRef),
			"status":   schemaRef(statusRef),
		},
	)
}

func typedListResponseSchema(itemRef string) *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"kind", "items", "total"},
		openapi3.Schemas{
			"kind":  strSchema(),
			"items": newArraySchema(schemaRef(itemRef)),
			"total": intSchema(),
		},
	)
}

func putRequestSchema(specRef string) *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"spec"},
		openapi3.Schemas{
			"metadata": newObjectSchema(nil, openapi3.Schemas{
				"conflictPolicy": schemaRef("ConflictPolicy"),
				"labels":         stringMapSchema(),
			}),
			"spec": schemaRef(specRef),
		},
	)
}

func healthResponseSchema() *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"status", "timestamp", "version", "uptime"},
		openapi3.Schemas{
			"status":    strSchema(),
			"timestamp": dateTimeSchema(),
			"version":   strSchema(),
			"uptime":    strSchema(),
		},
	)
}

func deleteResponseSchema() *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"message"},
		openapi3.Schemas{
			"message": strSchema(),
		},
	)
}
