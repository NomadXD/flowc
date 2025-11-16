package config

import (
	"os"
	"strconv"
)

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *Config) {
	// Server configuration overrides
	if val := os.Getenv("FLOWC_API_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 && port < 65536 {
			config.Server.APIPort = port
		}
	}

	if val := os.Getenv("FLOWC_XDS_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 && port < 65536 {
			config.Server.XDSPort = port
		}
	}

	if val := os.Getenv("FLOWC_READ_TIMEOUT"); val != "" {
		config.Server.ReadTimeout = val
	}

	if val := os.Getenv("FLOWC_WRITE_TIMEOUT"); val != "" {
		config.Server.WriteTimeout = val
	}

	if val := os.Getenv("FLOWC_IDLE_TIMEOUT"); val != "" {
		config.Server.IdleTimeout = val
	}

	if val := os.Getenv("FLOWC_SHUTDOWN_TIMEOUT"); val != "" {
		config.Server.ShutdownTimeout = val
	}

	if val := os.Getenv("FLOWC_GRACEFUL_SHUTDOWN"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Server.GracefulShutdown = enabled
		}
	}

	// XDS configuration overrides
	if val := os.Getenv("FLOWC_DEFAULT_LISTENER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 && port < 65536 {
			config.XDS.DefaultListenerPort = port
		}
	}

	if val := os.Getenv("FLOWC_DEFAULT_NODE_ID"); val != "" {
		config.XDS.DefaultNodeID = val
	}

	if val := os.Getenv("FLOWC_XDS_ADS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.XDS.SnapshotCache.ADS = enabled
		}
	}

	if val := os.Getenv("FLOWC_GRPC_KEEPALIVE_TIME"); val != "" {
		config.XDS.GRPC.KeepaliveTime = val
	}

	if val := os.Getenv("FLOWC_GRPC_KEEPALIVE_TIMEOUT"); val != "" {
		config.XDS.GRPC.KeepaliveTimeout = val
	}

	if val := os.Getenv("FLOWC_GRPC_KEEPALIVE_MIN_TIME"); val != "" {
		config.XDS.GRPC.KeepaliveMinTime = val
	}

	if val := os.Getenv("FLOWC_GRPC_KEEPALIVE_PERMIT_WITHOUT_STREAM"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.XDS.GRPC.KeepalivePermitWithoutStream = enabled
		}
	}

	// Logging configuration overrides
	if val := os.Getenv("FLOWC_LOG_LEVEL"); val != "" {
		config.Logging.Level = val
	}

	if val := os.Getenv("FLOWC_LOG_FORMAT"); val != "" {
		config.Logging.Format = val
	}

	if val := os.Getenv("FLOWC_LOG_OUTPUT"); val != "" {
		config.Logging.Output = val
	}

	if val := os.Getenv("FLOWC_LOG_STRUCTURED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Logging.Structured = enabled
		}
	}

	if val := os.Getenv("FLOWC_LOG_ENABLE_CALLER"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Logging.EnableCaller = enabled
		}
	}

	if val := os.Getenv("FLOWC_LOG_ENABLE_STACKTRACE"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Logging.EnableStacktrace = enabled
		}
	}

	// Feature flags overrides
	if val := os.Getenv("FLOWC_FEATURE_EXTERNAL_TRANSLATORS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.ExternalTranslators = enabled
		}
	}

	if val := os.Getenv("FLOWC_FEATURE_OPENAPI_VALIDATION"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.OpenAPIValidation = enabled
		}
	}

	if val := os.Getenv("FLOWC_FEATURE_METRICS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.Metrics = enabled
		}
	}

	if val := os.Getenv("FLOWC_FEATURE_TRACING"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.Tracing = enabled
		}
	}

	if val := os.Getenv("FLOWC_FEATURE_RATE_LIMITING"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Features.RateLimiting = enabled
		}
	}
}
