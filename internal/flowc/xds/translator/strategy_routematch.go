package translator

import (
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/flowc-labs/flowc/internal/flowc/ir"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// =============================================================================
// ROUTE MATCH STRATEGIES
// =============================================================================

// PrefixRouteMatchStrategy matches routes by prefix
type PrefixRouteMatchStrategy struct {
	caseSensitive bool
}

func NewPrefixRouteMatchStrategy(caseSensitive bool) *PrefixRouteMatchStrategy {
	return &PrefixRouteMatchStrategy{
		caseSensitive: caseSensitive,
	}
}

func (s *PrefixRouteMatchStrategy) CreateMatcher(path, method string, endpoint *ir.Endpoint) *routev3.RouteMatch {
	return &routev3.RouteMatch{
		PathSpecifier: &routev3.RouteMatch_Prefix{
			Prefix: path,
		},
		Headers: []*routev3.HeaderMatcher{
			{
				Name: ":method",
				HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
					StringMatch: &matcherv3.StringMatcher{
						MatchPattern: &matcherv3.StringMatcher_Exact{
							Exact: method,
						},
					},
				},
			},
		},
		CaseSensitive: wrapperspb.Bool(s.caseSensitive),
	}
}

func (s *PrefixRouteMatchStrategy) Name() string {
	return "prefix"
}

// ExactRouteMatchStrategy matches routes exactly
type ExactRouteMatchStrategy struct {
	caseSensitive bool
}

func NewExactRouteMatchStrategy(caseSensitive bool) *ExactRouteMatchStrategy {
	return &ExactRouteMatchStrategy{
		caseSensitive: caseSensitive,
	}
}

func (s *ExactRouteMatchStrategy) CreateMatcher(path, method string, endpoint *ir.Endpoint) *routev3.RouteMatch {
	return &routev3.RouteMatch{
		PathSpecifier: &routev3.RouteMatch_Path{
			Path: path,
		},
		Headers: []*routev3.HeaderMatcher{
			{
				Name: ":method",
				HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
					StringMatch: &matcherv3.StringMatcher{
						MatchPattern: &matcherv3.StringMatcher_Exact{
							Exact: method,
						},
					},
				},
			},
		},
		CaseSensitive: wrapperspb.Bool(s.caseSensitive),
	}
}

func (s *ExactRouteMatchStrategy) Name() string {
	return "exact"
}

// RegexRouteMatchStrategy matches routes by regex
type RegexRouteMatchStrategy struct {
	caseSensitive bool
}

func NewRegexRouteMatchStrategy(caseSensitive bool) *RegexRouteMatchStrategy {
	return &RegexRouteMatchStrategy{
		caseSensitive: caseSensitive,
	}
}

func (s *RegexRouteMatchStrategy) CreateMatcher(path, method string, endpoint *ir.Endpoint) *routev3.RouteMatch {
	// Convert OpenAPI path parameters to regex
	// e.g., /users/{id} -> /users/[^/]+
	regexPath := convertPathToRegex(path)

	return &routev3.RouteMatch{
		PathSpecifier: &routev3.RouteMatch_SafeRegex{
			SafeRegex: &matcherv3.RegexMatcher{
				Regex: regexPath,
			},
		},
		Headers: []*routev3.HeaderMatcher{
			{
				Name: ":method",
				HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
					StringMatch: &matcherv3.StringMatcher{
						MatchPattern: &matcherv3.StringMatcher_Exact{
							Exact: method,
						},
					},
				},
			},
		},
		CaseSensitive: wrapperspb.Bool(s.caseSensitive),
	}
}

func (s *RegexRouteMatchStrategy) Name() string {
	return "regex"
}

// HeaderVersionedRouteMatchStrategy routes based on API version in header
type HeaderVersionedRouteMatchStrategy struct {
	versionHeader string
	caseSensitive bool
}

func NewHeaderVersionedRouteMatchStrategy(versionHeader string, caseSensitive bool) *HeaderVersionedRouteMatchStrategy {
	if versionHeader == "" {
		versionHeader = "x-api-version"
	}
	return &HeaderVersionedRouteMatchStrategy{
		versionHeader: versionHeader,
		caseSensitive: caseSensitive,
	}
}

func (s *HeaderVersionedRouteMatchStrategy) CreateMatcher(path, method string, endpoint *ir.Endpoint) *routev3.RouteMatch {
	return &routev3.RouteMatch{
		PathSpecifier: &routev3.RouteMatch_Prefix{
			Prefix: path,
		},
		Headers: []*routev3.HeaderMatcher{
			{
				Name: ":method",
				HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
					StringMatch: &matcherv3.StringMatcher{
						MatchPattern: &matcherv3.StringMatcher_Exact{
							Exact: method,
						},
					},
				},
			},
			// Add version header matching if needed
			// This can be customized based on the deployment
		},
		CaseSensitive: wrapperspb.Bool(s.caseSensitive),
	}
}

func (s *HeaderVersionedRouteMatchStrategy) Name() string {
	return "header-versioned"
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// convertPathToRegex converts OpenAPI path with parameters to regex
// e.g., /users/{id} -> /users/[^/]+
// e.g., /users/{id}/posts/{postId} -> /users/[^/]+/posts/[^/]+
func convertPathToRegex(path string) string {
	// Simple implementation - replace {param} with [^/]+
	inParam := false
	var builder []rune

	for _, ch := range path {
		if ch == '{' {
			inParam = true
			continue
		}
		if ch == '}' {
			inParam = false
			builder = append(builder, []rune("[^/]+")...)
			continue
		}
		if !inParam {
			// Escape regex special characters
			if ch == '.' || ch == '*' || ch == '+' || ch == '?' || ch == '^' || ch == '$' || ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '|' {
				builder = append(builder, '\\')
			}
			builder = append(builder, ch)
		}
	}

	return string(builder)
}
