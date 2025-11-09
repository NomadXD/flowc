package handlers

import (
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/flowc-labs/flowc/pkg/logger"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	ClusterName  = "example_proxy_cluster"
	RouteName    = "local_route"
	ListenerName = "listener_0"
	ListenerPort = 10000
	UpstreamHost = "www.envoyproxy.io"
	UpstreamPort = 80
)

// XDSHandlers provides methods to create basic XDS resources
type XDSHandlers struct {
	logger *logger.EnvoyLogger
}

// NewXDSHandlers creates a new XDS handlers instance
func NewXDSHandlers(logger *logger.EnvoyLogger) *XDSHandlers {
	return &XDSHandlers{
		logger: logger,
	}
}

// CreateBasicCluster creates a basic cluster configuration
func (h *XDSHandlers) CreateBasicCluster(clusterName, serviceName string, port uint32) *clusterv3.Cluster {
	h.logger.WithFields(map[string]interface{}{
		"clusterName": clusterName,
		"serviceName": serviceName,
		"port":        port,
	}).Info("Creating basic cluster")

	return &clusterv3.Cluster{
		Name:           clusterName,
		ConnectTimeout: durationpb.New(5 * time.Second),
		// Use LOGICAL_DNS for hostname resolution
		ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_LOGICAL_DNS},
		LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
		DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
		// Load assignment with hostname
		LoadAssignment: &endpointv3.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints: []*endpointv3.LocalityLbEndpoints{
				{
					LbEndpoints: []*endpointv3.LbEndpoint{
						{
							HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
								Endpoint: &endpointv3.Endpoint{
									Address: &corev3.Address{
										Address: &corev3.Address_SocketAddress{
											SocketAddress: &corev3.SocketAddress{
												Address: serviceName,
												PortSpecifier: &corev3.SocketAddress_PortValue{
													PortValue: port,
												},
												Protocol: corev3.SocketAddress_TCP,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateBasicEndpoint creates a basic endpoint configuration
func (h *XDSHandlers) CreateBasicEndpoint(clusterName, address string, port uint32) *endpointv3.ClusterLoadAssignment {
	h.logger.WithFields(map[string]interface{}{
		"address":     address,
		"port":        port,
		"clusterName": clusterName,
	}).Info("Creating basic endpoint")

	// For now, return a placeholder resource
	return &endpointv3.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpointv3.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointv3.LbEndpoint{
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Address: address,
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: port,
											},
											Protocol: corev3.SocketAddress_TCP,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateBasicListener creates a basic listener configuration
func (h *XDSHandlers) CreateBasicListener(listenerName, routeName string, port uint32) *listenerv3.Listener {
	h.logger.WithFields(map[string]interface{}{
		"listenerName": listenerName,
		"routeName":    routeName,
		"port":         port,
	}).Info("Creating basic listener")
	routerConfig, _ := anypb.New(&routerv3.Router{})
	// HTTP filter configuration
	manager := &hcmv3.HttpConnectionManager{
		CodecType:  hcmv3.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				ConfigSource:    makeConfigSource(),
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

// CreateBasicRoute creates a basic route configuration
func (h *XDSHandlers) CreateBasicRoute(routeName, clusterName, prefix string) *routev3.RouteConfiguration {
	h.logger.WithFields(map[string]interface{}{
		"routeName":   routeName,
		"clusterName": clusterName,
		"prefix":      prefix,
	}).Info("Creating basic route")

	// For now, return a placeholder resource
	return &routev3.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*routev3.VirtualHost{{
			Name:    "local_service",
			Domains: []string{"*"},
			Routes: []*routev3.Route{{
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: clusterName,
						},
						HostRewriteSpecifier: &routev3.RouteAction_HostRewriteLiteral{
							HostRewriteLiteral: UpstreamHost,
						},
					},
				},
			}},
		}},
	}
}

func makeConfigSource() *corev3.ConfigSource {
	source := &corev3.ConfigSource{}
	source.ResourceApiVersion = resourcev3.DefaultAPIVersion
	source.ConfigSourceSpecifier = &corev3.ConfigSource_Ads{
		Ads: &corev3.AggregatedConfigSource{},
	}
	return source
}
