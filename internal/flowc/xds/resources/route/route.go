package route

import (
	"fmt"
	"strings"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

func CreateRoute(routeName string, clusterName string, context string) *routev3.Route {
	return &routev3.Route{
		Name: routeName,
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: context,
			},
		},
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_Cluster{
					Cluster: clusterName,
				},
			},
		},
	}
}

// createRouteForOperation creates an xDS route for a specific OpenAPI operation
func CreateRouteForOperation(path string, method string, clusterName string) *routev3.Route {
	// Determine the path match type
	var pathMatch *routev3.RouteMatch

	// Check if the path has parameters (e.g., /pets/{petId})
	if containsPathParams(path) {
		// Use regex match for paths with parameters
		// Convert OpenAPI path params {param} to regex (.+)
		regexPath := convertPathToRegex(path)
		pathMatch = &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					Regex: regexPath,
				},
			},
		}
	} else {
		// Use exact match for static paths
		pathMatch = &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Path{
				Path: path,
			},
		}
	}

	// Add method matching
	pathMatch.Headers = []*routev3.HeaderMatcher{
		{
			Name: ":method",
			HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
				ExactMatch: method,
			},
		},
	}

	return &routev3.Route{
		Name:  fmt.Sprintf("%s-%s", method, path),
		Match: pathMatch,
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_Cluster{
					Cluster: clusterName,
				},
				// Optionally add timeout, retry policy, etc.
			},
		},
	}
}

// containsPathParams checks if a path contains OpenAPI path parameters
func containsPathParams(path string) bool {
	return strings.Contains(path, "{") && strings.Contains(path, "}")
}

// convertPathToRegex converts OpenAPI path format to regex
// e.g., /pets/{petId} -> ^/pets/[^/]+$
func convertPathToRegex(path string) string {
	// Replace {param} with regex pattern
	regex := path
	regex = strings.ReplaceAll(regex, "{", "")
	regex = strings.ReplaceAll(regex, "}", "")

	// Split by / and rebuild with proper regex
	parts := strings.Split(regex, "/")
	for i, part := range parts {
		if part != "" && !strings.Contains(path, "{"+part+"}") {
			// Static part, keep as is
			parts[i] = part
		} else if part != "" {
			// Parameter part, match anything except /
			parts[i] = "[^/]+"
		}
	}

	return "^" + strings.Join(parts, "/") + "$"
}
