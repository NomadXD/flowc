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

// APIPolicySpec defines the desired state of APIPolicy.
type APIPolicySpec struct {
	// targetRef identifies the resource this policy targets (API or Deployment).
	// +required
	TargetRef PolicyTargetRef `json:"targetRef"`

	// overrides contains policy configurations that cannot be overridden by more specific policies.
	// +optional
	Overrides *APIPolicyConfig `json:"overrides,omitempty"`

	// defaults contains policy configurations that can be overridden by more specific policies.
	// +optional
	Defaults *APIPolicyConfig `json:"defaults,omitempty"`
}

// APIPolicyConfig holds API-level policy configurations organized by stage.
type APIPolicyConfig struct {
	// entry contains Entry stage policies (CORS).
	// +optional
	Entry *APIEntryConfig `json:"entry,omitempty"`

	// authn contains AuthN stage policies (provider selection, claims extraction).
	// +optional
	AuthN *APIAuthNConfig `json:"authn,omitempty"`

	// authz contains AuthZ stage policies (RBAC).
	// +optional
	AuthZ *APIAuthZConfig `json:"authz,omitempty"`

	// ratelimit contains RateLimit stage policies (per-user rate limiting).
	// +optional
	RateLimit *APIRateLimitConfig `json:"ratelimit,omitempty"`

	// transformation contains Transformation stage policies (header/body rewriting).
	// +optional
	Transformation *TransformationConfig `json:"transformation,omitempty"`

	// customFilters are user-provided filters positioned at hook points.
	// +optional
	CustomFilters []CustomFilter `json:"customFilters,omitempty"`
}

// APIEntryConfig holds API entry-stage configuration.
type APIEntryConfig struct {
	// cors configures Cross-Origin Resource Sharing.
	// +optional
	CORS *CORSConfig `json:"cors,omitempty"`
}

// CORSConfig configures Cross-Origin Resource Sharing.
type CORSConfig struct {
	// allowOrigins is the list of allowed origins.
	// +required
	AllowOrigins []string `json:"allowOrigins"`

	// allowMethods is the list of allowed HTTP methods.
	// +optional
	AllowMethods []string `json:"allowMethods,omitempty"`

	// allowHeaders is the list of allowed request headers.
	// +optional
	AllowHeaders []string `json:"allowHeaders,omitempty"`

	// exposeHeaders is the list of headers exposed to the browser.
	// +optional
	ExposeHeaders []string `json:"exposeHeaders,omitempty"`

	// maxAge is the max cache duration for preflight responses in seconds.
	// +optional
	MaxAge int `json:"maxAge,omitempty"`

	// allowCredentials allows credentials in CORS requests.
	// +optional
	AllowCredentials bool `json:"allowCredentials,omitempty"`
}

// APIAuthNConfig holds API-level authentication configuration.
type APIAuthNConfig struct {
	// providers selects JWT providers registered in a GatewayPolicy.
	// +optional
	Providers []AuthNProviderRef `json:"providers,omitempty"`
}

// AuthNProviderRef references a JWT provider registered in a GatewayPolicy.
type AuthNProviderRef struct {
	// ref is the name of the JWT provider registered in a GatewayPolicy.
	// +required
	Ref string `json:"ref"`

	// audiences is the list of required JWT audiences.
	// +optional
	Audiences []string `json:"audiences,omitempty"`

	// claimsToHeaders extracts JWT claims into request headers.
	// +optional
	ClaimsToHeaders []ClaimToHeader `json:"claimsToHeaders,omitempty"`
}

// ClaimToHeader maps a JWT claim to a request header.
type ClaimToHeader struct {
	// claim is the JWT claim name.
	// +required
	Claim string `json:"claim"`

	// header is the request header to set.
	// +required
	Header string `json:"header"`
}

// APIAuthZConfig holds API-level authorization configuration.
type APIAuthZConfig struct {
	// rbac configures role-based access control.
	// +optional
	RBAC *RBACConfig `json:"rbac,omitempty"`
}

// RBACConfig configures role-based access control.
type RBACConfig struct {
	// rules is the list of RBAC rules.
	// +required
	Rules []RBACRule `json:"rules"`
}

// RBACRule defines a single RBAC rule.
type RBACRule struct {
	// name is a human-readable name for this rule.
	// +optional
	Name string `json:"name,omitempty"`

	// match defines which requests this rule applies to.
	// +required
	Match RBACMatch `json:"match"`

	// requires defines what is needed to satisfy this rule.
	// +required
	Requires RBACRequires `json:"requires"`
}

// RBACMatch defines request matching criteria for an RBAC rule.
type RBACMatch struct {
	// paths is the list of path patterns to match.
	// +optional
	Paths []string `json:"paths,omitempty"`

	// methods is the list of HTTP methods to match.
	// +optional
	Methods []string `json:"methods,omitempty"`
}

// RBACRequires defines what is needed to satisfy an RBAC rule.
type RBACRequires struct {
	// headers are required header values (header name -> required value pattern).
	// +optional
	Headers map[string]string `json:"headers,omitempty"`
}

// APIRateLimitConfig holds API-level rate limiting configuration.
type APIRateLimitConfig struct {
	// perUser configures per-user rate limiting (requires identity from AuthN stage).
	// +optional
	PerUser *PerUserRateLimitConfig `json:"perUser,omitempty"`
}

// PerUserRateLimitConfig configures per-user rate limiting.
type PerUserRateLimitConfig struct {
	// identityHeader is the header containing the user identity (e.g., "X-User-Id").
	// +required
	IdentityHeader string `json:"identityHeader"`

	// requests is the number of allowed requests per window.
	// +required
	Requests uint32 `json:"requests"`

	// window is the rate limit window duration (e.g., "60s").
	// +required
	Window string `json:"window"`

	// burst is the burst allowance.
	// +optional
	Burst uint32 `json:"burst,omitempty"`
}

// TransformationConfig holds request and response transformation configuration.
type TransformationConfig struct {
	// request configures request transformations.
	// +optional
	Request *RequestTransform `json:"request,omitempty"`

	// response configures response transformations.
	// +optional
	Response *ResponseTransform `json:"response,omitempty"`
}

// RequestTransform configures request header transformations.
type RequestTransform struct {
	// setHeaders sets or overwrites request headers.
	// +optional
	SetHeaders map[string]string `json:"setHeaders,omitempty"`

	// addHeaders adds request headers (does not overwrite existing).
	// +optional
	AddHeaders map[string]string `json:"addHeaders,omitempty"`

	// removeHeaders removes request headers by name.
	// +optional
	RemoveHeaders []string `json:"removeHeaders,omitempty"`
}

// ResponseTransform configures response header transformations.
type ResponseTransform struct {
	// setHeaders sets or overwrites response headers.
	// +optional
	SetHeaders map[string]string `json:"setHeaders,omitempty"`

	// removeHeaders removes response headers by name.
	// +optional
	RemoveHeaders []string `json:"removeHeaders,omitempty"`
}

// APIPolicyStatus defines the observed state of APIPolicy.
type APIPolicyStatus struct {
	// conditions represent the current state of the APIPolicy.
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

// APIPolicy is the Schema for the apipolicies API
type APIPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of APIPolicy
	// +required
	Spec APIPolicySpec `json:"spec"`

	// status defines the observed state of APIPolicy
	// +optional
	Status APIPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// APIPolicyList contains a list of APIPolicy
type APIPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []APIPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIPolicy{}, &APIPolicyList{})
}
