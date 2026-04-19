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

// BackendPolicySpec defines the desired state of BackendPolicy.
type BackendPolicySpec struct {
	// targetRef identifies the resource this policy targets (API or Deployment).
	// +required
	TargetRef PolicyTargetRef `json:"targetRef"`

	// overrides contains backend configurations that cannot be overridden.
	// +optional
	Overrides *BackendPolicyConfig `json:"overrides,omitempty"`

	// defaults contains backend configurations that can be overridden.
	// +optional
	Defaults *BackendPolicyConfig `json:"defaults,omitempty"`
}

// BackendPolicyConfig holds backend-level configurations.
type BackendPolicyConfig struct {
	// timeout configures connection and request timeouts.
	// +optional
	Timeout *TimeoutConfig `json:"timeout,omitempty"`

	// retry configures retry behavior.
	// +optional
	Retry *BackendRetryConfig `json:"retry,omitempty"`

	// circuitBreaker configures circuit breaking.
	// +optional
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`

	// healthCheck configures active health checking.
	// +optional
	HealthCheck *BackendHealthCheckConfig `json:"healthCheck,omitempty"`

	// loadBalancing configures the load balancing algorithm.
	// +optional
	LoadBalancing *BackendLoadBalancingConfig `json:"loadBalancing,omitempty"`
}

// TimeoutConfig configures connection and request timeouts.
type TimeoutConfig struct {
	// connectionTimeout is the timeout for establishing a connection (e.g., "5s").
	// +optional
	ConnectionTimeout string `json:"connectionTimeout,omitempty"`

	// requestTimeout is the timeout for a complete request (e.g., "30s").
	// +optional
	RequestTimeout string `json:"requestTimeout,omitempty"`
}

// BackendRetryConfig configures retry behavior for backend connections.
type BackendRetryConfig struct {
	// attempts is the maximum number of retry attempts.
	// +optional
	Attempts uint32 `json:"attempts,omitempty"`

	// retryOn specifies conditions that trigger a retry (e.g., ["5xx", "connect-failure"]).
	// +optional
	RetryOn []string `json:"retryOn,omitempty"`

	// backoff is the base retry backoff duration (e.g., "25ms").
	// +optional
	Backoff string `json:"backoff,omitempty"`
}

// CircuitBreakerConfig configures circuit breaking for backend connections.
type CircuitBreakerConfig struct {
	// maxConnections is the maximum number of connections to the upstream.
	// +optional
	MaxConnections uint32 `json:"maxConnections,omitempty"`

	// maxPendingRequests is the maximum number of pending requests.
	// +optional
	MaxPendingRequests uint32 `json:"maxPendingRequests,omitempty"`

	// maxRetries is the maximum number of concurrent retries.
	// +optional
	MaxRetries uint32 `json:"maxRetries,omitempty"`
}

// BackendHealthCheckConfig configures active health checking for backends.
type BackendHealthCheckConfig struct {
	// interval is the health check interval (e.g., "10s").
	// +optional
	Interval string `json:"interval,omitempty"`

	// timeout is the health check timeout (e.g., "2s").
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// healthyThreshold is the number of consecutive successes to mark healthy.
	// +optional
	HealthyThreshold uint32 `json:"healthyThreshold,omitempty"`

	// unhealthyThreshold is the number of consecutive failures to mark unhealthy.
	// +optional
	UnhealthyThreshold uint32 `json:"unhealthyThreshold,omitempty"`

	// path is the HTTP path for health checks.
	// +optional
	Path string `json:"path,omitempty"`
}

// BackendLoadBalancingConfig configures the load balancing algorithm.
type BackendLoadBalancingConfig struct {
	// type is the load balancing algorithm.
	// +optional
	// +kubebuilder:validation:Enum=round-robin;least-request;random;consistent-hash
	Type string `json:"type,omitempty"`
}

// BackendPolicyStatus defines the observed state of BackendPolicy.
type BackendPolicyStatus struct {
	// conditions represent the current state of the BackendPolicy.
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

// BackendPolicy is the Schema for the backendpolicies API
type BackendPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of BackendPolicy
	// +required
	Spec BackendPolicySpec `json:"spec"`

	// status defines the observed state of BackendPolicy
	// +optional
	Status BackendPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BackendPolicyList contains a list of BackendPolicy
type BackendPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []BackendPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackendPolicy{}, &BackendPolicyList{})
}
