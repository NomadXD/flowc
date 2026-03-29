package profile

import (
	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/pkg/types"
)

// edgeProfile returns the built-in edge proxy profile.
// Edge proxies are public-facing and emphasize TLS termination, rate limiting,
// CORS, and JWT validation.
func edgeProfile() *resource.GatewayProfileResource {
	return &resource.GatewayProfileResource{
		Meta: resource.ResourceMeta{
			Kind:    resource.KindGatewayProfile,
			Name:    "edge",
					},
		Spec: resource.GatewayProfileSpec{
			DisplayName: "Edge Proxy",
			Description: "Public-facing API gateway with TLS termination, rate limiting, CORS, and JWT validation.",
			ProfileType: "edge",
			DefaultStrategy: &types.StrategyConfig{
				Deployment:    &types.DeploymentStrategyConfig{Type: "basic"},
				RouteMatching: &types.RouteMatchStrategyConfig{Type: "prefix", CaseSensitive: true},
				LoadBalancing: &types.LoadBalancingStrategyConfig{Type: "round-robin"},
				Retry:         &types.RetryStrategyConfig{Type: "conservative"},
				RateLimit:     &types.RateLimitStrategyConfig{Type: "per-ip", RequestsPerMinute: 1000, BurstSize: 100},
				Observability: &types.ObservabilityStrategyConfig{
					AccessLogs: &types.AccessLogsConfig{Enabled: true, Format: "json"},
				},
			},
			ListenerPresets: []resource.ListenerPreset{
				{Name: "https", Port: 443, TLS: &resource.TLSConfig{MinVersion: "TLSv1.2"}},
				{Name: "http", Port: 80},
			},
			Bootstrap: &resource.BootstrapConfig{
				AdminPort:      9901,
				XDSClusterName: "flowc_xds_cluster",
			},
			EnvoyImage: "envoyproxy/envoy:v1.31-latest",
		},
	}
}

// mediationProfile returns the built-in mediation proxy profile.
// Mediation proxies handle protocol translation (REST<->gRPC, REST<->SOAP),
// message transformation, and request/response mapping.
func mediationProfile() *resource.GatewayProfileResource {
	return &resource.GatewayProfileResource{
		Meta: resource.ResourceMeta{
			Kind:    resource.KindGatewayProfile,
			Name:    "mediation",
					},
		Spec: resource.GatewayProfileSpec{
			DisplayName: "Mediation Proxy",
			Description: "Protocol translation proxy for REST-gRPC, REST-SOAP transcoding, message transformation, and request/response mapping.",
			ProfileType: "mediation",
			DefaultStrategy: &types.StrategyConfig{
				Deployment:    &types.DeploymentStrategyConfig{Type: "basic"},
				RouteMatching: &types.RouteMatchStrategyConfig{Type: "prefix", CaseSensitive: true},
				LoadBalancing: &types.LoadBalancingStrategyConfig{Type: "round-robin"},
				Retry:         &types.RetryStrategyConfig{Type: "aggressive"},
				RateLimit:     &types.RateLimitStrategyConfig{Type: "none"},
				Observability: &types.ObservabilityStrategyConfig{
					AccessLogs: &types.AccessLogsConfig{Enabled: true, Format: "json"},
				},
			},
			ListenerPresets: []resource.ListenerPreset{
				{Name: "grpc", Port: 8443, HTTP2: true, TLS: &resource.TLSConfig{MinVersion: "TLSv1.2"}},
				{Name: "http", Port: 8080},
			},
			Bootstrap: &resource.BootstrapConfig{
				AdminPort:      9901,
				XDSClusterName: "flowc_xds_cluster",
			},
			EnvoyImage: "envoyproxy/envoy:v1.31-latest",
		},
	}
}

// sidecarProfile returns the built-in sidecar proxy profile.
// Sidecar proxies run alongside application containers with mTLS,
// circuit breaking, and locality-aware load balancing.
func sidecarProfile() *resource.GatewayProfileResource {
	return &resource.GatewayProfileResource{
		Meta: resource.ResourceMeta{
			Kind:    resource.KindGatewayProfile,
			Name:    "sidecar",
					},
		Spec: resource.GatewayProfileSpec{
			DisplayName: "Sidecar Proxy",
			Description: "Application sidecar with mTLS, circuit breaking, locality-aware load balancing, and minimal policy chain.",
			ProfileType: "sidecar",
			DefaultStrategy: &types.StrategyConfig{
				Deployment:    &types.DeploymentStrategyConfig{Type: "basic"},
				RouteMatching: &types.RouteMatchStrategyConfig{Type: "prefix", CaseSensitive: true},
				LoadBalancing: &types.LoadBalancingStrategyConfig{Type: "locality-aware"},
				Retry:         &types.RetryStrategyConfig{Type: "conservative"},
				RateLimit:     &types.RateLimitStrategyConfig{Type: "none"},
			},
			ListenerPresets: []resource.ListenerPreset{
				{Name: "inbound", Port: 15006, TLS: &resource.TLSConfig{RequireClientCert: true, MinVersion: "TLSv1.3"}},
				{Name: "outbound", Port: 15001},
			},
			Bootstrap: &resource.BootstrapConfig{
				AdminPort:      15000,
				XDSClusterName: "flowc_xds_cluster",
			},
			EnvoyImage: "envoyproxy/envoy:v1.31-latest",
		},
	}
}

// egressProfile returns the built-in egress proxy profile.
// Egress proxies control outbound traffic from the mesh.
func egressProfile() *resource.GatewayProfileResource {
	return &resource.GatewayProfileResource{
		Meta: resource.ResourceMeta{
			Kind:    resource.KindGatewayProfile,
			Name:    "egress",
					},
		Spec: resource.GatewayProfileSpec{
			DisplayName: "Egress Proxy",
			Description: "Outbound traffic control proxy for external service access, authorization, and auditing.",
			ProfileType: "egress",
			DefaultStrategy: &types.StrategyConfig{
				Deployment:    &types.DeploymentStrategyConfig{Type: "basic"},
				RouteMatching: &types.RouteMatchStrategyConfig{Type: "prefix", CaseSensitive: true},
				LoadBalancing: &types.LoadBalancingStrategyConfig{Type: "round-robin"},
				Retry:         &types.RetryStrategyConfig{Type: "conservative"},
				RateLimit:     &types.RateLimitStrategyConfig{Type: "none"},
				Observability: &types.ObservabilityStrategyConfig{
					AccessLogs: &types.AccessLogsConfig{Enabled: true, Format: "json"},
				},
			},
			ListenerPresets: []resource.ListenerPreset{
				{Name: "egress-https", Port: 3128, TLS: &resource.TLSConfig{MinVersion: "TLSv1.2"}},
				{Name: "egress-http", Port: 3129},
			},
			Bootstrap: &resource.BootstrapConfig{
				AdminPort:      9901,
				XDSClusterName: "flowc_xds_cluster",
			},
			EnvoyImage: "envoyproxy/envoy:v1.31-latest",
		},
	}
}

// aiProfile returns the built-in AI proxy profile.
// AI proxies handle AI/LLM traffic with token-based rate limiting,
// model routing, and observability for AI workloads.
func aiProfile() *resource.GatewayProfileResource {
	return &resource.GatewayProfileResource{
		Meta: resource.ResourceMeta{
			Kind:    resource.KindGatewayProfile,
			Name:    "ai",
					},
		Spec: resource.GatewayProfileSpec{
			DisplayName: "AI Proxy",
			Description: "AI/LLM gateway with token-based rate limiting, model routing, and AI workload observability.",
			ProfileType: "ai",
			DefaultStrategy: &types.StrategyConfig{
				Deployment:    &types.DeploymentStrategyConfig{Type: "basic"},
				RouteMatching: &types.RouteMatchStrategyConfig{Type: "prefix", CaseSensitive: true},
				LoadBalancing: &types.LoadBalancingStrategyConfig{Type: "round-robin"},
				Retry:         &types.RetryStrategyConfig{Type: "none"},
				RateLimit:     &types.RateLimitStrategyConfig{Type: "per-user", RequestsPerMinute: 60},
				Observability: &types.ObservabilityStrategyConfig{
					AccessLogs: &types.AccessLogsConfig{Enabled: true, Format: "json"},
					Metrics:    &types.MetricsConfig{Enabled: true},
				},
			},
			ListenerPresets: []resource.ListenerPreset{
				{Name: "https", Port: 443, TLS: &resource.TLSConfig{MinVersion: "TLSv1.2"}},
				{Name: "http", Port: 8080},
			},
			Bootstrap: &resource.BootstrapConfig{
				AdminPort:      9901,
				XDSClusterName: "flowc_xds_cluster",
			},
			EnvoyImage: "envoyproxy/envoy:v1.31-latest",
		},
	}
}
