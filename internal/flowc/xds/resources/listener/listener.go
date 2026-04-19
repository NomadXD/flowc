package listener

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	tlsinspectorv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/flowc-labs/flowc/pkg/types"
)

const (
	DefaultListenerName = "flowc_default_listener"
	DefaultListenerPort = 9095
	DefaultRouteName    = "flowc_default_route"
	DefaultNodeID       = "test-envoy-node"
)

// CreateListener creates a listener configuration
func CreateListener(listenerName, routeName string, port uint32) *listenerv3.Listener {
	routerConfig, _ := anypb.New(&routerv3.Router{})
	// HTTP filter configuration
	manager := &hcmv3.HttpConnectionManager{
		CodecType:  hcmv3.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				ConfigSource:    createXdsConfigSource(),
				RouteConfigName: routeName,
			},
		},
		HttpFilters: []*hcmv3.HttpFilter{{
			Name:       "http-router",
			ConfigType: &hcmv3.HttpFilter_TypedConfig{TypedConfig: routerConfig},
		}},
	}
	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	// For now, return a placeholder resource
	return &listenerv3.Listener{
		Name: listenerName,
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listenerv3.FilterChain{
			{
				Filters: []*listenerv3.Filter{
					{
						Name: "http_connection_manager",
						ConfigType: &listenerv3.Filter_TypedConfig{
							TypedConfig: pbst,
						},
					},
				},
			},
		},
	}
}

func createXdsConfigSource() *corev3.ConfigSource {
	source := &corev3.ConfigSource{}
	source.ResourceApiVersion = resourcev3.DefaultAPIVersion
	source.ConfigSourceSpecifier = &corev3.ConfigSource_Ads{
		Ads: &corev3.AggregatedConfigSource{},
	}
	return source
}

// FilterChainConfig contains configuration for a single filter chain with SNI matching
type FilterChainConfig struct {
	// Name of the filter chain (for logging/debugging)
	Name string

	// Hostname for SNI matching (e.g., "api.example.com")
	Hostname string

	// HTTPFilters are environment-specific HTTP filters to apply
	HTTPFilters []types.HTTPFilter

	// RouteConfigName is the name of the RDS route configuration
	RouteConfigName string

	// TLS configuration for this filter chain
	TLS *TLSConfig
}

// TLSConfig contains TLS settings for a filter chain
type TLSConfig struct {
	CertPath          string
	KeyPath           string
	CAPath            string
	RequireClientCert bool
	MinVersion        string
	CipherSuites      []string
}

// ListenerConfig contains configuration for creating a listener with multiple filter chains
type ListenerConfig struct {
	// Name of the listener
	Name string

	// Port to bind to
	Port uint32

	// Address to bind to (default: "0.0.0.0")
	Address string

	// FilterChains for SNI-based routing
	FilterChains []*FilterChainConfig

	// HTTP2 enables HTTP/2 support
	HTTP2 bool

	// AccessLog path
	AccessLog string
}

// CreateListenerWithFilterChains creates a listener with multiple SNI-matched filter chains.
// This is used for environment-based routing where each environment has its own hostname.
func CreateListenerWithFilterChains(config *ListenerConfig) (*listenerv3.Listener, error) {
	if config.Address == "" {
		config.Address = "0.0.0.0"
	}

	filterChains := make([]*listenerv3.FilterChain, 0, len(config.FilterChains))

	// Track whether any filter chain needs TLS — only then do we add the
	// tls_inspector listener filter and SNI-based server_names matching.
	hasTLS := false
	for _, fc := range config.FilterChains {
		if fc.TLS != nil {
			hasTLS = true
			break
		}
	}

	for _, fcConfig := range config.FilterChains {
		// Create HTTP Connection Manager for this filter chain
		routerConfig, _ := anypb.New(&routerv3.Router{})

		// TODO: Add environment-specific HTTP filters from fcConfig.HTTPFilters
		httpFilters := []*hcmv3.HttpFilter{{
			Name:       "http-router",
			ConfigType: &hcmv3.HttpFilter_TypedConfig{TypedConfig: routerConfig},
		}}

		manager := &hcmv3.HttpConnectionManager{
			CodecType:  hcmv3.HttpConnectionManager_AUTO,
			StatPrefix: "http",
			RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
				Rds: &hcmv3.Rds{
					ConfigSource:    createXdsConfigSource(),
					RouteConfigName: fcConfig.RouteConfigName,
				},
			},
			HttpFilters: httpFilters,
		}

		if config.HTTP2 {
			manager.Http2ProtocolOptions = &corev3.Http2ProtocolOptions{}
		}

		pbst, err := anypb.New(manager)
		if err != nil {
			return nil, err
		}

		filterChain := &listenerv3.FilterChain{
			Filters: []*listenerv3.Filter{
				{
					Name: "http_connection_manager",
					ConfigType: &listenerv3.Filter_TypedConfig{
						TypedConfig: pbst,
					},
				},
			},
		}

		// SNI-based server_names matching only works with TLS (the tls_inspector
		// extracts the SNI from the ClientHello). For plain HTTP listeners we
		// skip server_names entirely — hostname routing is handled at the
		// virtual-host level in the route configuration instead.
		if hasTLS && fcConfig.Hostname != "" && fcConfig.Hostname != "*" {
			filterChain.FilterChainMatch = &listenerv3.FilterChainMatch{
				ServerNames: []string{fcConfig.Hostname},
			}
		}

		// TODO: Add TLS configuration if fcConfig.TLS is set

		filterChains = append(filterChains, filterChain)
	}

	l := &listenerv3.Listener{
		Name: config.Name,
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Address: config.Address,
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: config.Port,
					},
				},
			},
		},
		FilterChains: filterChains,
	}

	// Only add the tls_inspector when at least one filter chain uses TLS.
	// Without TLS there is no ClientHello for the inspector to parse, and
	// adding it to a plain HTTP listener causes Envoy to drop connections.
	if hasTLS {
		tlsInspector, err := anypb.New(&tlsinspectorv3.TlsInspector{})
		if err != nil {
			return nil, err
		}
		l.ListenerFilters = []*listenerv3.ListenerFilter{
			{
				Name: "envoy.filters.listener.tls_inspector",
				ConfigType: &listenerv3.ListenerFilter_TypedConfig{
					TypedConfig: tlsInspector,
				},
			},
		}
	}

	return l, nil
}
