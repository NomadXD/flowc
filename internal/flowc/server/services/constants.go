package service

// Default values for gateway, listener, and environment creation

const (
	// DefaultListenerPort is the default port for listeners when not specified.
	// This can be overridden by the control plane configuration (flowc-config.yaml).
	DefaultListenerPort = 10000

	// DefaultListenerAddress is the default bind address for listeners
	DefaultListenerAddress = "0.0.0.0"

	// DefaultEnvironmentName is the default name for auto-created environments
	DefaultEnvironmentName = "production"

	// DefaultEnvironmentHostname is the default SNI hostname for auto-created environments.
	// The wildcard "*" acts as a catch-all for any hostname.
	DefaultEnvironmentHostname = "*"
)
