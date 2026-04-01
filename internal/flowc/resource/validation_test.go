package resource

import (
	"testing"

	"github.com/flowc-labs/flowc/pkg/types"
)

func TestValidateName(t *testing.T) {
	valid := []string{"a", "abc", "my-gateway", "gw-123", "a1"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) unexpected error: %v", name, err)
		}
	}

	invalid := []string{"", "A", "ABC", "-abc", "abc-", "my_gateway", "a b", "123abc",
		// 64 chars (too long)
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) expected error, got nil", name)
		}
	}
}

func TestGatewayResource_Validate(t *testing.T) {
	valid := &GatewayResource{
		Meta: ResourceMeta{Name: "my-gw"},
		Spec: GatewaySpec{NodeID: "envoy-node-1"},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Missing nodeId
	noNode := &GatewayResource{
		Meta: ResourceMeta{Name: "my-gw"},
		Spec: GatewaySpec{},
	}
	if err := noNode.Validate(); err == nil {
		t.Error("expected error for missing nodeId")
	}

	// Invalid name
	badName := &GatewayResource{
		Meta: ResourceMeta{Name: "My_GW"},
		Spec: GatewaySpec{NodeID: "node-1"},
	}
	if err := badName.Validate(); err == nil {
		t.Error("expected error for invalid name")
	}
}

func TestListenerResource_Validate(t *testing.T) {
	valid := &ListenerResource{
		Meta: ResourceMeta{Name: "http-listener"},
		Spec: ListenerSpec{GatewayRef: "my-gw", Port: 8080},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Missing port
	noPort := &ListenerResource{
		Meta: ResourceMeta{Name: "http-listener"},
		Spec: ListenerSpec{GatewayRef: "my-gw"},
	}
	if err := noPort.Validate(); err == nil {
		t.Error("expected error for missing port")
	}

	// Missing gatewayRef
	noGw := &ListenerResource{
		Meta: ResourceMeta{Name: "http-listener"},
		Spec: ListenerSpec{Port: 8080},
	}
	if err := noGw.Validate(); err == nil {
		t.Error("expected error for missing gatewayRef")
	}
}

func TestVirtualHostResource_Validate(t *testing.T) {
	valid := &VirtualHostResource{
		Meta: ResourceMeta{Name: "production"},
		Spec: VirtualHostSpec{
			GatewayRef:  "my-gw",
			ListenerRef: "http-listener",
			Hostname:    "api.example.com",
		},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Missing hostname
	noHost := &VirtualHostResource{
		Meta: ResourceMeta{Name: "production"},
		Spec: VirtualHostSpec{GatewayRef: "my-gw", ListenerRef: "http-listener"},
	}
	if err := noHost.Validate(); err == nil {
		t.Error("expected error for missing hostname")
	}
}

func TestAPIResource_Validate(t *testing.T) {
	valid := &APIResource{
		Meta: ResourceMeta{Name: "petstore"},
		Spec: APISpec{
			Version: "1.0.0",
			Context: "/petstore",
			Upstream: types.UpstreamConfig{Host: "backend", Port: 8080},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Missing context
	noCtx := &APIResource{
		Meta: ResourceMeta{Name: "petstore"},
		Spec: APISpec{
			Version:  "1.0.0",
			Upstream: types.UpstreamConfig{Host: "backend", Port: 8080},
		},
	}
	if err := noCtx.Validate(); err == nil {
		t.Error("expected error for missing context")
	}

	// Context without leading slash
	badCtx := &APIResource{
		Meta: ResourceMeta{Name: "petstore"},
		Spec: APISpec{
			Version:  "1.0.0",
			Context:  "petstore",
			Upstream: types.UpstreamConfig{Host: "backend", Port: 8080},
		},
	}
	if err := badCtx.Validate(); err == nil {
		t.Error("expected error for context without leading slash")
	}
}

func TestDeploymentResource_Validate(t *testing.T) {
	// Valid with only required fields (gateway.name + apiRef)
	valid := &DeploymentResource{
		Meta: ResourceMeta{Name: "petstore-prod"},
		Spec: DeploymentSpec{
			APIRef:  "petstore",
			Gateway: DeploymentGatewayRef{Name: "my-gw"},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Valid with all fields specified
	validFull := &DeploymentResource{
		Meta: ResourceMeta{Name: "petstore-prod"},
		Spec: DeploymentSpec{
			APIRef: "petstore",
			Gateway: DeploymentGatewayRef{
				Name:        "my-gw",
				Listener:    "http-listener",
				VirtualHost: "production",
			},
		},
	}
	if err := validFull.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Missing apiRef
	noAPI := &DeploymentResource{
		Meta: ResourceMeta{Name: "petstore-prod"},
		Spec: DeploymentSpec{
			Gateway: DeploymentGatewayRef{Name: "my-gw"},
		},
	}
	if err := noAPI.Validate(); err == nil {
		t.Error("expected error for missing apiRef")
	}

	// Missing gateway.name
	noGw := &DeploymentResource{
		Meta: ResourceMeta{Name: "petstore-prod"},
		Spec: DeploymentSpec{
			APIRef: "petstore",
		},
	}
	if err := noGw.Validate(); err == nil {
		t.Error("expected error for missing gateway.name")
	}
}
