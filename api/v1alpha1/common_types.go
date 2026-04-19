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

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// TLSConfig contains TLS settings for a listener.
type TLSConfig struct {
	// certPath is the path to the TLS certificate file.
	// +required
	CertPath string `json:"certPath"`

	// keyPath is the path to the TLS private key file.
	// +required
	KeyPath string `json:"keyPath"`

	// caPath is the path to the CA certificate for client verification.
	// +optional
	CAPath string `json:"caPath,omitempty"`

	// requireClientCert enables mutual TLS (mTLS).
	// +optional
	RequireClientCert bool `json:"requireClientCert,omitempty"`

	// minVersion is the minimum TLS version (e.g., "1.2", "1.3").
	// +optional
	MinVersion string `json:"minVersion,omitempty"`

	// cipherSuites is the list of allowed cipher suites.
	// +optional
	CipherSuites []string `json:"cipherSuites,omitempty"`
}

// UpstreamConfig defines the backend service connection parameters.
type UpstreamConfig struct {
	// host is the hostname or IP of the upstream service.
	// +required
	Host string `json:"host"`

	// port is the port of the upstream service.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port uint32 `json:"port"`

	// scheme is the protocol scheme (http or https).
	// +optional
	// +kubebuilder:default="http"
	Scheme string `json:"scheme,omitempty"`

	// timeout is the request timeout (e.g., "30s", "5m").
	// +optional
	// +kubebuilder:default="30s"
	Timeout string `json:"timeout,omitempty"`
}

// RoutingConfig defines route matching behavior for an API.
type RoutingConfig struct {
	// matchType is the route matching strategy: prefix, exact, regex, header-versioned.
	// +optional
	// +kubebuilder:default="prefix"
	// +kubebuilder:validation:Enum=prefix;exact;regex;header-versioned
	MatchType string `json:"matchType,omitempty"`

	// caseSensitive enables case-sensitive path matching.
	// +optional
	CaseSensitive bool `json:"caseSensitive,omitempty"`

	// loadBalancing is the load balancing algorithm.
	// +optional
	// +kubebuilder:validation:Enum=round-robin;least-request;random;consistent-hash;locality-aware
	LoadBalancing string `json:"loadBalancing,omitempty"`
}

// PolicyInstance represents an attached policy with its configuration.
type PolicyInstance struct {
	// id is a unique identifier for this policy instance.
	// +required
	ID string `json:"id"`

	// policyType identifies the type of policy (e.g., "rate-limit", "cors", "jwt-auth").
	// +required
	PolicyType string `json:"policyType"`

	// order determines execution order within a stage (lower runs first).
	// +optional
	Order int `json:"order,omitempty"`

	// enabled indicates whether this policy instance is active.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// inheritanceMode controls policy merging across levels: inherit, override, disable, add.
	// +optional
	// +kubebuilder:validation:Enum=inherit;override;disable;add
	InheritanceMode string `json:"inheritanceMode,omitempty"`

	// config holds type-specific policy configuration.
	// +optional
	Config *apiextensionsv1.JSON `json:"config,omitempty"`
}

// StrategyConfig contains all strategy configurations for xDS generation.
type StrategyConfig struct {
	// deployment configures the deployment strategy (basic, canary, blue-green).
	// +optional
	Deployment *DeploymentStrategyConfig `json:"deployment,omitempty"`

	// routeMatching configures how routes are matched.
	// +optional
	RouteMatching *RouteMatchStrategyConfig `json:"routeMatching,omitempty"`

	// loadBalancing configures load balancing behavior.
	// +optional
	LoadBalancing *LoadBalancingStrategyConfig `json:"loadBalancing,omitempty"`

	// retry configures retry behavior.
	// +optional
	Retry *RetryStrategyConfig `json:"retry,omitempty"`

	// rateLimit configures rate limiting.
	// +optional
	RateLimit *RateLimitStrategyConfig `json:"rateLimit,omitempty"`

	// observability configures tracing, metrics, and logging.
	// +optional
	Observability *ObservabilityStrategyConfig `json:"observability,omitempty"`
}

// DeploymentStrategyConfig configures the deployment strategy.
type DeploymentStrategyConfig struct {
	// type is the deployment strategy: basic, canary, blue-green.
	// +required
	// +kubebuilder:validation:Enum=basic;canary;blue-green
	Type string `json:"type"`

	// canary holds canary-specific configuration.
	// +optional
	Canary *CanaryConfig `json:"canary,omitempty"`

	// blueGreen holds blue-green-specific configuration.
	// +optional
	BlueGreen *BlueGreenConfig `json:"blueGreen,omitempty"`
}

// CanaryConfig defines canary deployment settings.
type CanaryConfig struct {
	// canaryWeight is the percentage of traffic routed to the canary (0-100).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	CanaryWeight int `json:"canaryWeight,omitempty"`
}

// BlueGreenConfig defines blue-green deployment settings.
type BlueGreenConfig struct {
	// activeVersion is the currently serving version.
	// +optional
	ActiveVersion string `json:"activeVersion,omitempty"`

	// standbyVersion is the version ready to switch to.
	// +optional
	StandbyVersion string `json:"standbyVersion,omitempty"`
}

// RouteMatchStrategyConfig configures route matching.
type RouteMatchStrategyConfig struct {
	// type is the matching strategy: prefix, exact, regex, header-versioned.
	// +required
	// +kubebuilder:validation:Enum=prefix;exact;regex;header-versioned
	Type string `json:"type"`

	// versionHeader is the header name for header-versioned routing.
	// +optional
	VersionHeader string `json:"versionHeader,omitempty"`

	// caseSensitive enables case-sensitive matching.
	// +optional
	CaseSensitive bool `json:"caseSensitive,omitempty"`
}

// LoadBalancingStrategyConfig configures load balancing.
type LoadBalancingStrategyConfig struct {
	// type is the LB algorithm: round-robin, least-request, random, consistent-hash, locality-aware.
	// +required
	// +kubebuilder:validation:Enum=round-robin;least-request;random;consistent-hash;locality-aware
	Type string `json:"type"`

	// hashOn selects the hash key for consistent-hash: header, cookie, source-ip.
	// +optional
	HashOn string `json:"hashOn,omitempty"`

	// headerName is the header to hash on (when hashOn=header).
	// +optional
	HeaderName string `json:"headerName,omitempty"`

	// healthCheck configures active health checking.
	// +optional
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// HealthCheckConfig configures health checking.
type HealthCheckConfig struct {
	// enabled activates health checking.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// interval is the health check interval (e.g., "10s").
	// +optional
	Interval string `json:"interval,omitempty"`

	// timeout is the health check timeout (e.g., "5s").
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// path is the HTTP path for health checks.
	// +optional
	Path string `json:"path,omitempty"`

	// expectedStatus is the expected HTTP status code.
	// +optional
	ExpectedStatus uint32 `json:"expectedStatus,omitempty"`
}

// RetryStrategyConfig configures retry behavior.
type RetryStrategyConfig struct {
	// type is the retry preset: none, conservative, aggressive, custom.
	// +required
	// +kubebuilder:validation:Enum=none;conservative;aggressive;custom
	Type string `json:"type"`

	// maxRetries is the maximum number of retries.
	// +optional
	MaxRetries uint32 `json:"maxRetries,omitempty"`

	// retryOn specifies which conditions trigger a retry (e.g., "5xx,reset,connect-failure").
	// +optional
	RetryOn string `json:"retryOn,omitempty"`

	// perTryTimeout is the timeout per retry attempt (e.g., "2s").
	// +optional
	PerTryTimeout string `json:"perTryTimeout,omitempty"`
}

// RateLimitStrategyConfig configures rate limiting.
type RateLimitStrategyConfig struct {
	// type is the rate limit scope: none, global, per-ip, per-user.
	// +required
	// +kubebuilder:validation:Enum=none;global;per-ip;per-user
	Type string `json:"type"`

	// requestsPerMinute is the rate limit threshold.
	// +optional
	RequestsPerMinute uint32 `json:"requestsPerMinute,omitempty"`

	// burstSize is the burst allowance above the rate limit.
	// +optional
	BurstSize uint32 `json:"burstSize,omitempty"`
}

// ObservabilityStrategyConfig configures observability.
type ObservabilityStrategyConfig struct {
	// accessLogs configures access logging.
	// +optional
	AccessLogs *AccessLogsConfig `json:"accessLogs,omitempty"`
}

// AccessLogsConfig configures access logging.
type AccessLogsConfig struct {
	// enabled activates access logging.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// format is the log format: json or text.
	// +optional
	// +kubebuilder:validation:Enum=json;text
	Format string `json:"format,omitempty"`

	// path is the log output path (stdout, stderr, or file path).
	// +optional
	Path string `json:"path,omitempty"`
}

// ParsedInfo contains metadata extracted from a parsed API specification.
type ParsedInfo struct {
	// title is the API title from the spec.
	// +optional
	Title string `json:"title,omitempty"`

	// version is the API version from the spec.
	// +optional
	Version string `json:"version,omitempty"`

	// paths lists the API paths/endpoints.
	// +optional
	Paths []string `json:"paths,omitempty"`

	// servers lists the server URLs from the spec.
	// +optional
	Servers []string `json:"servers,omitempty"`
}

// ListenerPreset is a recommended listener configuration for a gateway profile.
type ListenerPreset struct {
	// name is a human-readable name for this preset.
	// +required
	Name string `json:"name"`

	// port is the listener port number.
	// +required
	Port uint32 `json:"port"`

	// tls contains optional TLS configuration.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty"`

	// http2 enables HTTP/2 on the listener.
	// +optional
	HTTP2 bool `json:"http2,omitempty"`

	// address is the bind address (default "0.0.0.0").
	// +optional
	Address string `json:"address,omitempty"`
}

// BootstrapConfig contains Envoy bootstrap template configuration.
type BootstrapConfig struct {
	// adminPort is the Envoy admin interface port.
	// +optional
	AdminPort uint32 `json:"adminPort,omitempty"`

	// xdsClusterName is the cluster name for the control plane connection.
	// +optional
	XDSClusterName string `json:"xdsClusterName,omitempty"`

	// staticResources allows arbitrary static resource configuration.
	// +optional
	StaticResources *apiextensionsv1.JSON `json:"staticResources,omitempty"`
}

// PolicyTargetRef identifies the resource a policy targets (GEP-713 style).
type PolicyTargetRef struct {
	// group is the API group of the target resource.
	// +optional
	// +kubebuilder:default="flowc.io"
	Group string `json:"group,omitempty"`

	// kind is the kind of the target resource (e.g., Gateway, API, Deployment).
	// +required
	Kind string `json:"kind"`

	// name is the name of the target resource.
	// +required
	Name string `json:"name"`

	// sectionName targets a sub-section of the resource (e.g., a specific listener).
	// +optional
	SectionName string `json:"sectionName,omitempty"`
}

// CustomFilter defines a user-provided filter at a hook point.
type CustomFilter struct {
	// name is a unique name for this custom filter.
	// +required
	Name string `json:"name"`

	// stage is the hook predicate (e.g., "before:authn", "during:authz", "after:ratelimit").
	// +required
	Stage string `json:"stage"`

	// weight determines execution order within the same stage+predicate (lower runs first).
	// +optional
	// +kubebuilder:default=100
	Weight int `json:"weight,omitempty"`

	// execution is the backend type: cel, wasm, ext_proc_wasm, ext_proc_plugin.
	// +required
	// +kubebuilder:validation:Enum=cel;wasm;ext_proc_wasm;ext_proc_plugin
	Execution string `json:"execution"`

	// expressions holds CEL expressions (for execution=cel).
	// +optional
	Expressions *CELExpressions `json:"expressions,omitempty"`

	// module is the path to a .wasm or .so file (for wasm/ext_proc_wasm/ext_proc_plugin).
	// +optional
	Module string `json:"module,omitempty"`

	// config is arbitrary configuration passed to the filter.
	// +optional
	Config *apiextensionsv1.JSON `json:"config,omitempty"`
}

// CELExpressions holds CEL expressions for each filter callback.
type CELExpressions struct {
	// onRequestHeaders is the CEL expression evaluated when request headers arrive.
	// +optional
	OnRequestHeaders string `json:"onRequestHeaders,omitempty"`

	// onRequestBody is the CEL expression evaluated when request body arrives.
	// +optional
	OnRequestBody string `json:"onRequestBody,omitempty"`

	// onResponseHeaders is the CEL expression evaluated when response headers arrive.
	// +optional
	OnResponseHeaders string `json:"onResponseHeaders,omitempty"`

	// onResponseBody is the CEL expression evaluated when response body arrives.
	// +optional
	OnResponseBody string `json:"onResponseBody,omitempty"`
}
