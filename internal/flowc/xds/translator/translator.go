package translator

import (
	"context"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

// XDSResources represents the complete set of xDS resources
type XDSResources struct {
	Clusters  []*clusterv3.Cluster
	Endpoints []*endpointv3.ClusterLoadAssignment
	Listeners []*listenerv3.Listener
	Routes    []*routev3.RouteConfiguration
}

// Translator is the interface that all xDS translators must implement
// It converts a FlowC DeploymentModel into Envoy xDS resources
type Translator interface {
	// Translate converts a deployment model into xDS resources
	Translate(ctx context.Context, model *DeploymentModel) (*XDSResources, error)

	// Name returns the name/type of this translator
	Name() string

	// Validate checks if the deployment model is valid for this translator
	Validate(model *DeploymentModel) error
}

// TranslatorOptions provides configuration options for translators
type TranslatorOptions struct {
	// DefaultListenerPort is the default port for listeners
	DefaultListenerPort uint32

	// EnableHTTPS enables HTTPS/TLS configuration
	EnableHTTPS bool

	// EnableTracing enables distributed tracing
	EnableTracing bool

	// EnableMetrics enables metrics collection
	EnableMetrics bool

	// Additional custom options
	CustomOptions map[string]interface{}
}

// DefaultTranslatorOptions returns default translator options
func DefaultTranslatorOptions() *TranslatorOptions {
	return &TranslatorOptions{
		DefaultListenerPort: 9095,
		EnableHTTPS:         false,
		EnableTracing:       false,
		EnableMetrics:       false,
		CustomOptions:       make(map[string]interface{}),
	}
}
