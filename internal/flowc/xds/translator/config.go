package translator

import "github.com/flowc-labs/flowc/pkg/types"

// DefaultStrategyConfig returns the built-in default configuration
func DefaultStrategyConfig() *types.StrategyConfig {
	return &types.StrategyConfig{
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
			Type:              "none",
			RequestsPerMinute: 0,
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
	}
}

// Merge merges config with defaults, preferring non-nil values from config
func Merge(defaults *types.StrategyConfig, c *types.StrategyConfig) *types.StrategyConfig {
	if c == nil {
		return defaults
	}

	merged := &types.StrategyConfig{}

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
