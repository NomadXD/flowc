package config

import "github.com/flowc-labs/flowc/pkg/types"

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			APIPort:          8080,
			XDSPort:          18000,
			ReadTimeout:      "30s",
			WriteTimeout:     "30s",
			IdleTimeout:      "60s",
			GracefulShutdown: true,
			ShutdownTimeout:  "10s",
		},
		XDS: XDSConfig{
			DefaultListenerPort: 9095,
			DefaultNodeID:       "test-envoy-node",
			SnapshotCache: SnapshotCacheConfig{
				ADS: true,
			},
			GRPC: GRPCConfig{
				KeepaliveTime:                "30s",
				KeepaliveTimeout:             "5s",
				KeepaliveMinTime:             "5s",
				KeepalivePermitWithoutStream: true,
			},
		},
		DefaultStrategy: &types.StrategyConfig{
			Deployment: &types.DeploymentStrategyConfig{
				Type: "basic",
			},
			RouteMatching: &types.RouteMatchStrategyConfig{
				Type:          "prefix",
				CaseSensitive: true,
			},
			LoadBalancing: &types.LoadBalancingStrategyConfig{
				Type:        "round-robin",
				ChoiceCount: 2,
			},
			Retry: &types.RetryStrategyConfig{
				Type:          "conservative",
				MaxRetries:    1,
				RetryOn:       "5xx,reset",
				PerTryTimeout: "5s",
			},
			RateLimit: &types.RateLimitStrategyConfig{
				Type: "none",
			},
			Observability: &types.ObservabilityStrategyConfig{
				Tracing: &types.TracingConfig{
					Enabled:      false,
					SamplingRate: 0.01,
				},
				Metrics: &types.MetricsConfig{
					Enabled: false,
				},
				AccessLogs: &types.AccessLogsConfig{
					Enabled: false,
					Format:  "json",
				},
			},
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			Output:           "stdout",
			Structured:       true,
			EnableCaller:     false,
			EnableStacktrace: false,
		},
		Features: FeaturesConfig{
			ExternalTranslators: true,
			OpenAPIValidation:   true,
			Metrics:             false,
			Tracing:             false,
			RateLimiting:        false,
		},
	}
}

// Example returns an example configuration with comments for documentation
func Example() string {
	return `# FlowC Control Plane Configuration
# This is the global configuration file for the FlowC control plane

# Server configuration
server:
  # Port for the REST API server
  api_port: 8080
  
  # Port for the XDS gRPC server
  xds_port: 18000
  
  # HTTP server timeouts
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"
  
  # Graceful shutdown settings
  graceful_shutdown: true
  shutdown_timeout: "10s"

# XDS server configuration
xds:
  # Default listener port for Envoy proxies
  default_listener_port: 9095
  
  # Default node ID for testing/development
  default_node_id: "test-envoy-node"
  
  # Snapshot cache settings
  snapshot_cache:
    # Enable Aggregated Discovery Service
    ads: true
  
  # gRPC server settings
  grpc:
    # Keepalive time (time between keepalive pings)
    keepalive_time: "30s"
    
    # Keepalive timeout (time to wait for keepalive response)
    keepalive_timeout: "5s"
    
    # Minimum time between keepalive pings
    keepalive_min_time: "5s"
    
    # Allow keepalive pings without active streams
    keepalive_permit_without_stream: true

# Default strategy configurations (OPTIONAL)
# 
# IMPORTANT: Strategies are deployment-specific configuration!
# Each deployment should define its own strategies in flowc.yaml
# 
# Only uncomment this section if you want to provide organization-wide
# fallback defaults for deployments that don't specify strategies.
# 
# Most users should leave this commented out and define strategies
# per-deployment in their flowc.yaml files.
#
# defaults:
#   strategies:
#     deployment:
#       type: "basic"
#     route_matching:
#       type: "prefix"
#       case_sensitive: true
#     load_balancing:
#       type: "round-robin"
#       choice_count: 2
#     retry:
#       type: "conservative"
#       max_retries: 1
#       retry_on: "5xx,reset"
#       per_try_timeout: "5s"
#     rate_limiting:
#       type: "none"
#     observability:
#       tracing:
#         enabled: false
#       metrics:
#         enabled: false
#       access_logs:
#         enabled: false

# Logging configuration
logging:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log format: json, text
  format: "json"
  
  # Output: stdout, stderr, or file path
  output: "stdout"
  
  # Enable structured logging
  structured: true
  
  # Enable caller information in logs
  enable_caller: false
  
  # Enable stack traces for errors
  enable_stacktrace: false

# Feature flags
features:
  # Enable external translator support
  external_translators: true
  
  # Enable OpenAPI validation
  openapi_validation: true
  
  # Enable metrics collection
  metrics: false
  
  # Enable distributed tracing
  tracing: false
  
  # Enable rate limiting
  rate_limiting: false
`
}
