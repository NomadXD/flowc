package apigen

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestGenerate_ProducesValidSpec(t *testing.T) {
	doc := Generate()
	// Must round-trip through the loader to resolve $ref pointers before validation.
	data, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	loader := openapi3.NewLoader()
	resolved, err := loader.LoadFromData(data)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if err := resolved.Validate(context.Background()); err != nil {
		t.Fatalf("generated spec is invalid: %v", err)
	}
}

func TestGenerate_RoundTrip(t *testing.T) {
	doc := Generate()
	data, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	loader := openapi3.NewLoader()
	reloaded, err := loader.LoadFromData(data)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if err := reloaded.Validate(context.Background()); err != nil {
		t.Fatalf("reloaded spec is invalid: %v", err)
	}
}

func TestGenerate_HasAllPaths(t *testing.T) {
	doc := Generate()
	expected := []string{
		"/health",
		"/api/v1/gateways",
		"/api/v1/gateways/{name}",
		"/api/v1/gateways/{name}/bootstrap",
		"/api/v1/gateways/{name}/deploy",
		"/api/v1/listeners",
		"/api/v1/listeners/{name}",
		"/api/v1/apis",
		"/api/v1/apis/{name}",
		"/api/v1/deployments",
		"/api/v1/deployments/{name}",
		"/api/v1/gatewaypolicies",
		"/api/v1/gatewaypolicies/{name}",
		"/api/v1/apipolicies",
		"/api/v1/apipolicies/{name}",
		"/api/v1/backendpolicies",
		"/api/v1/backendpolicies/{name}",
		"/api/v1/apply",
		"/api/v1/upload",
	}
	for _, p := range expected {
		if doc.Paths.Find(p) == nil {
			t.Errorf("missing path: %s", p)
		}
	}
}

func TestGenerate_HasAllSchemas(t *testing.T) {
	doc := Generate()
	expected := []string{
		"ConflictPolicy", "StoreMeta",
		"ErrorResponse", "ApplyRequest", "ApplyResult", "ApplyResultItem",
		"GatewaySpec", "GatewayStatus",
		"ListenerSpec", "ListenerStatus", "TLSConfig",
		"APISpec", "APIStatus", "RoutingConfig", "PolicyInstance", "ParsedInfo",
		"DeploymentSpec", "DeploymentStatus", "DeploymentGatewayRef",
		"GatewayPolicySpec", "GatewayPolicyStatus",
		"APIPolicySpec", "APIPolicyStatus",
		"BackendPolicySpec", "BackendPolicyStatus",
		"UpstreamConfig", "StrategyConfig",
		"DeploymentStrategyConfig", "CanaryConfig", "BlueGreenConfig",
		"RouteMatchStrategyConfig", "LoadBalancingStrategyConfig", "HealthCheckConfig",
		"RetryStrategyConfig", "RateLimitStrategyConfig",
		"ObservabilityStrategyConfig", "AccessLogsConfig",
		"PolicyTargetRef", "CustomFilter",
		"DeployInstructions", "GatewayInfo", "DockerInstructions", "K8sInstructions",
		"GatewayResponse", "GatewayListResponse", "GatewayPutRequest",
		"ListenerResponse", "ListenerListResponse", "ListenerPutRequest",
		"APIResponse", "APIListResponse", "APIPutRequest",
		"DeploymentResponse", "DeploymentListResponse", "DeploymentPutRequest",
		"GatewayPolicyResponse", "GatewayPolicyListResponse", "GatewayPolicyPutRequest",
		"APIPolicyResponse", "APIPolicyListResponse", "APIPolicyPutRequest",
		"BackendPolicyResponse", "BackendPolicyListResponse", "BackendPolicyPutRequest",
		"HealthResponse", "DeleteResponse",
	}
	for _, s := range expected {
		if _, ok := doc.Components.Schemas[s]; !ok {
			t.Errorf("missing schema: %s", s)
		}
	}
}

func TestGenerate_JSONMarshalSucceeds(t *testing.T) {
	doc := Generate()
	data, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	// Verify it's valid JSON
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}
