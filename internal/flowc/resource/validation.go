package resource

import (
	"fmt"
	"regexp"
	"strings"
)

// nameRegex validates resource names: lowercase alphanumeric with hyphens, 1-63 chars.
var nameRegex = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateName checks if a name is valid for a resource.
func ValidateName(name string) error {
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("invalid name %q: must match %s", name, nameRegex.String())
	}
	return nil
}

// ValidateMeta validates the common metadata fields.
func ValidateMeta(m *ResourceMeta) error {
	if !IsValidKind(m.Kind) {
		return fmt.Errorf("invalid kind %q", m.Kind)
	}
	if err := ValidateName(m.Name); err != nil {
		return fmt.Errorf("metadata.name: %w", err)
	}
	return nil
}

// Validate checks if a GatewayResource is valid.
func (r *GatewayResource) Validate() error {
	r.Meta.Kind = KindGateway
	if err := ValidateMeta(&r.Meta); err != nil {
		return err
	}
	if r.Spec.NodeID == "" {
		return fmt.Errorf("spec.nodeId is required")
	}
	return nil
}

// Validate checks if a ListenerResource is valid.
func (r *ListenerResource) Validate() error {
	r.Meta.Kind = KindListener
	if err := ValidateMeta(&r.Meta); err != nil {
		return err
	}
	if r.Spec.GatewayRef == "" {
		return fmt.Errorf("spec.gatewayRef is required")
	}
	if r.Spec.Port == 0 {
		return fmt.Errorf("spec.port is required")
	}
	if r.Spec.Port > 65535 {
		return fmt.Errorf("spec.port must be <= 65535")
	}
	return nil
}

// Validate checks if a VirtualHostResource is valid.
func (r *VirtualHostResource) Validate() error {
	r.Meta.Kind = KindVirtualHost
	if err := ValidateMeta(&r.Meta); err != nil {
		return err
	}
	if r.Spec.GatewayRef == "" {
		return fmt.Errorf("spec.gatewayRef is required")
	}
	if r.Spec.ListenerRef == "" {
		return fmt.Errorf("spec.listenerRef is required")
	}
	if r.Spec.Hostname == "" {
		return fmt.Errorf("spec.hostname is required")
	}
	return nil
}

// Validate checks if an APIResource is valid.
func (r *APIResource) Validate() error {
	r.Meta.Kind = KindAPI
	if err := ValidateMeta(&r.Meta); err != nil {
		return err
	}
	if r.Spec.Version == "" {
		return fmt.Errorf("spec.version is required")
	}
	if r.Spec.Context == "" {
		return fmt.Errorf("spec.context is required")
	}
	if !strings.HasPrefix(r.Spec.Context, "/") {
		return fmt.Errorf("spec.context must start with /")
	}
	if r.Spec.Upstream.Host == "" {
		return fmt.Errorf("spec.upstream.host is required")
	}
	if r.Spec.Upstream.Port == 0 {
		return fmt.Errorf("spec.upstream.port is required")
	}
	return nil
}

// validProfileTypes are the allowed values for GatewayProfileSpec.ProfileType.
var validProfileTypes = map[string]bool{
	"edge": true, "mediation": true, "sidecar": true,
	"egress": true, "ai": true, "custom": true,
}

// Validate checks if a GatewayProfileResource is valid.
func (r *GatewayProfileResource) Validate() error {
	r.Meta.Kind = KindGatewayProfile
	if err := ValidateMeta(&r.Meta); err != nil {
		return err
	}
	if r.Spec.DisplayName == "" {
		return fmt.Errorf("spec.displayName is required")
	}
	if r.Spec.ProfileType == "" {
		return fmt.Errorf("spec.profileType is required")
	}
	if !validProfileTypes[r.Spec.ProfileType] {
		return fmt.Errorf("spec.profileType %q is not valid (must be one of: edge, mediation, sidecar, egress, ai, custom)", r.Spec.ProfileType)
	}
	for i, lp := range r.Spec.ListenerPresets {
		if lp.Port == 0 || lp.Port > 65535 {
			return fmt.Errorf("spec.listenerPresets[%d].port must be 1-65535", i)
		}
	}
	return nil
}

// Validate checks if a DeploymentResource is valid.
func (r *DeploymentResource) Validate() error {
	r.Meta.Kind = KindDeployment
	if err := ValidateMeta(&r.Meta); err != nil {
		return err
	}
	if r.Spec.APIRef == "" {
		return fmt.Errorf("spec.apiRef is required")
	}
	if r.Spec.Gateway.Name == "" {
		return fmt.Errorf("spec.gateway.name is required")
	}
	return nil
}
