package generator

import (
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/cluster"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/endpoint"
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

func (g *ResourceGenerator) GenerateResources(deployment *types.FlowCMetadata, spec *openapi3.T, options *GenerateOptions) (*models.XDSResources, error) {
	if options == nil {
		options = DefaultGenerateOptions()
	}

	clusterName := options.ClusterNameFn(deployment.Name, deployment.Version)
	listenerName := options.ListenerNameFn(deployment.Name, deployment.Version)
	routeName := options.RouteNameFn(deployment.Name, deployment.Version)

	cluster := g.generateClusters(clusterName, deployment.Upstream.Host, deployment.Upstream.Port)
	routes := g.generateRoutes(clusterName, spec, deployment.Context, options)
	endpoints := g.generateEndpoints(clusterName, deployment.Upstream.Host, deployment.Upstream.Port)
	listener := g.generateListeners(listenerName, routeName, options.ListenerPort)

	return &models.XDSResources{
		Clusters:  []*clusterv3.Cluster{cluster},
		Routes:    routes,
		Endpoints: []*endpointv3.ClusterLoadAssignment{endpoints},
		Listeners: []*listenerv3.Listener{listener},
	}, nil
}

func (g *ResourceGenerator) generateClusters(clusterName string, host string, port uint32) *clusterv3.Cluster {
	return cluster.CreateCluster(clusterName, host, port)
}

func (g *ResourceGenerator) generateListeners(listenerName string, routeName string, port uint32) *listenerv3.Listener {
	return listener.CreateListener(listenerName, routeName, port)
}

func (g *ResourceGenerator) generateRoutes(clusterName string, spec *openapi3.T, contextPath string, options *GenerateOptions) []*routev3.RouteConfiguration {
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
		Name: options.RouteNameFn("main", "v1"), // You can customize this
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

func (g *ResourceGenerator) generateEndpoints(clusterName string, host string, port uint32) *endpointv3.ClusterLoadAssignment {
	return endpoint.CreateLbEndpoint(clusterName, host, port)
}
