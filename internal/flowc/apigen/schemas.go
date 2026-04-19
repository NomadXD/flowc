package apigen

import (
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"

	v1alpha1 "github.com/flowc-labs/flowc/api/v1alpha1"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/internal/flowc/server/handlers"
	"github.com/flowc-labs/flowc/pkg/types"
)

// buildSchemas returns all component schemas for the OpenAPI spec.
// Struct schemas are derived from Go types via reflection; composite
// API-layer schemas (responses, requests, envelopes) are defined manually.
func buildSchemas() openapi3.Schemas {
	r := NewSchemaRegistry()

	// ── Enums ──
	r.RegisterEnum("ConflictPolicy", reflect.TypeOf(""),
		[]interface{}{"strict", "warn", "takeover"})

	// ── Store metadata ──
	r.Register("StoreMeta", reflect.TypeOf(store.StoreMeta{}))

	// ── Envelope types (from handlers) ──
	r.Register("ErrorResponse", reflect.TypeOf(handlers.ErrorResponse{}))
	r.Register("ApplyRequest", reflect.TypeOf(handlers.ApplyRequest{}))
	r.Register("ApplyResult", reflect.TypeOf(handlers.ApplyResult{}))
	r.Register("ApplyResultItem", reflect.TypeOf(handlers.ApplyResultItem{}))

	// ── Resource specs & status (from api/v1) ──
	r.Register("GatewaySpec", reflect.TypeOf(v1alpha1.GatewaySpec{}))
	r.Register("GatewayStatus", reflect.TypeOf(v1alpha1.GatewayStatus{}))
	r.Register("ListenerSpec", reflect.TypeOf(v1alpha1.ListenerSpec{}))
	r.Register("ListenerStatus", reflect.TypeOf(v1alpha1.ListenerStatus{}))
	r.Register("APISpec", reflect.TypeOf(v1alpha1.APISpec{}))
	r.Register("APIStatus", reflect.TypeOf(v1alpha1.APIStatus{}))
	r.Register("DeploymentSpec", reflect.TypeOf(v1alpha1.DeploymentSpec{}))
	r.Register("DeploymentStatus", reflect.TypeOf(v1alpha1.DeploymentStatus{}))
	r.Register("GatewayPolicySpec", reflect.TypeOf(v1alpha1.GatewayPolicySpec{}))
	r.Register("GatewayPolicyStatus", reflect.TypeOf(v1alpha1.GatewayPolicyStatus{}))
	r.Register("APIPolicySpec", reflect.TypeOf(v1alpha1.APIPolicySpec{}))
	r.Register("APIPolicyStatus", reflect.TypeOf(v1alpha1.APIPolicyStatus{}))
	r.Register("BackendPolicySpec", reflect.TypeOf(v1alpha1.BackendPolicySpec{}))
	r.Register("BackendPolicyStatus", reflect.TypeOf(v1alpha1.BackendPolicyStatus{}))

	// ── Common v1 types ──
	r.Register("TLSConfig", reflect.TypeOf(v1alpha1.TLSConfig{}))
	r.Register("UpstreamConfig", reflect.TypeOf(v1alpha1.UpstreamConfig{}))
	r.Register("RoutingConfig", reflect.TypeOf(v1alpha1.RoutingConfig{}))
	r.Register("PolicyInstance", reflect.TypeOf(v1alpha1.PolicyInstance{}))
	r.Register("ParsedInfo", reflect.TypeOf(v1alpha1.ParsedInfo{}))
	r.Register("StrategyConfig", reflect.TypeOf(v1alpha1.StrategyConfig{}))
	r.Register("DeploymentStrategyConfig", reflect.TypeOf(v1alpha1.DeploymentStrategyConfig{}))
	r.Register("CanaryConfig", reflect.TypeOf(v1alpha1.CanaryConfig{}))
	r.Register("BlueGreenConfig", reflect.TypeOf(v1alpha1.BlueGreenConfig{}))
	r.Register("RouteMatchStrategyConfig", reflect.TypeOf(v1alpha1.RouteMatchStrategyConfig{}))
	r.Register("LoadBalancingStrategyConfig", reflect.TypeOf(v1alpha1.LoadBalancingStrategyConfig{}))
	r.Register("HealthCheckConfig", reflect.TypeOf(v1alpha1.HealthCheckConfig{}))
	r.Register("RetryStrategyConfig", reflect.TypeOf(v1alpha1.RetryStrategyConfig{}))
	r.Register("RateLimitStrategyConfig", reflect.TypeOf(v1alpha1.RateLimitStrategyConfig{}))
	r.Register("ObservabilityStrategyConfig", reflect.TypeOf(v1alpha1.ObservabilityStrategyConfig{}))
	r.Register("AccessLogsConfig", reflect.TypeOf(v1alpha1.AccessLogsConfig{}))
	r.Register("DeploymentGatewayRef", reflect.TypeOf(v1alpha1.DeploymentGatewayRef{}))
	r.Register("PolicyTargetRef", reflect.TypeOf(v1alpha1.PolicyTargetRef{}))
	r.Register("CustomFilter", reflect.TypeOf(v1alpha1.CustomFilter{}))

	// ── Shared types (pkg/types) ──
	r.Register("TypesUpstreamConfig", reflect.TypeOf(types.UpstreamConfig{}))
	r.Register("HTTPFilter", reflect.TypeOf(types.HTTPFilter{}))

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
		{"Gateway", "GatewaySpec", "GatewayStatus"},
		{"Listener", "ListenerSpec", "ListenerStatus"},
		{"API", "APISpec", "APIStatus"},
		{"Deployment", "DeploymentSpec", "DeploymentStatus"},
		{"GatewayPolicy", "GatewayPolicySpec", "GatewayPolicyStatus"},
		{"APIPolicy", "APIPolicySpec", "APIPolicyStatus"},
		{"BackendPolicy", "BackendPolicySpec", "BackendPolicyStatus"},
	} {
		s[rc.name+"Response"] = typedResponseSchema(rc.spec, rc.status)
		s[rc.name+"ListResponse"] = typedListResponseSchema(rc.name + "Response")
		s[rc.name+"PutRequest"] = putRequestSchema(rc.spec)
	}

	s["HealthResponse"] = healthResponseSchema()
	s["DeleteResponse"] = deleteResponseSchema()

	return s
}

// --- Composite schema builders ---

func typedResponseSchema(specRef, statusRef string) *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"kind", "metadata", "spec"},
		openapi3.Schemas{
			"apiVersion": strSchema(),
			"kind":       strSchema(),
			"metadata":   schemaRef("StoreMeta"),
			"spec":       schemaRef(specRef),
			"status":     schemaRef(statusRef),
		},
	)
}

func typedListResponseSchema(itemRef string) *openapi3.SchemaRef {
	return newObjectSchema(
		[]string{"kind", "items"},
		openapi3.Schemas{
			"apiVersion": strSchema(),
			"kind":       strSchema(),
			"items":      newArraySchema(schemaRef(itemRef)),
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
