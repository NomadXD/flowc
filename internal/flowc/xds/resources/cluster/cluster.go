package cluster

import (
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

// CreateCluster creates a cluster configuration
func CreateCluster(clusterName, serviceName string, port uint32) *clusterv3.Cluster {
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
