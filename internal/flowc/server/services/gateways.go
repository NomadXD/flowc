package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/server/repository"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	listenerresource "github.com/flowc-labs/flowc/internal/flowc/xds/resources/listener"
	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/google/uuid"
)

// GatewayService manages gateway lifecycle including registration, updates, and deletion.
// It coordinates between the repository layer for persistence and the xDS cache for
// Envoy configuration management.
type GatewayService struct {
	configManager *cache.ConfigManager
	logger        *logger.EnvoyLogger
	repo          repository.Repository
}

// NewGatewayService creates a new gateway service with a default in-memory repository.
func NewGatewayService(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *GatewayService {
	return NewGatewayServiceWithRepository(configManager, logger, repository.NewDefaultRepository())
}

// NewGatewayServiceWithRepository creates a gateway service with a custom repository.
// This is useful for testing or when using a different storage backend.
func NewGatewayServiceWithRepository(configManager *cache.ConfigManager, logger *logger.EnvoyLogger, repo repository.Repository) *GatewayService {
	return &GatewayService{
		configManager: configManager,
		logger:        logger,
		repo:          repo,
	}
}

// CreateGateway registers a new gateway in the control plane with listeners and environments.
// It validates the entire request (including nested listeners and environments), creates all entities atomically,
// and initializes the xDS snapshot with all resources. If any step fails, all created entities are rolled back.
//
// Behavior:
//   - If req.Listeners is empty/nil, a default listener on port 10000 with a "production" environment is created
//   - If listeners are provided but have no environments, a "production" environment is auto-created for each
//   - All operations are atomic - failure at any point triggers complete rollback
func (s *GatewayService) CreateGateway(ctx context.Context, req *models.CreateGatewayRequest) (*models.Gateway, error) {
	// Validate basic required fields
	if req.NodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Validate the entire request including nested entities
	if err := s.validateGatewayRequest(ctx, req); err != nil {
		return nil, err
	}

	// Create gateway entity
	gateway := &models.Gateway{
		ID:          uuid.New().String(),
		NodeID:      req.NodeID,
		Name:        req.Name,
		Description: req.Description,
		Status:      models.GatewayStatusUnknown,
		Defaults:    req.Defaults,
		Labels:      req.Labels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store gateway in repository
	if err := s.repo.CreateGateway(ctx, gateway); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, fmt.Errorf("gateway with node_id '%s' already exists", req.NodeID)
		}
		return nil, fmt.Errorf("failed to create gateway: %w", err)
	}

	// Determine listener configurations (use defaults if empty)
	listenerConfigs := req.Listeners
	if len(listenerConfigs) == 0 {
		// Create default listener configuration
		listenerConfigs = []models.ListenerConfig{
			{
				Port:    s.getDefaultListenerPort(),
				Address: DefaultListenerAddress,
				Environments: []models.EnvironmentConfig{
					{
						Name:     DefaultEnvironmentName,
						Hostname: DefaultEnvironmentHostname,
					},
				},
			},
		}
		s.logger.WithField("gatewayID", gateway.ID).Debug("No listeners provided, using default configuration")
	}

	// Track created resources for potential rollback
	createdListeners := make([]*models.Listener, 0, len(listenerConfigs))
	createdEnvironments := make(map[string][]*models.GatewayEnvironment) // listenerID -> environments

	// Create all listeners and their environments
	for _, lc := range listenerConfigs {
		// Ensure each listener has at least one environment
		envConfigs := lc.Environments
		if len(envConfigs) == 0 {
			envConfigs = []models.EnvironmentConfig{
				{
					Name:     DefaultEnvironmentName,
					Hostname: DefaultEnvironmentHostname,
				},
			}
			s.logger.WithFields(map[string]interface{}{
				"gatewayID": gateway.ID,
				"port":      lc.Port,
			}).Debug("No environments provided for listener, using default")
		}

		// Create listener with environments
		listener, environments, err := s.createListenerWithEnvironments(ctx, gateway, &lc, envConfigs)
		if err != nil {
			// Rollback all created resources
			s.rollbackGatewayCreation(ctx, gateway, createdListeners, createdEnvironments)
			return nil, fmt.Errorf("failed to create listener on port %d: %w", lc.Port, err)
		}

		createdListeners = append(createdListeners, listener)
		createdEnvironments[listener.ID] = environments
	}

	// Initialize xDS snapshot - all listeners/environments are created, now update xDS atomically
	// Note: updateListenerSnapshot is called per listener in createListenerWithEnvironments,
	// but we could also do a single BulkUpdate here for all resources.
	// For now, the per-listener updates are sufficient.

	// Calculate total environments for logging
	totalEnvs := 0
	for _, envs := range createdEnvironments {
		totalEnvs += len(envs)
	}

	s.logger.WithFields(map[string]interface{}{
		"gatewayID":         gateway.ID,
		"nodeID":            gateway.NodeID,
		"name":              gateway.Name,
		"listeners":         len(createdListeners),
		"totalEnvironments": totalEnvs,
	}).Info("Gateway created successfully with listeners and environments")

	return gateway, nil
}

// GetGateway retrieves a gateway by its ID.
func (s *GatewayService) GetGateway(ctx context.Context, id string) (*models.Gateway, error) {
	gateway, err := s.repo.GetGateway(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("gateway not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}
	return gateway, nil
}

// GetGatewayByNodeID retrieves a gateway by its Envoy node ID.
func (s *GatewayService) GetGatewayByNodeID(ctx context.Context, nodeID string) (*models.Gateway, error) {
	gateway, err := s.repo.GetGatewayByNodeID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("gateway with node_id '%s' not found", nodeID)
		}
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}
	return gateway, nil
}

// ListGateways retrieves all registered gateways.
func (s *GatewayService) ListGateways(ctx context.Context) ([]*models.Gateway, error) {
	gateways, err := s.repo.ListGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateways: %w", err)
	}
	return gateways, nil
}

// UpdateGateway updates an existing gateway's configuration.
// Only the fields provided in the request will be updated.
func (s *GatewayService) UpdateGateway(ctx context.Context, id string, req *models.UpdateGatewayRequest) (*models.Gateway, error) {
	gateway, err := s.repo.GetGateway(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("gateway not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}

	// Apply partial updates - only update fields that are provided
	if req.Name != nil {
		gateway.Name = *req.Name
	}
	if req.Description != nil {
		gateway.Description = *req.Description
	}
	if req.Defaults != nil {
		gateway.Defaults = req.Defaults
	}
	if req.Labels != nil {
		gateway.Labels = req.Labels
	}
	gateway.UpdatedAt = time.Now()

	if err := s.repo.UpdateGateway(ctx, gateway); err != nil {
		return nil, fmt.Errorf("failed to update gateway: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"gatewayID": gateway.ID,
		"nodeID":    gateway.NodeID,
	}).Info("Gateway updated successfully")

	return gateway, nil
}

// DeleteGateway removes a gateway from the control plane.
// If force is false and the gateway has active listeners, the deletion will fail.
// If force is true, all listeners, environments, and deployments will be removed before deleting the gateway.
func (s *GatewayService) DeleteGateway(ctx context.Context, id string, force bool) error {
	gateway, err := s.repo.GetGateway(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("gateway not found: %s", id)
		}
		return fmt.Errorf("failed to get gateway: %w", err)
	}

	// Check for listeners on this gateway
	listeners, err := s.repo.ListListenersByGateway(ctx, gateway.ID)
	if err != nil {
		return fmt.Errorf("failed to check listeners: %w", err)
	}

	if len(listeners) > 0 {
		if !force {
			return fmt.Errorf("gateway has %d active listener(s); use force=true to delete anyway", len(listeners))
		}

		// Force delete: cascade delete all listeners, environments, and deployments
		s.logger.WithFields(map[string]interface{}{
			"gatewayID": id,
			"nodeID":    gateway.NodeID,
			"listeners": len(listeners),
		}).Warn("Force deleting gateway with active listeners")

		for _, listener := range listeners {
			if err := s.cascadeDeleteListener(ctx, listener.ID); err != nil {
				s.logger.WithError(err).WithField("listenerID", listener.ID).Error("Failed to cascade delete listener during gateway cleanup")
			}
		}
	}

	// Remove xDS snapshot for this gateway
	s.configManager.RemoveNode(gateway.NodeID)

	// Delete gateway from repository
	if err := s.repo.DeleteGateway(ctx, id); err != nil {
		return fmt.Errorf("failed to delete gateway: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"gatewayID": id,
		"nodeID":    gateway.NodeID,
		"force":     force,
	}).Info("Gateway deleted successfully")

	return nil
}

// cascadeDeleteListener deletes a listener and all its child environments and deployments.
func (s *GatewayService) cascadeDeleteListener(ctx context.Context, listenerID string) error {
	// Get all environments for this listener
	environments, err := s.repo.ListEnvironmentsByListener(ctx, listenerID)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	// Delete all environments and their deployments
	for _, env := range environments {
		if err := s.cascadeDeleteEnvironment(ctx, env.ID); err != nil {
			s.logger.WithError(err).WithField("environmentID", env.ID).Error("Failed to cascade delete environment during listener cleanup")
		}
	}

	// Delete the listener
	if err := s.repo.DeleteListener(ctx, listenerID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("failed to delete listener: %w", err)
	}

	return nil
}

// cascadeDeleteEnvironment deletes an environment and all its deployments.
func (s *GatewayService) cascadeDeleteEnvironment(ctx context.Context, environmentID string) error {
	// Get all deployments for this environment
	deploymentIDs, err := s.repo.GetDeploymentsByEnvironment(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("failed to get deployments: %w", err)
	}

	// Delete all deployments
	for _, depID := range deploymentIDs {
		if err := s.repo.Delete(ctx, depID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			s.logger.WithError(err).WithField("deploymentID", depID).Error("Failed to delete deployment during environment cleanup")
		}
		if err := s.repo.DeleteNodeID(ctx, depID); err != nil {
			s.logger.WithError(err).WithField("deploymentID", depID).Error("Failed to delete node mapping during environment cleanup")
		}
		if err := s.repo.DeleteEnvironmentID(ctx, depID); err != nil {
			s.logger.WithError(err).WithField("deploymentID", depID).Error("Failed to delete environment mapping during environment cleanup")
		}
	}

	// Delete the environment
	if err := s.repo.DeleteEnvironment(ctx, environmentID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	return nil
}

// GetGatewayAPIs retrieves all API deployments associated with a gateway.
func (s *GatewayService) GetGatewayAPIs(ctx context.Context, id string) ([]*models.APIDeployment, error) {
	gateway, err := s.repo.GetGateway(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("gateway not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}

	deploymentIDs, err := s.repo.GetDeploymentsByNodeID(ctx, gateway.NodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	deployments := make([]*models.APIDeployment, 0, len(deploymentIDs))
	for _, depID := range deploymentIDs {
		deployment, err := s.repo.Get(ctx, depID)
		if err != nil {
			// Skip deployments that can't be found (may have been deleted)
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			s.logger.WithError(err).WithField("deploymentID", depID).Warn("Failed to retrieve deployment")
			continue
		}
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// CountGatewayAPIs returns the number of API deployments on a gateway.
func (s *GatewayService) CountGatewayAPIs(ctx context.Context, id string) (int, error) {
	gateway, err := s.repo.GetGateway(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, fmt.Errorf("gateway not found: %s", id)
		}
		return 0, fmt.Errorf("failed to get gateway: %w", err)
	}

	deploymentIDs, err := s.repo.GetDeploymentsByNodeID(ctx, gateway.NodeID)
	if err != nil {
		return 0, fmt.Errorf("failed to get deployments: %w", err)
	}

	return len(deploymentIDs), nil
}

// validateGatewayRequest validates the entire gateway creation request including nested entities.
// It checks for duplicate ports, environment names, and hostnames within the request.
func (s *GatewayService) validateGatewayRequest(ctx context.Context, req *models.CreateGatewayRequest) error {
	// Check node ID uniqueness
	exists, err := s.repo.GatewayExists(ctx, req.NodeID)
	if err != nil {
		return fmt.Errorf("failed to check gateway existence: %w", err)
	}
	if exists {
		return fmt.Errorf("gateway with node_id '%s' already exists", req.NodeID)
	}

	// Validate listeners if provided
	if len(req.Listeners) > 0 {
		// Check for duplicate ports within request
		portMap := make(map[uint32]bool)
		for _, lc := range req.Listeners {
			if lc.Port == 0 {
				return fmt.Errorf("listener port is required")
			}
			if portMap[lc.Port] {
				return fmt.Errorf("duplicate port %d in listeners", lc.Port)
			}
			portMap[lc.Port] = true

			// Validate environments for this listener
			if len(lc.Environments) > 0 {
				nameMap := make(map[string]bool)
				hostnameMap := make(map[string]bool)

				for _, ec := range lc.Environments {
					if ec.Name == "" {
						return fmt.Errorf("environment name is required")
					}
					if ec.Hostname == "" {
						return fmt.Errorf("environment hostname is required")
					}

					// Check for duplicate names within this listener
					if nameMap[ec.Name] {
						return fmt.Errorf("duplicate environment name '%s' in listener on port %d", ec.Name, lc.Port)
					}
					nameMap[ec.Name] = true

					// Check for duplicate hostnames within this listener
					if hostnameMap[ec.Hostname] {
						return fmt.Errorf("duplicate hostname '%s' in listener on port %d", ec.Hostname, lc.Port)
					}
					hostnameMap[ec.Hostname] = true
				}
			}
		}
	}

	return nil
}

// createListenerWithEnvironments creates a listener and its environments atomically.
// It creates the listener entity, all environment entities, and updates the xDS snapshot.
// Returns the created listener and environments, or an error if any step fails.
func (s *GatewayService) createListenerWithEnvironments(
	ctx context.Context,
	gateway *models.Gateway,
	lc *models.ListenerConfig,
	envConfigs []models.EnvironmentConfig,
) (*models.Listener, []*models.GatewayEnvironment, error) {
	// Create listener entity
	listener := &models.Listener{
		ID:        uuid.New().String(),
		GatewayID: gateway.ID,
		Port:      lc.Port,
		Address:   lc.Address,
		TLS:       lc.TLS,
		HTTP2:     lc.HTTP2,
		AccessLog: lc.AccessLog,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set default address if not provided
	if listener.Address == "" {
		listener.Address = DefaultListenerAddress
	}

	// Check port uniqueness within gateway
	exists, err := s.repo.ListenerExists(ctx, gateway.ID, lc.Port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check listener existence: %w", err)
	}
	if exists {
		return nil, nil, fmt.Errorf("port %d is already in use on this gateway", lc.Port)
	}

	// Store listener in repository
	if err := s.repo.CreateListener(ctx, listener); err != nil {
		return nil, nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Create all environments for this listener
	environments := make([]*models.GatewayEnvironment, 0, len(envConfigs))
	for _, ec := range envConfigs {
		// Check name uniqueness within listener
		nameExists, err := s.repo.EnvironmentExists(ctx, listener.ID, ec.Name)
		if err != nil {
			// Rollback listener
			_ = s.repo.DeleteListener(ctx, listener.ID)
			return nil, nil, fmt.Errorf("failed to check environment name uniqueness: %w", err)
		}
		if nameExists {
			// Rollback listener
			_ = s.repo.DeleteListener(ctx, listener.ID)
			return nil, nil, fmt.Errorf("environment name '%s' already exists on this listener", ec.Name)
		}

		// Check hostname uniqueness within listener
		hostnameExists, err := s.repo.HostnameExistsOnListener(ctx, listener.ID, ec.Hostname)
		if err != nil {
			// Rollback listener and created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			_ = s.repo.DeleteListener(ctx, listener.ID)
			return nil, nil, fmt.Errorf("failed to check hostname uniqueness: %w", err)
		}
		if hostnameExists {
			// Rollback listener and created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			_ = s.repo.DeleteListener(ctx, listener.ID)
			return nil, nil, fmt.Errorf("hostname '%s' already exists on this listener", ec.Hostname)
		}

		environment := &models.GatewayEnvironment{
			ID:          uuid.New().String(),
			ListenerID:  listener.ID,
			Name:        ec.Name,
			Hostname:    ec.Hostname,
			Description: ec.Description,
			HTTPFilters: ec.HTTPFilters,
			Labels:      ec.Labels,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := s.repo.CreateEnvironment(ctx, environment); err != nil {
			// Rollback listener and all created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			_ = s.repo.DeleteListener(ctx, listener.ID)
			return nil, nil, fmt.Errorf("failed to create environment '%s': %w", ec.Name, err)
		}

		environments = append(environments, environment)
	}

	// Update xDS snapshot with the new listener and its environments
	// This delegates to the ListenerService's updateListenerSnapshot logic
	if err := s.updateListenerSnapshot(ctx, gateway.NodeID, listener); err != nil {
		// Rollback all created entities
		for _, env := range environments {
			_ = s.repo.DeleteEnvironment(ctx, env.ID)
		}
		_ = s.repo.DeleteListener(ctx, listener.ID)
		return nil, nil, fmt.Errorf("failed to update xDS snapshot: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"listenerID":   listener.ID,
		"gatewayID":    gateway.ID,
		"port":         listener.Port,
		"environments": len(environments),
	}).Debug("Listener created with environments")

	return listener, environments, nil
}

// updateListenerSnapshot generates and updates the xDS listener resource for a listener.
// This creates a listener with filter chains for all environments on the listener,
// and creates empty route configurations for each environment.
func (s *GatewayService) updateListenerSnapshot(ctx context.Context, nodeID string, listener *models.Listener) error {
	// Get all environments for this listener
	environments, err := s.repo.ListEnvironmentsByListener(ctx, listener.ID)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	// Build filter chain configurations and route configs for each environment
	filterChains := make([]*listenerresource.FilterChainConfig, 0, len(environments))
	routeConfigs := make([]*routev3.RouteConfiguration, 0, len(environments))

	for _, env := range environments {
		// Create route config name for this environment
		routeConfigName := fmt.Sprintf("route_%s_%s", listener.ID, env.Name)

		filterChains = append(filterChains, &listenerresource.FilterChainConfig{
			Name:            env.Name,
			Hostname:        env.Hostname,
			HTTPFilters:     env.HTTPFilters,
			RouteConfigName: routeConfigName,
			// TODO: Map models.TLSConfig to listenerresource.TLSConfig if listener has TLS
		})

		// Create an empty route configuration for this environment
		// Deployments will add actual routes to this later
		routeConfig := &routev3.RouteConfiguration{
			Name: routeConfigName,
			VirtualHosts: []*routev3.VirtualHost{
				{
					Name:    fmt.Sprintf("vh_%s_%s", listener.ID, env.Name),
					Domains: []string{"*"},      // Match all domains for now
					Routes:  []*routev3.Route{}, // Empty routes - deployments will populate this
				},
			},
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	// Create listener configuration
	listenerConfig := &listenerresource.ListenerConfig{
		Name:         fmt.Sprintf("listener_%d", listener.Port),
		Port:         listener.Port,
		Address:      listener.Address,
		FilterChains: filterChains,
		HTTP2:        listener.HTTP2,
		AccessLog:    listener.AccessLog,
	}

	// Generate xDS listener resource
	xdsListener, err := listenerresource.CreateListenerWithFilterChains(listenerConfig)
	if err != nil {
		return fmt.Errorf("failed to create xDS listener: %w", err)
	}

	// Use BulkUpdate to atomically add the listener and all route configurations
	if err := s.configManager.BulkUpdate(nodeID, &cache.BulkResourceUpdate{
		AddListeners: []*listenerv3.Listener{xdsListener},
		AddRoutes:    routeConfigs,
	}); err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"nodeID":       nodeID,
		"listenerID":   listener.ID,
		"port":         listener.Port,
		"environments": len(environments),
		"routes":       len(routeConfigs),
	}).Info("Updated xDS listener snapshot with route configurations")

	return nil
}

// rollbackGatewayCreation rolls back all created entities in reverse order.
// This is a best-effort rollback - it logs errors but continues with the rollback process.
func (s *GatewayService) rollbackGatewayCreation(
	ctx context.Context,
	gateway *models.Gateway,
	listeners []*models.Listener,
	environments map[string][]*models.GatewayEnvironment,
) {
	s.logger.WithField("gatewayID", gateway.ID).Warn("Rolling back gateway creation")

	rollbackErrs := make([]error, 0)

	// 1. Remove xDS snapshot
	s.configManager.RemoveNode(gateway.NodeID)

	// 2. Delete all environments
	for _, envs := range environments {
		for _, env := range envs {
			if err := s.repo.DeleteEnvironment(ctx, env.ID); err != nil && !errors.Is(err, repository.ErrNotFound) {
				rollbackErrs = append(rollbackErrs, fmt.Errorf("failed to delete environment %s: %w", env.ID, err))
				s.logger.WithError(err).WithField("environmentID", env.ID).Error("Failed to delete environment during rollback")
			}
		}
	}

	// 3. Delete all listeners
	for _, listener := range listeners {
		if err := s.repo.DeleteListener(ctx, listener.ID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("failed to delete listener %s: %w", listener.ID, err))
			s.logger.WithError(err).WithField("listenerID", listener.ID).Error("Failed to delete listener during rollback")
		}
	}

	// 4. Delete gateway
	if err := s.repo.DeleteGateway(ctx, gateway.ID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		rollbackErrs = append(rollbackErrs, fmt.Errorf("failed to delete gateway %s: %w", gateway.ID, err))
		s.logger.WithError(err).WithField("gatewayID", gateway.ID).Error("Failed to delete gateway during rollback")
	}

	if len(rollbackErrs) > 0 {
		s.logger.WithFields(map[string]interface{}{
			"gatewayID": gateway.ID,
			"errors":    len(rollbackErrs),
		}).Error("Rollback completed with errors")
	} else {
		s.logger.WithField("gatewayID", gateway.ID).Info("Rollback completed successfully")
	}
}

// getDefaultListenerPort returns the default listener port.
// This uses the constant defined in constants.go.
// In the future, this could be enhanced to read from config if needed.
func (s *GatewayService) getDefaultListenerPort() uint32 {
	return DefaultListenerPort
}
