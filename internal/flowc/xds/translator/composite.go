package translator

import (
	"context"
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/listener"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// CompositeTranslator orchestrates multiple strategies to generate xDS resources
// It implements the Translator interface while delegating to specialized strategies
type CompositeTranslator struct {
	// Strategy set
	strategies *StrategySet

	// Options
	options *TranslatorOptions

	// Logger
	logger *logger.EnvoyLogger
}

// NewCompositeTranslator creates a new composite translator
func NewCompositeTranslator(strategies *StrategySet, options *TranslatorOptions, log *logger.EnvoyLogger) (*CompositeTranslator, error) {
	if strategies == nil {
		return nil, fmt.Errorf("strategies cannot be nil")
	}

	// Validate required strategies
	if err := strategies.Validate(); err != nil {
		return nil, fmt.Errorf("invalid strategy set: %w", err)
	}

	if options == nil {
		options = DefaultTranslatorOptions()
	}

	// Ensure optional strategies have no-op implementations if nil
	if strategies.LoadBalancing == nil {
		strategies.LoadBalancing = &NoOpLoadBalancingStrategy{}
	}
	if strategies.Retry == nil {
		strategies.Retry = &NoOpRetryStrategy{}
	}
	if strategies.RateLimit == nil {
		strategies.RateLimit = &NoOpRateLimitStrategy{}
	}
	if strategies.Observability == nil {
		strategies.Observability = &NoOpObservabilityStrategy{}
	}

	return &CompositeTranslator{
		strategies: strategies,
		options:    options,
		logger:     log,
	}, nil
}

// Name returns the translator name
func (t *CompositeTranslator) Name() string {
	return fmt.Sprintf("composite[deployment=%s,route=%s,lb=%s,retry=%s]",
		t.strategies.Deployment.Name(),
		t.strategies.RouteMatch.Name(),
		t.strategies.LoadBalancing.Name(),
		t.strategies.Retry.Name(),
	)
}

// Validate validates the deployment model
func (t *CompositeTranslator) Validate(model *DeploymentModel) error {
	// Validate with deployment strategy (most critical)
	return t.strategies.Deployment.Validate(model)
}

// Translate converts a deployment model into xDS resources
func (t *CompositeTranslator) Translate(ctx context.Context, model *DeploymentModel) (*XDSResources, error) {
	if err := t.Validate(model); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if t.logger != nil {
		t.logger.WithFields(map[string]interface{}{
			"translator":          t.Name(),
			"deployment":          model.DeploymentID,
			"deployment_strategy": t.strategies.Deployment.Name(),
			"route_strategy":      t.strategies.RouteMatch.Name(),
			"lb_strategy":         t.strategies.LoadBalancing.Name(),
			"retry_strategy":      t.strategies.Retry.Name(),
		}).Info("Starting xDS translation with composite strategy")
	}

	// PHASE 1: Generate clusters using deployment strategy
	clusters, err := t.strategies.Deployment.GenerateClusters(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("cluster generation failed: %w", err)
	}

	if t.logger != nil {
		t.logger.WithFields(map[string]interface{}{
			"clusters_count": len(clusters),
		}).Debug("Generated clusters")
	}

	// PHASE 2: Apply load balancing strategy to clusters
	for _, cluster := range clusters {
		if err := t.strategies.LoadBalancing.ConfigureCluster(cluster, model); err != nil {
			return nil, fmt.Errorf("load balancing configuration failed for cluster %s: %w", cluster.Name, err)
		}
	}

	// PHASE 3: Generate routes
	routes, err := t.generateRoutes(ctx, model, clusters)
	if err != nil {
		return nil, fmt.Errorf("route generation failed: %w", err)
	}

	if t.logger != nil {
		t.logger.WithFields(map[string]interface{}{
			"route_configs_count": len(routes),
		}).Debug("Generated routes")
	}

	// PHASE 4: Apply retry strategy to routes
	for _, routeConfig := range routes {
		for _, vhost := range routeConfig.VirtualHosts {
			for _, route := range vhost.Routes {
				if err := t.strategies.Retry.ConfigureRetry(route, model); err != nil {
					return nil, fmt.Errorf("retry configuration failed: %w", err)
				}
			}
		}
	}

	// PHASE 5: Generate listeners (if needed)
	var listeners []*listenerv3.Listener
	if t.shouldGenerateListener(model) {
		listeners = append(listeners, t.generateListener(model, routes))
	}

	// PHASE 6: Apply rate limiting to listeners
	for _, listener := range listeners {
		if err := t.strategies.RateLimit.ConfigureRateLimit(listener, model); err != nil {
			return nil, fmt.Errorf("rate limit configuration failed: %w", err)
		}
	}

	// PHASE 7: Apply observability configuration
	if len(listeners) > 0 {
		if err := t.strategies.Observability.ConfigureObservability(listeners[0], clusters, model); err != nil {
			return nil, fmt.Errorf("observability configuration failed: %w", err)
		}
	}

	if t.logger != nil {
		t.logger.WithFields(map[string]interface{}{
			"clusters":  len(clusters),
			"routes":    len(routes),
			"listeners": len(listeners),
		}).Info("Successfully completed xDS translation")
	}

	return &XDSResources{
		Clusters:  clusters,
		Routes:    routes,
		Listeners: listeners,
		Endpoints: nil, // Typically not needed for LOGICAL_DNS clusters
	}, nil
}

// generateRoutes creates route configurations from OpenAPI spec
func (t *CompositeTranslator) generateRoutes(ctx context.Context, model *DeploymentModel, clusters []*clusterv3.Cluster) ([]*routev3.RouteConfiguration, error) {
	spec := model.OpenAPISpec
	metadata := model.Metadata

	if spec == nil || spec.Paths == nil {
		return []*routev3.RouteConfiguration{}, nil
	}

	// Get cluster names from deployment strategy
	clusterNames := t.strategies.Deployment.GetClusterNames(model)
	if len(clusterNames) == 0 {
		return nil, fmt.Errorf("no cluster names returned from deployment strategy")
	}

	// Primary cluster is the first one (or only one for basic deployments)
	primaryCluster := clusterNames[0]

	var xdsRoutes []*routev3.Route

	// Create routes for each OpenAPI path and method
	for path, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}

		// Build the full path with context prefix
		fullPath := metadata.Context
		if fullPath != "" && fullPath[0] != '/' {
			fullPath = "/" + fullPath
		}
		fullPath = fullPath + path

		// Create routes for each HTTP method
		operations := map[string]interface{}{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			// Use route match strategy to create matcher
			routeMatch := t.strategies.RouteMatch.CreateMatcher(fullPath, method, nil)

			// Create route with primary cluster as destination
			// (Deployment strategies like canary will override this with weighted clusters)
			route := &routev3.Route{
				Match: routeMatch,
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: primaryCluster,
						},
					},
				},
			}

			xdsRoutes = append(xdsRoutes, route)
		}
	}

	// Create route configuration
	// routeName := fmt.Sprintf("%s-%s-route", metadata.Name, metadata.Version)
	routeName := "flowc_default_route"
	routeConfig := &routev3.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*routev3.VirtualHost{
			{
				Name:    t.generateVirtualHostName(model),
				Domains: t.getDomains(model),
				Routes:  xdsRoutes,
			},
		},
	}

	return []*routev3.RouteConfiguration{routeConfig}, nil
}

// generateListener creates a listener
func (t *CompositeTranslator) generateListener(model *DeploymentModel, routes []*routev3.RouteConfiguration) *listenerv3.Listener {
	listenerName := fmt.Sprintf("%s-%s-listener", model.Metadata.Name, model.Metadata.Version)
	routeName := routes[0].Name // Use first route config name

	return listener.CreateListener(listenerName, routeName, t.options.DefaultListenerPort)
}

// shouldGenerateListener determines if a dedicated listener should be created
func (t *CompositeTranslator) shouldGenerateListener(model *DeploymentModel) bool {
	// Check if model explicitly requests a dedicated listener
	if model.Context != nil && model.Context.CustomConfig != nil {
		if dedicatedListener, ok := model.Context.CustomConfig["dedicated_listener"].(bool); ok {
			return dedicatedListener
		}
	}
	return false
}

// generateVirtualHostName creates a virtual host name
func (t *CompositeTranslator) generateVirtualHostName(model *DeploymentModel) string {
	if model.Metadata.Gateway.VirtualHost.Name != "" {
		return model.Metadata.Gateway.VirtualHost.Name
	}
	return fmt.Sprintf("%s-%s-vhost", model.Metadata.Name, model.Metadata.Version)
}

// getDomains returns the domains for the virtual host
func (t *CompositeTranslator) getDomains(model *DeploymentModel) []string {
	if len(model.Metadata.Gateway.VirtualHost.Domains) > 0 {
		return model.Metadata.Gateway.VirtualHost.Domains
	}
	return []string{"*"} // Default to wildcard
}
