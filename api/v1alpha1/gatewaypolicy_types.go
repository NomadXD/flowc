/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayPolicySpec defines the desired state of GatewayPolicy.
type GatewayPolicySpec struct {
	// targetRef identifies the resource this policy targets (Gateway, Listener, or VirtualHost).
	// +required
	TargetRef PolicyTargetRef `json:"targetRef"`

	// overrides contains policy configurations that cannot be overridden by more specific policies.
	// +optional
	Overrides *GatewayPolicyConfig `json:"overrides,omitempty"`

	// defaults contains policy configurations that can be overridden by more specific policies.
	// +optional
	Defaults *GatewayPolicyConfig `json:"defaults,omitempty"`
}

// GatewayPolicyConfig holds gateway-level policy configurations organized by stage.
type GatewayPolicyConfig struct {
	// entry contains Entry stage policies (IP filter, request size limits).
	// +optional
	Entry *EntryPolicyConfig `json:"entry,omitempty"`

	// authn contains AuthN stage policies (JWT provider registration).
	// +optional
	AuthN *GatewayAuthNConfig `json:"authn,omitempty"`

	// ratelimit contains RateLimit stage policies (IP-based rate limiting).
	// +optional
	RateLimit *GatewayRateLimitConfig `json:"ratelimit,omitempty"`

	// observability contains Observability policies (access logging, metrics).
	// +optional
	Observability *ObservabilityConfig `json:"observability,omitempty"`

	// requireAuthN enforces that all APIs on the target must have authentication configured.
	// +optional
	RequireAuthN bool `json:"requireAuthn,omitempty"`

	// allowedHooks restricts which before/during/after hook points developers can use.
	// If empty, all hooks are allowed.
	// +optional
	AllowedHooks []string `json:"allowedHooks,omitempty"`
}

// EntryPolicyConfig holds entry-stage policy configurations.
type EntryPolicyConfig struct {
	// ipFilter configures IP-based access control.
	// +optional
	IPFilter *IPFilterConfig `json:"ipFilter,omitempty"`

	// requestSizeLimit sets the maximum request body size.
	// +optional
	RequestSizeLimit *RequestSizeLimitConfig `json:"requestSizeLimit,omitempty"`
}

// IPFilterConfig configures IP-based access control.
type IPFilterConfig struct {
	// action is the filter action: allow or deny.
	// +required
	// +kubebuilder:validation:Enum=allow;deny
	Action string `json:"action"`

	// cidrs is the list of CIDR ranges to match.
	// +required
	CIDRs []string `json:"cidrs"`
}

// RequestSizeLimitConfig configures request body size limits.
type RequestSizeLimitConfig struct {
	// maxBodySize is the maximum request body size (e.g., "5MB", "1GB").
	// +required
	MaxBodySize string `json:"maxBodySize"`
}

// GatewayAuthNConfig holds gateway-level authentication configuration.
type GatewayAuthNConfig struct {
	// jwtProviders registers JWT identity providers available to APIs on this gateway.
	// +optional
	JWTProviders []JWTProvider `json:"jwtProviders,omitempty"`
}

// JWTProvider defines a JWT identity provider registration.
type JWTProvider struct {
	// name is a unique identifier for this provider (referenced by APIPolicies).
	// +required
	Name string `json:"name"`

	// issuer is the JWT issuer URL.
	// +required
	Issuer string `json:"issuer"`

	// jwksUri is the URL for the JWKS endpoint.
	// +required
	JWKSUri string `json:"jwksUri"`

	// jwksCacheDuration is the cache duration for JWKS (e.g., "300s").
	// +optional
	// +kubebuilder:default="300s"
	JWKSCacheDuration string `json:"jwksCacheDuration,omitempty"`
}

// GatewayRateLimitConfig holds gateway-level rate limiting configuration.
type GatewayRateLimitConfig struct {
	// ipRateLimit configures IP-based rate limiting.
	// +optional
	IPRateLimit *IPRateLimitConfig `json:"ipRateLimit,omitempty"`
}

// IPRateLimitConfig configures IP-based rate limiting.
type IPRateLimitConfig struct {
	// requests is the number of allowed requests per window.
	// +required
	Requests uint32 `json:"requests"`

	// window is the rate limit window duration (e.g., "60s", "1m").
	// +required
	Window string `json:"window"`

	// burst is the burst allowance above the rate limit.
	// +optional
	Burst uint32 `json:"burst,omitempty"`
}

// ObservabilityConfig holds observability policy configuration.
type ObservabilityConfig struct {
	// accessLog configures access logging.
	// +optional
	AccessLog *AccessLogsConfig `json:"accessLog,omitempty"`
}

// GatewayPolicyStatus defines the observed state of GatewayPolicy.
type GatewayPolicyStatus struct {
	// conditions represent the current state of the GatewayPolicy.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Target Kind",type=string,JSONPath=`.spec.targetRef.kind`
// +kubebuilder:printcolumn:name="Target Name",type=string,JSONPath=`.spec.targetRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GatewayPolicy is the Schema for the gatewaypolicies API
type GatewayPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of GatewayPolicy
	// +required
	Spec GatewayPolicySpec `json:"spec"`

	// status defines the observed state of GatewayPolicy
	// +optional
	Status GatewayPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// GatewayPolicyList contains a list of GatewayPolicy
type GatewayPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []GatewayPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GatewayPolicy{}, &GatewayPolicyList{})
}
