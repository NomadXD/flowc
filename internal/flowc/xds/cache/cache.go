package cache

import (
	"context"
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// ConfigManager manages XDS configuration snapshots
type ConfigManager struct {
	cache  cachev3.SnapshotCache
	logger *logger.EnvoyLogger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(cache cachev3.SnapshotCache, logger *logger.EnvoyLogger) *ConfigManager {
	return &ConfigManager{
		cache:  cache,
		logger: logger,
	}
}

// UpdateSnapshot updates the configuration snapshot for a given node ID
func (cm *ConfigManager) UpdateSnapshot(nodeID string, snapshot *cachev3.Snapshot) error {
	if err := snapshot.Consistent(); err != nil {
		return fmt.Errorf("snapshot inconsistent: %w", err)
	}

	cm.cache.SetSnapshot(context.Background(), nodeID, snapshot)
	cm.logger.Infof("Updated snapshot for node %s", nodeID)
	return nil
}

// GetSnapshot retrieves the current snapshot for a given node ID
func (cm *ConfigManager) GetSnapshot(nodeID string) (*cachev3.Snapshot, error) {
	snapshot, err := cm.cache.GetSnapshot(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot for node %s: %w", nodeID, err)
	}
	// Type assert the ResourceSnapshot interface to concrete Snapshot type
	concreteSnapshot, ok := snapshot.(*cachev3.Snapshot)
	if !ok {
		return nil, fmt.Errorf("snapshot is not of type *cachev3.Snapshot")
	}
	return concreteSnapshot, nil
}

// CreateEmptySnapshot creates an empty snapshot for a node
func (cm *ConfigManager) CreateEmptySnapshot(nodeID string) (*cachev3.Snapshot, error) {
	snapshot, err := cachev3.NewSnapshot(
		"0", // Initial version for empty snapshot
		map[resourcev3.Type][]types.Resource{
			resourcev3.ClusterType:  {}, // Empty cluster list
			resourcev3.EndpointType: {}, // Empty endpoint list
			resourcev3.ListenerType: {}, // Empty listener list
			resourcev3.RouteType:    {}, // Empty route list
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}
	return snapshot, nil
}

// AddCluster adds a cluster configuration to the snapshot
func (cm *ConfigManager) AddCluster(nodeID string, clusterName string, clusterConfig *clusterv3.Cluster) error {
	// Set the cluster name
	clusterConfig.Name = clusterName

	// Get existing snapshot or create empty one
	snapshot, err := cm.GetSnapshot(nodeID)
	if err != nil {
		// Create new snapshot if it doesn't exist
		snapshot, err = cm.CreateEmptySnapshot(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	// Collect existing resources
	resources := make(map[resourcev3.Type][]types.Resource)

	// Copy existing clusters and add the new one
	existingClusters := snapshot.GetResources(resourcev3.ClusterType)
	clusterResources := make([]types.Resource, 0, len(existingClusters)+1)
	for _, res := range existingClusters {
		clusterResources = append(clusterResources, res)
	}
	clusterResources = append(clusterResources, clusterConfig)
	resources[resourcev3.ClusterType] = clusterResources

	// Copy other existing resources
	resources[resourcev3.EndpointType] = convertResourceMap(snapshot.GetResources(resourcev3.EndpointType))
	resources[resourcev3.ListenerType] = convertResourceMap(snapshot.GetResources(resourcev3.ListenerType))
	resources[resourcev3.RouteType] = convertResourceMap(snapshot.GetResources(resourcev3.RouteType))

	// Create new snapshot with incremented version
	newVersion := fmt.Sprintf("v%d", len(clusterResources))
	newSnapshot, err := cachev3.NewSnapshot(newVersion, resources)
	if err != nil {
		return fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Update the snapshot
	return cm.UpdateSnapshot(nodeID, newSnapshot)
}

// convertResourceMap converts a map[string]types.Resource to []types.Resource
func convertResourceMap(resourceMap map[string]types.Resource) []types.Resource {
	resources := make([]types.Resource, 0, len(resourceMap))
	for _, res := range resourceMap {
		resources = append(resources, res)
	}
	return resources
}

// APIDeployment represents a complete API deployment with all required resources
type APIDeployment struct {
	Clusters  []*clusterv3.Cluster
	Endpoints []*endpointv3.ClusterLoadAssignment
	Listeners []*listenerv3.Listener
	Routes    []*routev3.RouteConfiguration
}

// DeployAPI atomically deploys a complete API with all its resources
func (cm *ConfigManager) DeployAPI(nodeID string, deployment *APIDeployment) error {
	// Get existing snapshot or create empty one
	snapshot, err := cm.GetSnapshot(nodeID)
	if err != nil {
		snapshot, err = cm.CreateEmptySnapshot(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	// Collect existing resources
	resources := make(map[resourcev3.Type][]types.Resource)

	// Copy existing clusters and add new ones
	existingClusters := snapshot.GetResources(resourcev3.ClusterType)
	clusterResources := make([]types.Resource, 0, len(existingClusters)+len(deployment.Clusters))
	for _, res := range existingClusters {
		clusterResources = append(clusterResources, res)
	}
	for _, cluster := range deployment.Clusters {
		clusterResources = append(clusterResources, cluster)
	}
	resources[resourcev3.ClusterType] = clusterResources

	// Copy existing endpoints and add new ones
	existingEndpoints := snapshot.GetResources(resourcev3.EndpointType)
	endpointResources := make([]types.Resource, 0, len(existingEndpoints)+len(deployment.Endpoints))
	for _, res := range existingEndpoints {
		endpointResources = append(endpointResources, res)
	}
	for _, endpoint := range deployment.Endpoints {
		endpointResources = append(endpointResources, endpoint)
	}
	resources[resourcev3.EndpointType] = endpointResources

	// Copy existing listeners and add new ones
	existingListeners := snapshot.GetResources(resourcev3.ListenerType)
	listenerResources := make([]types.Resource, 0, len(existingListeners)+len(deployment.Listeners))
	for _, res := range existingListeners {
		listenerResources = append(listenerResources, res)
	}
	for _, listener := range deployment.Listeners {
		listenerResources = append(listenerResources, listener)
	}
	resources[resourcev3.ListenerType] = listenerResources

	// Copy existing routes and update/add new ones
	// For routes, we need to replace existing ones with the same name (for updates)
	// rather than just appending, which would create duplicates
	existingRoutes := snapshot.GetResources(resourcev3.RouteType)
	routeMap := make(map[string]types.Resource)

	// First, add all existing routes to the map
	for _, res := range existingRoutes {
		if routeConfig, ok := res.(*routev3.RouteConfiguration); ok {
			routeMap[routeConfig.Name] = res
		}
	}

	// Then, update/add new routes (this replaces existing ones with the same name)
	for _, route := range deployment.Routes {
		routeMap[route.Name] = route
	}

	// Convert map back to slice
	routeResources := make([]types.Resource, 0, len(routeMap))
	for _, res := range routeMap {
		routeResources = append(routeResources, res)
	}
	resources[resourcev3.RouteType] = routeResources

	// Create new snapshot with incremented version
	totalResources := len(clusterResources) + len(endpointResources) + len(listenerResources) + len(routeResources)
	newVersion := fmt.Sprintf("v%d", totalResources)
	newSnapshot, err := cachev3.NewSnapshot(newVersion, resources)
	if err != nil {
		return fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Atomically update the snapshot
	return cm.UpdateSnapshot(nodeID, newSnapshot)
}

// BulkResourceUpdate represents a bulk update of multiple resource types
type BulkResourceUpdate struct {
	AddClusters  []*clusterv3.Cluster
	AddEndpoints []*endpointv3.ClusterLoadAssignment
	AddListeners []*listenerv3.Listener
	AddRoutes    []*routev3.RouteConfiguration
	// Future: RemoveClusters, RemoveEndpoints, etc.
}

// BulkUpdate atomically updates multiple resources in a single snapshot operation
func (cm *ConfigManager) BulkUpdate(nodeID string, update *BulkResourceUpdate) error {
	// Get existing snapshot or create empty one
	snapshot, err := cm.GetSnapshot(nodeID)
	if err != nil {
		snapshot, err = cm.CreateEmptySnapshot(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	// Collect existing resources
	resources := make(map[resourcev3.Type][]types.Resource)

	// Handle clusters
	existingClusters := snapshot.GetResources(resourcev3.ClusterType)
	clusterResources := make([]types.Resource, 0, len(existingClusters)+len(update.AddClusters))
	for _, res := range existingClusters {
		clusterResources = append(clusterResources, res)
	}
	for _, cluster := range update.AddClusters {
		clusterResources = append(clusterResources, cluster)
	}
	resources[resourcev3.ClusterType] = clusterResources

	// Handle endpoints
	existingEndpoints := snapshot.GetResources(resourcev3.EndpointType)
	endpointResources := make([]types.Resource, 0, len(existingEndpoints)+len(update.AddEndpoints))
	for _, res := range existingEndpoints {
		endpointResources = append(endpointResources, res)
	}
	for _, endpoint := range update.AddEndpoints {
		endpointResources = append(endpointResources, endpoint)
	}
	resources[resourcev3.EndpointType] = endpointResources

	// Handle listeners
	existingListeners := snapshot.GetResources(resourcev3.ListenerType)
	listenerResources := make([]types.Resource, 0, len(existingListeners)+len(update.AddListeners))
	for _, res := range existingListeners {
		listenerResources = append(listenerResources, res)
	}
	for _, listener := range update.AddListeners {
		listenerResources = append(listenerResources, listener)
	}
	resources[resourcev3.ListenerType] = listenerResources

	// Handle routes
	existingRoutes := snapshot.GetResources(resourcev3.RouteType)
	routeResources := make([]types.Resource, 0, len(existingRoutes)+len(update.AddRoutes))
	for _, res := range existingRoutes {
		routeResources = append(routeResources, res)
	}
	for _, route := range update.AddRoutes {
		routeResources = append(routeResources, route)
	}
	resources[resourcev3.RouteType] = routeResources

	// Create new snapshot with incremented version
	totalResources := len(clusterResources) + len(endpointResources) + len(listenerResources) + len(routeResources)
	newVersion := fmt.Sprintf("v%d", totalResources)
	newSnapshot, err := cachev3.NewSnapshot(newVersion, resources)
	if err != nil {
		return fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Atomically update the snapshot
	return cm.UpdateSnapshot(nodeID, newSnapshot)
}

// AddEndpoint adds an endpoint configuration to the snapshot
func (cm *ConfigManager) AddEndpoint(nodeID string, clusterName string, endpointConfig *endpointv3.ClusterLoadAssignment) error {
	// Set the cluster name
	endpointConfig.ClusterName = clusterName

	// Get existing snapshot or create empty one
	snapshot, err := cm.GetSnapshot(nodeID)
	if err != nil {
		// Create new snapshot if it doesn't exist
		snapshot, err = cm.CreateEmptySnapshot(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	// Collect existing resources
	resources := make(map[resourcev3.Type][]types.Resource)

	// Copy existing endpoints and add the new one
	existingEndpoints := snapshot.GetResources(resourcev3.EndpointType)
	endpointResources := make([]types.Resource, 0, len(existingEndpoints)+1)
	for _, res := range existingEndpoints {
		endpointResources = append(endpointResources, res)
	}
	endpointResources = append(endpointResources, endpointConfig)
	resources[resourcev3.EndpointType] = endpointResources

	// Copy other existing resources
	resources[resourcev3.ClusterType] = convertResourceMap(snapshot.GetResources(resourcev3.ClusterType))
	resources[resourcev3.ListenerType] = convertResourceMap(snapshot.GetResources(resourcev3.ListenerType))
	resources[resourcev3.RouteType] = convertResourceMap(snapshot.GetResources(resourcev3.RouteType))

	// Create new snapshot with incremented version
	newVersion := fmt.Sprintf("v%d", len(endpointResources))
	newSnapshot, err := cachev3.NewSnapshot(newVersion, resources)
	if err != nil {
		return fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Update the snapshot
	return cm.UpdateSnapshot(nodeID, newSnapshot)
}

// AddListener adds a listener configuration to the snapshot
func (cm *ConfigManager) AddListener(nodeID string, listenerName string, listenerConfig *listenerv3.Listener) error {
	// Set the listener name
	listenerConfig.Name = listenerName

	// Get existing snapshot or create empty one
	snapshot, err := cm.GetSnapshot(nodeID)
	if err != nil {
		// Create new snapshot if it doesn't exist
		snapshot, err = cm.CreateEmptySnapshot(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	// Collect existing resources
	resources := make(map[resourcev3.Type][]types.Resource)

	// Copy existing listeners and add the new one
	existingListeners := snapshot.GetResources(resourcev3.ListenerType)
	listenerResources := make([]types.Resource, 0, len(existingListeners)+1)
	for _, res := range existingListeners {
		listenerResources = append(listenerResources, res)
	}
	listenerResources = append(listenerResources, listenerConfig)
	resources[resourcev3.ListenerType] = listenerResources

	// Copy other existing resources
	resources[resourcev3.ClusterType] = convertResourceMap(snapshot.GetResources(resourcev3.ClusterType))
	resources[resourcev3.EndpointType] = convertResourceMap(snapshot.GetResources(resourcev3.EndpointType))
	resources[resourcev3.RouteType] = convertResourceMap(snapshot.GetResources(resourcev3.RouteType))

	// Create new snapshot with incremented version
	newVersion := fmt.Sprintf("v%d", len(listenerResources))
	newSnapshot, err := cachev3.NewSnapshot(newVersion, resources)
	if err != nil {
		return fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Update the snapshot
	return cm.UpdateSnapshot(nodeID, newSnapshot)
}

// AddRoute adds a route configuration to the snapshot
func (cm *ConfigManager) AddRoute(nodeID string, routeName string, routeConfig *routev3.RouteConfiguration) error {
	// Set the route name
	routeConfig.Name = routeName

	// Get existing snapshot or create empty one
	snapshot, err := cm.GetSnapshot(nodeID)
	if err != nil {
		// Create new snapshot if it doesn't exist
		snapshot, err = cm.CreateEmptySnapshot(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	// Collect existing resources
	resources := make(map[resourcev3.Type][]types.Resource)

	// Copy existing routes and add the new one
	existingRoutes := snapshot.GetResources(resourcev3.RouteType)
	routeResources := make([]types.Resource, 0, len(existingRoutes)+1)
	for _, res := range existingRoutes {
		routeResources = append(routeResources, res)
	}
	routeResources = append(routeResources, routeConfig)
	resources[resourcev3.RouteType] = routeResources

	// Copy other existing resources
	resources[resourcev3.ClusterType] = convertResourceMap(snapshot.GetResources(resourcev3.ClusterType))
	resources[resourcev3.EndpointType] = convertResourceMap(snapshot.GetResources(resourcev3.EndpointType))
	resources[resourcev3.ListenerType] = convertResourceMap(snapshot.GetResources(resourcev3.ListenerType))

	// Create new snapshot with incremented version
	newVersion := fmt.Sprintf("v%d", len(routeResources))
	newSnapshot, err := cachev3.NewSnapshot(newVersion, resources)
	if err != nil {
		return fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Update the snapshot
	return cm.UpdateSnapshot(nodeID, newSnapshot)
}

// RemoveNode removes all configuration for a given node ID
func (cm *ConfigManager) RemoveNode(nodeID string) {
	cm.cache.ClearSnapshot(nodeID)
	cm.logger.Infof("Removed configuration for node %s", nodeID)
}

// ListNodes returns a list of all configured node IDs
func (cm *ConfigManager) ListNodes() []string {
	return cm.cache.GetStatusKeys()
}
