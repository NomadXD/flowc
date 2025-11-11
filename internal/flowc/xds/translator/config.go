package translator

// XDSStrategyConfig contains all strategy configurations for xDS generation
// This can be specified at API level (flowc.yaml) or gateway level (gateway-config.yaml)
type XDSStrategyConfig struct {
	// Deployment strategy configuration (cluster-focused)
	Deployment *DeploymentStrategyConfig `yaml:"deployment,omitempty" json:"deployment,omitempty"`

	// Route matching strategy configuration
	RouteMatching *RouteMatchStrategyConfig `yaml:"route_matching,omitempty" json:"route_matching,omitempty"`

	// Load balancing strategy configuration
	LoadBalancing *LoadBalancingStrategyConfig `yaml:"load_balancing,omitempty" json:"load_balancing,omitempty"`

	// Retry strategy configuration
	Retry *RetryStrategyConfig `yaml:"retry,omitempty" json:"retry,omitempty"`

	// Rate limiting strategy configuration
	RateLimit *RateLimitStrategyConfig `yaml:"rate_limiting,omitempty" json:"rate_limiting,omitempty"`

	// Observability strategy configuration
	Observability *ObservabilityStrategyConfig `yaml:"observability,omitempty" json:"observability,omitempty"`
}

// DeploymentStrategyConfig configures the deployment strategy (cluster generation)
type DeploymentStrategyConfig struct {
	// Type: basic, canary, blue-green, external
	Type string `yaml:"type" json:"type"`

	// Canary configuration (if type is "canary")
	Canary *CanaryConfig `yaml:"canary,omitempty" json:"canary,omitempty"`

	// Blue-green configuration (if type is "blue-green")
	BlueGreen *BlueGreenConfig `yaml:"blue_green,omitempty" json:"blue_green,omitempty"`

	// External translator configuration (if type is "external")
	External *ExternalTranslatorConfig `yaml:"external,omitempty" json:"external,omitempty"`
}

// RouteMatchStrategyConfig configures how routes are matched
type RouteMatchStrategyConfig struct {
	// Type: prefix, exact, regex, header-versioned
	Type string `yaml:"type" json:"type"`

	// For header-versioned routing
	VersionHeader string `yaml:"version_header,omitempty" json:"version_header,omitempty"`

	// Case sensitivity for path matching
	CaseSensitive bool `yaml:"case_sensitive,omitempty" json:"case_sensitive,omitempty"`
}

// LoadBalancingStrategyConfig configures load balancing behavior
type LoadBalancingStrategyConfig struct {
	// Type: round-robin, least-request, random, consistent-hash, locality-aware
	Type string `yaml:"type" json:"type"`

	// For consistent-hash
	HashOn     string `yaml:"hash_on,omitempty" json:"hash_on,omitempty"`         // header, cookie, source-ip
	HeaderName string `yaml:"header_name,omitempty" json:"header_name,omitempty"` // if hash_on=header
	CookieName string `yaml:"cookie_name,omitempty" json:"cookie_name,omitempty"` // if hash_on=cookie

	// For least-request
	ChoiceCount uint32 `yaml:"choice_count,omitempty" json:"choice_count,omitempty"` // Number of hosts to consider

	// Health check settings
	HealthCheck *HealthCheckConfig `yaml:"health_check,omitempty" json:"health_check,omitempty"`
}

// HealthCheckConfig configures health checking
type HealthCheckConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Interval       string `yaml:"interval,omitempty" json:"interval,omitempty"`               // e.g., "10s"
	Timeout        string `yaml:"timeout,omitempty" json:"timeout,omitempty"`                 // e.g., "5s"
	HealthyCount   uint32 `yaml:"healthy_count,omitempty" json:"healthy_count,omitempty"`     // Consecutive successes
	UnhealthyCount uint32 `yaml:"unhealthy_count,omitempty" json:"unhealthy_count,omitempty"` // Consecutive failures
	Path           string `yaml:"path,omitempty" json:"path,omitempty"`                       // HTTP path for health check
	ExpectedStatus uint32 `yaml:"expected_status,omitempty" json:"expected_status,omitempty"` // Expected HTTP status
}

// RetryStrategyConfig configures retry behavior
type RetryStrategyConfig struct {
	// Type: none, conservative, aggressive, custom
	Type string `yaml:"type" json:"type"`

	// Retry settings (for custom type or to override presets)
	MaxRetries    uint32 `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	RetryOn       string `yaml:"retry_on,omitempty" json:"retry_on,omitempty"`               // e.g., "5xx,reset,connect-failure"
	PerTryTimeout string `yaml:"per_try_timeout,omitempty" json:"per_try_timeout,omitempty"` // e.g., "2s"

	// Retry priority
	RetriableStatusCodes []uint32 `yaml:"retriable_status_codes,omitempty" json:"retriable_status_codes,omitempty"`

	// Retry budget
	BudgetPercent float64 `yaml:"budget_percent,omitempty" json:"budget_percent,omitempty"` // Max % of requests that can be retried
}

// RateLimitStrategyConfig configures rate limiting
type RateLimitStrategyConfig struct {
	// Type: none, global, per-ip, per-user, custom
	Type string `yaml:"type" json:"type"`

	// Global rate limit
	RequestsPerMinute uint32 `yaml:"requests_per_minute,omitempty" json:"requests_per_minute,omitempty"`
	BurstSize         uint32 `yaml:"burst_size,omitempty" json:"burst_size,omitempty"`

	// Per-user rate limiting
	IdentifyBy string `yaml:"identify_by,omitempty" json:"identify_by,omitempty"` // jwt_claim, header, cookie
	ClaimName  string `yaml:"claim_name,omitempty" json:"claim_name,omitempty"`   // JWT claim name
	HeaderName string `yaml:"header_name,omitempty" json:"header_name,omitempty"` // Header name
	CookieName string `yaml:"cookie_name,omitempty" json:"cookie_name,omitempty"` // Cookie name

	// External rate limit service
	ExternalService string `yaml:"external_service,omitempty" json:"external_service,omitempty"` // gRPC endpoint
}

// ObservabilityStrategyConfig configures tracing, metrics, and logging
type ObservabilityStrategyConfig struct {
	// Tracing configuration
	Tracing *TracingConfig `yaml:"tracing,omitempty" json:"tracing,omitempty"`

	// Metrics configuration
	Metrics *MetricsConfig `yaml:"metrics,omitempty" json:"metrics,omitempty"`

	// Access logging configuration
	AccessLogs *AccessLogsConfig `yaml:"access_logs,omitempty" json:"access_logs,omitempty"`
}

// TracingConfig configures distributed tracing
type TracingConfig struct {
	Enabled      bool    `yaml:"enabled" json:"enabled"`
	Provider     string  `yaml:"provider,omitempty" json:"provider,omitempty"`           // zipkin, jaeger, datadog, opentelemetry
	SamplingRate float64 `yaml:"sampling_rate,omitempty" json:"sampling_rate,omitempty"` // 0.0 to 1.0
	Endpoint     string  `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`           // Collector endpoint
}

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"` // prometheus, statsd
	Path     string `yaml:"path,omitempty" json:"path,omitempty"`         // Metrics endpoint path (e.g., /metrics)
	Port     uint32 `yaml:"port,omitempty" json:"port,omitempty"`         // Metrics port
}

// AccessLogsConfig configures access logging
type AccessLogsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Format  string `yaml:"format,omitempty" json:"format,omitempty"` // json, text
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`     // Log file path or stdout/stderr
}

// DefaultXDSStrategyConfig returns the built-in default configuration
func DefaultXDSStrategyConfig() *XDSStrategyConfig {
	return &XDSStrategyConfig{
		Deployment: &DeploymentStrategyConfig{
			Type: "basic",
		},
		RouteMatching: &RouteMatchStrategyConfig{
			Type:          "prefix",
			CaseSensitive: true,
		},
		LoadBalancing: &LoadBalancingStrategyConfig{
			Type:        "round-robin",
			ChoiceCount: 2,
		},
		Retry: &RetryStrategyConfig{
			Type:          "conservative",
			MaxRetries:    1,
			RetryOn:       "5xx,reset",
			PerTryTimeout: "5s",
		},
		RateLimit: &RateLimitStrategyConfig{
			Type:              "none",
			RequestsPerMinute: 0,
		},
		Observability: &ObservabilityStrategyConfig{
			Tracing: &TracingConfig{
				Enabled:      false,
				SamplingRate: 0.01,
			},
			Metrics: &MetricsConfig{
				Enabled: false,
			},
			AccessLogs: &AccessLogsConfig{
				Enabled: false,
				Format:  "json",
			},
		},
	}
}

// Merge merges config with defaults, preferring non-nil values from config
func (c *XDSStrategyConfig) Merge(defaults *XDSStrategyConfig) *XDSStrategyConfig {
	if c == nil {
		return defaults
	}

	merged := &XDSStrategyConfig{}

	// Merge each strategy config
	if c.Deployment != nil {
		merged.Deployment = c.Deployment
	} else {
		merged.Deployment = defaults.Deployment
	}

	if c.RouteMatching != nil {
		merged.RouteMatching = c.RouteMatching
	} else {
		merged.RouteMatching = defaults.RouteMatching
	}

	if c.LoadBalancing != nil {
		merged.LoadBalancing = c.LoadBalancing
	} else {
		merged.LoadBalancing = defaults.LoadBalancing
	}

	if c.Retry != nil {
		merged.Retry = c.Retry
	} else {
		merged.Retry = defaults.Retry
	}

	if c.RateLimit != nil {
		merged.RateLimit = c.RateLimit
	} else {
		merged.RateLimit = defaults.RateLimit
	}

	if c.Observability != nil {
		merged.Observability = c.Observability
	} else {
		merged.Observability = defaults.Observability
	}

	return merged
}
