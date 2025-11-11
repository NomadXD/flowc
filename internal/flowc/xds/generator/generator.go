package generator

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/cluster"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/listener"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/route"
	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
)

type ResourceGenerator struct {
	// logger for future use
	// logger *logger.EnvoyLogger
}

// GenerateOptions contains options for resource generation
type GenerateOptions struct {
	// Listener configuration
	ListenerPort uint32

	// Custom naming strategies
	ClusterNameFn  func(name, version string) string
	ListenerNameFn func(name, version string) string
	RouteNameFn    func(name, version string) string
}

// DefaultGenerateOptions returns default generation options
func DefaultGenerateOptions() *GenerateOptions {
	return &GenerateOptions{
		ListenerPort: 9095,
		ClusterNameFn: func(name, version string) string {
			return fmt.Sprintf("%s-%s-cluster", name, version)
		},
		ListenerNameFn: func(name, version string) string {
			return fmt.Sprintf("%s-%s-listener", name, version)
		},
		RouteNameFn: func(name, version string) string {
			return fmt.Sprintf("%s-%s-route", name, version)
		},
	}
}
func NewResourceGenerator() *ResourceGenerator {
	return &ResourceGenerator{}
}

// GenerateResources generates xDS resources for a deployment
// Returns only the cluster and routes - listener is managed separately as a shared resource
func (g *ResourceGenerator) GenerateResources(deployment *types.FlowCMetadata, spec *openapi3.T, options *GenerateOptions) (*models.XDSResources, error) {
	if options == nil {
		options = DefaultGenerateOptions()
	}

	clusterName := options.ClusterNameFn(deployment.Name, deployment.Version)

	// Generate cluster for this deployment's upstream
	cluster := g.generateClusters(clusterName, deployment.Upstream.Host, deployment.Upstream.Port, deployment.Upstream.Scheme)

	// Generate routes for this deployment's OpenAPI paths
	routes := g.generateRoutes(clusterName, spec, deployment.Context, options)

	return &models.XDSResources{
		Clusters: []*clusterv3.Cluster{cluster},
		Routes:   routes,
		// Endpoints: nil, // Not needed - embedded in cluster (LOGICAL_DNS)
		// Listeners: nil, // Not needed - using shared default listener
	}, nil
}

func (g *ResourceGenerator) generateClusters(clusterName string, host string, port uint32, scheme string) *clusterv3.Cluster {
	if scheme == "" {
		scheme = "http" // Default to HTTP if not specified
	}
	return cluster.CreateClusterWithScheme(clusterName, host, port, scheme)
}

func (g *ResourceGenerator) generateRoutes(clusterName string, spec *openapi3.T, contextPath string, _ *GenerateOptions) []*routev3.RouteConfiguration {
	if spec == nil || spec.Paths == nil {
		return []*routev3.RouteConfiguration{}
	}

	// Create routes for each path and method combination
	var xdsRoutes []*routev3.Route

	for path, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}

		// Build the full path with context prefix
		fullPath := contextPath
		if fullPath != "" && fullPath[0] != '/' {
			fullPath = "/" + fullPath
		}
		fullPath = fullPath + path

		// Create a route for each HTTP method in this path
		operations := map[string]*openapi3.Operation{
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

			// Create a route that matches this specific path and method
			xdsRoute := route.CreateRouteForOperation(fullPath, method, clusterName)
			xdsRoutes = append(xdsRoutes, xdsRoute)
		}
	}

	// Wrap all routes in a RouteConfiguration
	routeConfig := &routev3.RouteConfiguration{
		Name: "flowc_default_route", // You can customize this
		VirtualHosts: []*routev3.VirtualHost{
			{
				Name:    "backend",
				Domains: []string{"*"}, // Accept all domains, can be customized
				Routes:  xdsRoutes,
			},
		},
	}

	return []*routev3.RouteConfiguration{routeConfig}
}

// CreateDefaultListener creates the default shared listener
// This should be called once at startup, not per deployment
func CreateDefaultListener(listenerPort uint32, routeConfigName string) *listenerv3.Listener {
	return listener.CreateListener("flowc_default_listener", routeConfigName, listenerPort)
}
