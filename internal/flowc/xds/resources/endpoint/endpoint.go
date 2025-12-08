package endpoint

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

// CreateLbEndpoint creates a load balancer endpoint configuration
func CreateLbEndpoint(clusterName, address string, port uint32) *endpointv3.ClusterLoadAssignment {
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
