package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/pkg/types"
)

// MemoryRepository is an in-memory implementation of the Repository interface.
// It uses sync.RWMutex for thread-safe operations.
// This implementation is suitable for development, testing, and single-instance deployments.
type MemoryRepository struct {
	// Deployment storage
	deployments map[string]*models.APIDeployment
	nodeIDs     map[string]string // Maps deployment ID to node ID

	// Gateway storage
	gateways        map[string]*models.Gateway // Maps gateway ID to gateway
	gatewayByNodeID map[string]string          // Maps node ID to gateway ID for fast lookup

	// Listener storage
	listeners             map[string]*models.Listener // Maps listener ID to listener
	listenerByGatewayPort map[string]string           // Maps "gatewayID:port" to listener ID

	// Environment storage
	environments      map[string]*models.GatewayEnvironment // Maps environment ID to environment
	envByListenerName map[string]string                     // Maps "listenerID:name" to environment ID
	envByListenerHost map[string]string                     // Maps "listenerID:hostname" to environment ID
	environmentIDs    map[string]string                     // Maps deployment ID to environment ID

	mutex sync.RWMutex
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		deployments:           make(map[string]*models.APIDeployment),
		nodeIDs:               make(map[string]string),
		gateways:              make(map[string]*models.Gateway),
		gatewayByNodeID:       make(map[string]string),
		listeners:             make(map[string]*models.Listener),
		listenerByGatewayPort: make(map[string]string),
		environments:          make(map[string]*models.GatewayEnvironment),
		envByListenerName:     make(map[string]string),
		envByListenerHost:     make(map[string]string),
		environmentIDs:        make(map[string]string),
	}
}

// Ensure MemoryRepository implements Repository interface.
var _ Repository = (*MemoryRepository)(nil)

// Create stores a new deployment.
func (r *MemoryRepository) Create(ctx context.Context, deployment *models.APIDeployment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deployment == nil {
		return ErrInvalidInput
	}

	if deployment.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[deployment.ID]; exists {
		return ErrAlreadyExists
	}

	// Store a copy to prevent external modifications
	r.deployments[deployment.ID] = copyDeployment(deployment)
	return nil
}

// Get retrieves a deployment by ID.
func (r *MemoryRepository) Get(ctx context.Context, id string) (*models.APIDeployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployment, exists := r.deployments[id]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modifications
	return copyDeployment(deployment), nil
}

// Update modifies an existing deployment.
func (r *MemoryRepository) Update(ctx context.Context, deployment *models.APIDeployment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deployment == nil || deployment.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[deployment.ID]; !exists {
		return ErrNotFound
	}

	r.deployments[deployment.ID] = copyDeployment(deployment)
	return nil
}

// Delete removes a deployment by ID.
func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[id]; !exists {
		return ErrNotFound
	}

	delete(r.deployments, id)
	return nil
}

// List retrieves all deployments.
func (r *MemoryRepository) List(ctx context.Context) ([]*models.APIDeployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployments := make([]*models.APIDeployment, 0, len(r.deployments))
	for _, deployment := range r.deployments {
		deployments = append(deployments, copyDeployment(deployment))
	}

	return deployments, nil
}

// ListByStatus retrieves deployments filtered by status.
func (r *MemoryRepository) ListByStatus(ctx context.Context, status models.DeploymentStatus) ([]*models.APIDeployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployments := make([]*models.APIDeployment, 0)
	for _, deployment := range r.deployments {
		if deployment.Status == string(status) {
			deployments = append(deployments, copyDeployment(deployment))
		}
	}

	return deployments, nil
}

// Count returns the total number of deployments.
func (r *MemoryRepository) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.deployments), nil
}

// Exists checks if a deployment with the given ID exists.
func (r *MemoryRepository) Exists(ctx context.Context, id string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.deployments[id]
	return exists, nil
}

// SetNodeID associates a node ID with a deployment.
func (r *MemoryRepository) SetNodeID(ctx context.Context, deploymentID, nodeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deploymentID == "" || nodeID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.nodeIDs[deploymentID] = nodeID
	return nil
}

// GetNodeID retrieves the node ID for a deployment.
func (r *MemoryRepository) GetNodeID(ctx context.Context, deploymentID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	nodeID, exists := r.nodeIDs[deploymentID]
	if !exists {
		return "", ErrNotFound
	}

	return nodeID, nil
}

// DeleteNodeID removes the node ID mapping for a deployment.
func (r *MemoryRepository) DeleteNodeID(ctx context.Context, deploymentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.nodeIDs, deploymentID)
	return nil
}

// GetDeploymentsByNodeID retrieves all deployment IDs associated with a node ID.
func (r *MemoryRepository) GetDeploymentsByNodeID(ctx context.Context, nodeID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deploymentIDs := make([]string, 0)
	for depID, nID := range r.nodeIDs {
		if nID == nodeID {
			deploymentIDs = append(deploymentIDs, depID)
		}
	}

	return deploymentIDs, nil
}

// GetStats retrieves deployment statistics.
func (r *MemoryRepository) GetStats(ctx context.Context) (*DeploymentStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := &DeploymentStats{
		Total: len(r.deployments),
	}

	for _, deployment := range r.deployments {
		switch deployment.Status {
		case string(models.StatusDeployed):
			stats.Deployed++
		case string(models.StatusFailed):
			stats.Failed++
		case string(models.StatusPending):
			stats.Pending++
		case string(models.StatusUpdating):
			stats.Updating++
		case string(models.StatusDeploying):
			stats.Deploying++
		}
	}

	return stats, nil
}

// Close releases any resources held by the repository.
// For the in-memory implementation, this is a no-op.
func (r *MemoryRepository) Close() error {
	return nil
}

// Ping checks if the underlying storage is accessible.
// For the in-memory implementation, this always succeeds.
func (r *MemoryRepository) Ping(ctx context.Context) error {
	return ctx.Err()
}

// copyDeployment creates a shallow copy of a deployment.
// Note: This does not deep copy nested structures like Metadata.
// For production use with mutations, consider implementing deep copy.
func copyDeployment(d *models.APIDeployment) *models.APIDeployment {
	if d == nil {
		return nil
	}

	return &models.APIDeployment{
		ID:        d.ID,
		Name:      d.Name,
		Version:   d.Version,
		Context:   d.Context,
		Status:    d.Status,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		Metadata:  d.Metadata,
	}
}

// ============================================================================
// GatewayRepository Implementation
// ============================================================================

// CreateGateway stores a new gateway.
func (r *MemoryRepository) CreateGateway(ctx context.Context, gateway *models.Gateway) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if gateway == nil {
		return ErrInvalidInput
	}

	if gateway.ID == "" || gateway.NodeID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if gateway with same ID already exists
	if _, exists := r.gateways[gateway.ID]; exists {
		return ErrAlreadyExists
	}

	// Check if gateway with same NodeID already exists
	if _, exists := r.gatewayByNodeID[gateway.NodeID]; exists {
		return ErrAlreadyExists
	}

	// Store a copy to prevent external modifications
	r.gateways[gateway.ID] = copyGateway(gateway)
	r.gatewayByNodeID[gateway.NodeID] = gateway.ID
	return nil
}

// GetGateway retrieves a gateway by ID.
func (r *MemoryRepository) GetGateway(ctx context.Context, id string) (*models.Gateway, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	gateway, exists := r.gateways[id]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modifications
	return copyGateway(gateway), nil
}

// GetGatewayByNodeID retrieves a gateway by its node ID.
func (r *MemoryRepository) GetGatewayByNodeID(ctx context.Context, nodeID string) (*models.Gateway, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	gatewayID, exists := r.gatewayByNodeID[nodeID]
	if !exists {
		return nil, ErrNotFound
	}

	gateway, exists := r.gateways[gatewayID]
	if !exists {
		return nil, ErrNotFound
	}

	return copyGateway(gateway), nil
}

// UpdateGateway modifies an existing gateway.
func (r *MemoryRepository) UpdateGateway(ctx context.Context, gateway *models.Gateway) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if gateway == nil || gateway.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.gateways[gateway.ID]
	if !exists {
		return ErrNotFound
	}

	// If NodeID is being changed, update the lookup map
	if existing.NodeID != gateway.NodeID {
		// Check if new NodeID already exists
		if _, exists := r.gatewayByNodeID[gateway.NodeID]; exists {
			return ErrAlreadyExists
		}
		delete(r.gatewayByNodeID, existing.NodeID)
		r.gatewayByNodeID[gateway.NodeID] = gateway.ID
	}

	r.gateways[gateway.ID] = copyGateway(gateway)
	return nil
}

// DeleteGateway removes a gateway by ID.
func (r *MemoryRepository) DeleteGateway(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	gateway, exists := r.gateways[id]
	if !exists {
		return ErrNotFound
	}

	delete(r.gatewayByNodeID, gateway.NodeID)
	delete(r.gateways, id)
	return nil
}

// ListGateways retrieves all gateways.
func (r *MemoryRepository) ListGateways(ctx context.Context) ([]*models.Gateway, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	gateways := make([]*models.Gateway, 0, len(r.gateways))
	for _, gateway := range r.gateways {
		gateways = append(gateways, copyGateway(gateway))
	}

	return gateways, nil
}

// GatewayExists checks if a gateway with the given node ID exists.
func (r *MemoryRepository) GatewayExists(ctx context.Context, nodeID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.gatewayByNodeID[nodeID]
	return exists, nil
}

// CountGateways returns the total number of gateways.
func (r *MemoryRepository) CountGateways(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.gateways), nil
}

// copyGateway creates a shallow copy of a gateway.
// Note: This does not deep copy nested structures like Defaults, Labels.
// For production use with mutations, consider implementing deep copy.
func copyGateway(g *models.Gateway) *models.Gateway {
	if g == nil {
		return nil
	}

	gw := &models.Gateway{
		ID:          g.ID,
		NodeID:      g.NodeID,
		Name:        g.Name,
		Description: g.Description,
		Status:      g.Status,
		Defaults:    g.Defaults,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}

	// Copy labels map
	if g.Labels != nil {
		gw.Labels = make(map[string]string, len(g.Labels))
		for k, v := range g.Labels {
			gw.Labels[k] = v
		}
	}

	return gw
}

// ============================================================================
// ListenerRepository Implementation
// ============================================================================

// listenerKey generates the key for gateway:port lookup
func listenerKey(gatewayID string, port uint32) string {
	return fmt.Sprintf("%s:%d", gatewayID, port)
}

// CreateListener stores a new listener.
func (r *MemoryRepository) CreateListener(ctx context.Context, listener *models.Listener) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if listener == nil {
		return ErrInvalidInput
	}

	if listener.ID == "" || listener.GatewayID == "" || listener.Port == 0 {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if listener with same ID already exists
	if _, exists := r.listeners[listener.ID]; exists {
		return ErrAlreadyExists
	}

	// Check if port is already in use for this gateway
	key := listenerKey(listener.GatewayID, listener.Port)
	if _, exists := r.listenerByGatewayPort[key]; exists {
		return ErrListenerPortInUse
	}

	// Store a copy to prevent external modifications
	r.listeners[listener.ID] = copyListener(listener)
	r.listenerByGatewayPort[key] = listener.ID
	return nil
}

// GetListener retrieves a listener by ID.
func (r *MemoryRepository) GetListener(ctx context.Context, id string) (*models.Listener, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	listener, exists := r.listeners[id]
	if !exists {
		return nil, ErrNotFound
	}

	return copyListener(listener), nil
}

// GetListenerByGatewayAndPort retrieves a listener by gateway ID and port.
func (r *MemoryRepository) GetListenerByGatewayAndPort(ctx context.Context, gatewayID string, port uint32) (*models.Listener, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := listenerKey(gatewayID, port)
	listenerID, exists := r.listenerByGatewayPort[key]
	if !exists {
		return nil, ErrNotFound
	}

	listener, exists := r.listeners[listenerID]
	if !exists {
		return nil, ErrNotFound
	}

	return copyListener(listener), nil
}

// UpdateListener modifies an existing listener.
func (r *MemoryRepository) UpdateListener(ctx context.Context, listener *models.Listener) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if listener == nil || listener.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.listeners[listener.ID]
	if !exists {
		return ErrNotFound
	}

	// If port is being changed, update the lookup map
	if existing.Port != listener.Port {
		oldKey := listenerKey(existing.GatewayID, existing.Port)
		newKey := listenerKey(listener.GatewayID, listener.Port)

		// Check if new port is already in use
		if _, exists := r.listenerByGatewayPort[newKey]; exists {
			return ErrListenerPortInUse
		}

		delete(r.listenerByGatewayPort, oldKey)
		r.listenerByGatewayPort[newKey] = listener.ID
	}

	r.listeners[listener.ID] = copyListener(listener)
	return nil
}

// DeleteListener removes a listener by ID.
func (r *MemoryRepository) DeleteListener(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	listener, exists := r.listeners[id]
	if !exists {
		return ErrNotFound
	}

	key := listenerKey(listener.GatewayID, listener.Port)
	delete(r.listenerByGatewayPort, key)
	delete(r.listeners, id)
	return nil
}

// ListListenersByGateway retrieves all listeners for a gateway.
func (r *MemoryRepository) ListListenersByGateway(ctx context.Context, gatewayID string) ([]*models.Listener, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	listeners := make([]*models.Listener, 0)
	for _, listener := range r.listeners {
		if listener.GatewayID == gatewayID {
			listeners = append(listeners, copyListener(listener))
		}
	}

	return listeners, nil
}

// ListenerExists checks if a listener with the given gateway ID and port exists.
func (r *MemoryRepository) ListenerExists(ctx context.Context, gatewayID string, port uint32) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := listenerKey(gatewayID, port)
	_, exists := r.listenerByGatewayPort[key]
	return exists, nil
}

// CountListenersByGateway returns the number of listeners for a gateway.
func (r *MemoryRepository) CountListenersByGateway(ctx context.Context, gatewayID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, listener := range r.listeners {
		if listener.GatewayID == gatewayID {
			count++
		}
	}

	return count, nil
}

// copyListener creates a shallow copy of a listener.
func copyListener(l *models.Listener) *models.Listener {
	if l == nil {
		return nil
	}

	listener := &models.Listener{
		ID:        l.ID,
		GatewayID: l.GatewayID,
		Port:      l.Port,
		Address:   l.Address,
		HTTP2:     l.HTTP2,
		AccessLog: l.AccessLog,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}

	// Copy TLS config if present
	if l.TLS != nil {
		listener.TLS = &models.TLSConfig{
			CertPath:          l.TLS.CertPath,
			KeyPath:           l.TLS.KeyPath,
			CAPath:            l.TLS.CAPath,
			RequireClientCert: l.TLS.RequireClientCert,
			MinVersion:        l.TLS.MinVersion,
		}
		if l.TLS.CipherSuites != nil {
			listener.TLS.CipherSuites = make([]string, len(l.TLS.CipherSuites))
			copy(listener.TLS.CipherSuites, l.TLS.CipherSuites)
		}
	}

	return listener
}

// ============================================================================
// EnvironmentRepository Implementation
// ============================================================================

// envNameKey generates the key for listener:name lookup
func envNameKey(listenerID, name string) string {
	return listenerID + ":" + name
}

// envHostKey generates the key for listener:hostname lookup
func envHostKey(listenerID, hostname string) string {
	return listenerID + ":" + hostname
}

// CreateEnvironment stores a new environment.
func (r *MemoryRepository) CreateEnvironment(ctx context.Context, env *models.GatewayEnvironment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if env == nil {
		return ErrInvalidInput
	}

	if env.ID == "" || env.ListenerID == "" || env.Name == "" || env.Hostname == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if environment with same ID already exists
	if _, exists := r.environments[env.ID]; exists {
		return ErrAlreadyExists
	}

	// Check if name is already in use for this listener
	nameKey := envNameKey(env.ListenerID, env.Name)
	if _, exists := r.envByListenerName[nameKey]; exists {
		return ErrEnvironmentNameInUse
	}

	// Check if hostname is already in use for this listener
	hostKey := envHostKey(env.ListenerID, env.Hostname)
	if _, exists := r.envByListenerHost[hostKey]; exists {
		return ErrHostnameInUse
	}

	// Store a copy to prevent external modifications
	r.environments[env.ID] = copyEnvironment(env)
	r.envByListenerName[nameKey] = env.ID
	r.envByListenerHost[hostKey] = env.ID
	return nil
}

// GetEnvironment retrieves an environment by ID.
func (r *MemoryRepository) GetEnvironment(ctx context.Context, id string) (*models.GatewayEnvironment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	env, exists := r.environments[id]
	if !exists {
		return nil, ErrNotFound
	}

	return copyEnvironment(env), nil
}

// GetEnvironmentByListenerAndName retrieves an environment by listener ID and name.
func (r *MemoryRepository) GetEnvironmentByListenerAndName(ctx context.Context, listenerID, name string) (*models.GatewayEnvironment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	nameKey := envNameKey(listenerID, name)
	envID, exists := r.envByListenerName[nameKey]
	if !exists {
		return nil, ErrNotFound
	}

	env, exists := r.environments[envID]
	if !exists {
		return nil, ErrNotFound
	}

	return copyEnvironment(env), nil
}

// UpdateEnvironment modifies an existing environment.
func (r *MemoryRepository) UpdateEnvironment(ctx context.Context, env *models.GatewayEnvironment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if env == nil || env.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.environments[env.ID]
	if !exists {
		return ErrNotFound
	}

	// If hostname is being changed, update the lookup map
	if existing.Hostname != env.Hostname {
		oldHostKey := envHostKey(existing.ListenerID, existing.Hostname)
		newHostKey := envHostKey(env.ListenerID, env.Hostname)

		// Check if new hostname is already in use
		if _, exists := r.envByListenerHost[newHostKey]; exists {
			return ErrHostnameInUse
		}

		delete(r.envByListenerHost, oldHostKey)
		r.envByListenerHost[newHostKey] = env.ID
	}

	// Note: Name changes are not allowed (name is part of the identity)
	// If needed, you'd delete and recreate the environment

	r.environments[env.ID] = copyEnvironment(env)
	return nil
}

// DeleteEnvironment removes an environment by ID.
func (r *MemoryRepository) DeleteEnvironment(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	env, exists := r.environments[id]
	if !exists {
		return ErrNotFound
	}

	nameKey := envNameKey(env.ListenerID, env.Name)
	hostKey := envHostKey(env.ListenerID, env.Hostname)
	delete(r.envByListenerName, nameKey)
	delete(r.envByListenerHost, hostKey)
	delete(r.environments, id)
	return nil
}

// ListEnvironmentsByListener retrieves all environments for a listener.
func (r *MemoryRepository) ListEnvironmentsByListener(ctx context.Context, listenerID string) ([]*models.GatewayEnvironment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	envs := make([]*models.GatewayEnvironment, 0)
	for _, env := range r.environments {
		if env.ListenerID == listenerID {
			envs = append(envs, copyEnvironment(env))
		}
	}

	return envs, nil
}

// EnvironmentExists checks if an environment with the given listener ID and name exists.
func (r *MemoryRepository) EnvironmentExists(ctx context.Context, listenerID, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	nameKey := envNameKey(listenerID, name)
	_, exists := r.envByListenerName[nameKey]
	return exists, nil
}

// HostnameExistsOnListener checks if a hostname is already in use on a listener.
func (r *MemoryRepository) HostnameExistsOnListener(ctx context.Context, listenerID, hostname string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	hostKey := envHostKey(listenerID, hostname)
	_, exists := r.envByListenerHost[hostKey]
	return exists, nil
}

// CountEnvironmentsByListener returns the number of environments for a listener.
func (r *MemoryRepository) CountEnvironmentsByListener(ctx context.Context, listenerID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, env := range r.environments {
		if env.ListenerID == listenerID {
			count++
		}
	}

	return count, nil
}

// copyEnvironment creates a shallow copy of an environment.
func copyEnvironment(e *models.GatewayEnvironment) *models.GatewayEnvironment {
	if e == nil {
		return nil
	}

	env := &models.GatewayEnvironment{
		ID:          e.ID,
		ListenerID:  e.ListenerID,
		Name:        e.Name,
		Hostname:    e.Hostname,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}

	// Copy HTTPFilters slice
	if e.HTTPFilters != nil {
		env.HTTPFilters = make([]types.HTTPFilter, len(e.HTTPFilters))
		copy(env.HTTPFilters, e.HTTPFilters)
	}

	// Copy labels map
	if e.Labels != nil {
		env.Labels = make(map[string]string, len(e.Labels))
		for k, v := range e.Labels {
			env.Labels[k] = v
		}
	}

	return env
}

// ============================================================================
// EnvironmentMappingRepository Implementation
// ============================================================================

// SetEnvironmentID associates an environment ID with a deployment.
func (r *MemoryRepository) SetEnvironmentID(ctx context.Context, deploymentID, environmentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deploymentID == "" || environmentID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.environmentIDs[deploymentID] = environmentID
	return nil
}

// GetEnvironmentID retrieves the environment ID for a deployment.
func (r *MemoryRepository) GetEnvironmentID(ctx context.Context, deploymentID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	envID, exists := r.environmentIDs[deploymentID]
	if !exists {
		return "", ErrNotFound
	}

	return envID, nil
}

// DeleteEnvironmentID removes the environment ID mapping for a deployment.
func (r *MemoryRepository) DeleteEnvironmentID(ctx context.Context, deploymentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.environmentIDs, deploymentID)
	return nil
}

// GetDeploymentsByEnvironment retrieves all deployment IDs associated with an environment.
func (r *MemoryRepository) GetDeploymentsByEnvironment(ctx context.Context, environmentID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deploymentIDs := make([]string, 0)
	for depID, envID := range r.environmentIDs {
		if envID == environmentID {
			deploymentIDs = append(deploymentIDs, depID)
		}
	}

	return deploymentIDs, nil
}
