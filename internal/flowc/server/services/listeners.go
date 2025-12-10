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

// ListenerService manages listener lifecycle within gateways.
// Listeners are port bindings that can host multiple environments.
type ListenerService struct {
	configManager *cache.ConfigManager
	logger        *logger.EnvoyLogger
	repo          repository.Repository
}

// NewListenerService creates a new listener service with a default in-memory repository.
func NewListenerService(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *ListenerService {
	return NewListenerServiceWithRepository(configManager, logger, repository.NewDefaultRepository())
}

// NewListenerServiceWithRepository creates a listener service with a custom repository.
func NewListenerServiceWithRepository(configManager *cache.ConfigManager, logger *logger.EnvoyLogger, repo repository.Repository) *ListenerService {
	return &ListenerService{
		configManager: configManager,
		logger:        logger,
		repo:          repo,
	}
}

// CreateListener creates a new listener within a gateway with the specified environments.
// At least one environment must be provided in the request.
// All operations are atomic - failure at any point triggers complete rollback.
func (s *ListenerService) CreateListener(ctx context.Context, gatewayID string, req *models.CreateListenerRequest) (*models.Listener, error) {
	// Validate required fields
	if req.Port == 0 {
		return nil, fmt.Errorf("port is required")
	}

	// Validate that at least one environment is provided
	if len(req.Environments) == 0 {
		return nil, fmt.Errorf("at least one environment is required")
	}

	// Validate environment uniqueness within request
	if err := s.validateEnvironmentUniqueness(req.Environments); err != nil {
		return nil, err
	}

	// Validate gateway exists
	gateway, err := s.repo.GetGateway(ctx, gatewayID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("gateway not found: %s", gatewayID)
		}
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}

	// Check if port is already in use for this gateway
	exists, err := s.repo.ListenerExists(ctx, gatewayID, req.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to check listener existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("port %d is already in use on this gateway", req.Port)
	}

	// Set default address if not provided
	address := req.Address
	if address == "" {
		address = "0.0.0.0"
	}

	listener := &models.Listener{
		ID:        uuid.New().String(),
		GatewayID: gatewayID,
		Port:      req.Port,
		Address:   address,
		TLS:       req.TLS,
		HTTP2:     req.HTTP2,
		AccessLog: req.AccessLog,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store listener in repository
	if err := s.repo.CreateListener(ctx, listener); err != nil {
		if errors.Is(err, repository.ErrListenerPortInUse) {
			return nil, fmt.Errorf("port %d is already in use on this gateway", req.Port)
		}
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Create all environments for this listener
	environments, err := s.createEnvironmentsForListener(ctx, listener, req.Environments)
	if err != nil {
		// Rollback listener creation
		_ = s.repo.DeleteListener(ctx, listener.ID)
		return nil, fmt.Errorf("failed to create environments: %w", err)
	}

	// Generate xDS listener resource and update snapshot
	if err := s.updateListenerSnapshot(ctx, gateway.NodeID, listener); err != nil {
		// Rollback all created entities
		for _, env := range environments {
			_ = s.repo.DeleteEnvironment(ctx, env.ID)
		}
		_ = s.repo.DeleteListener(ctx, listener.ID)
		return nil, fmt.Errorf("failed to update xDS snapshot: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"listenerID":   listener.ID,
		"gatewayID":    gatewayID,
		"port":         listener.Port,
		"environments": len(environments),
	}).Info("Listener created successfully with environments")

	return listener, nil
}

// GetListener retrieves a listener by ID.
func (s *ListenerService) GetListener(ctx context.Context, id string) (*models.Listener, error) {
	listener, err := s.repo.GetListener(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("listener not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}
	return listener, nil
}

// GetListenerByGatewayAndPort retrieves a listener by gateway ID and port.
func (s *ListenerService) GetListenerByGatewayAndPort(ctx context.Context, gatewayID string, port uint32) (*models.Listener, error) {
	listener, err := s.repo.GetListenerByGatewayAndPort(ctx, gatewayID, port)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("listener on port %d not found on gateway %s", port, gatewayID)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}
	return listener, nil
}

// ListListenersByGateway retrieves all listeners for a gateway.
func (s *ListenerService) ListListenersByGateway(ctx context.Context, gatewayID string) ([]*models.Listener, error) {
	// Validate gateway exists
	_, err := s.repo.GetGateway(ctx, gatewayID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("gateway not found: %s", gatewayID)
		}
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}

	listeners, err := s.repo.ListListenersByGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("failed to list listeners: %w", err)
	}
	return listeners, nil
}

// UpdateListener updates an existing listener's configuration.
// Only the fields provided in the request will be updated.
// Note: Port cannot be changed after creation.
func (s *ListenerService) UpdateListener(ctx context.Context, id string, req *models.UpdateListenerRequest) (*models.Listener, error) {
	listener, err := s.repo.GetListener(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("listener not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}

	// Apply partial updates - only update fields that are provided
	if req.Address != nil {
		listener.Address = *req.Address
	}
	if req.TLS != nil {
		listener.TLS = req.TLS
	}
	if req.HTTP2 != nil {
		listener.HTTP2 = *req.HTTP2
	}
	if req.AccessLog != nil {
		listener.AccessLog = *req.AccessLog
	}
	listener.UpdatedAt = time.Now()

	if err := s.repo.UpdateListener(ctx, listener); err != nil {
		return nil, fmt.Errorf("failed to update listener: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"listenerID": listener.ID,
		"gatewayID":  listener.GatewayID,
		"port":       listener.Port,
	}).Info("Listener updated successfully")

	return listener, nil
}

// DeleteListener removes a listener from a gateway.
// If force is false and the listener has active environments, the deletion will fail.
// If force is true, all environments and their deployments will be removed first.
func (s *ListenerService) DeleteListener(ctx context.Context, id string, force bool) error {
	listener, err := s.repo.GetListener(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("listener not found: %s", id)
		}
		return fmt.Errorf("failed to get listener: %w", err)
	}

	// Check for environments on this listener
	environments, err := s.repo.ListEnvironmentsByListener(ctx, listener.ID)
	if err != nil {
		return fmt.Errorf("failed to check environments: %w", err)
	}

	if len(environments) > 0 {
		if !force {
			return fmt.Errorf("listener has %d active environment(s); use force=true to delete anyway", len(environments))
		}

		// Force delete: cascade delete all environments and their deployments
		s.logger.WithFields(map[string]interface{}{
			"listenerID":   id,
			"port":         listener.Port,
			"environments": len(environments),
		}).Warn("Force deleting listener with active environments")

		for _, env := range environments {
			if err := s.cascadeDeleteEnvironment(ctx, env.ID); err != nil {
				s.logger.WithError(err).WithField("environmentID", env.ID).Error("Failed to cascade delete environment during listener cleanup")
			}
		}
	}

	// Delete listener from repository
	if err := s.repo.DeleteListener(ctx, id); err != nil {
		return fmt.Errorf("failed to delete listener: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"listenerID": id,
		"gatewayID":  listener.GatewayID,
		"port":       listener.Port,
		"force":      force,
	}).Info("Listener deleted successfully")

	return nil
}

// cascadeDeleteEnvironment deletes an environment and all its deployments.
func (s *ListenerService) cascadeDeleteEnvironment(ctx context.Context, environmentID string) error {
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

// CountEnvironmentsByListener returns the number of environments for a listener.
func (s *ListenerService) CountEnvironmentsByListener(ctx context.Context, id string) (int, error) {
	// Validate listener exists
	_, err := s.repo.GetListener(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, fmt.Errorf("listener not found: %s", id)
		}
		return 0, fmt.Errorf("failed to get listener: %w", err)
	}

	count, err := s.repo.CountEnvironmentsByListener(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("failed to count environments: %w", err)
	}
	return count, nil
}

// updateListenerSnapshot generates and updates the xDS listener resource for a listener.
// This creates a listener with filter chains for all environments on the listener,
// and creates empty route configurations for each environment.
func (s *ListenerService) updateListenerSnapshot(ctx context.Context, nodeID string, listener *models.Listener) error {
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

// validateEnvironmentUniqueness checks for duplicate environment names and hostnames within the request.
// This prevents conflicts when creating multiple environments for a listener.
func (s *ListenerService) validateEnvironmentUniqueness(envConfigs []models.EnvironmentConfig) error {
	nameMap := make(map[string]bool)
	hostnameMap := make(map[string]bool)

	for _, ec := range envConfigs {
		// Validate required fields
		if ec.Name == "" {
			return fmt.Errorf("environment name is required")
		}
		if ec.Hostname == "" {
			return fmt.Errorf("environment hostname is required")
		}

		// Check for duplicate names
		if nameMap[ec.Name] {
			return fmt.Errorf("duplicate environment name '%s' in request", ec.Name)
		}
		nameMap[ec.Name] = true

		// Check for duplicate hostnames
		if hostnameMap[ec.Hostname] {
			return fmt.Errorf("duplicate hostname '%s' in request", ec.Hostname)
		}
		hostnameMap[ec.Hostname] = true
	}

	return nil
}

// createEnvironmentsForListener creates multiple environments for a listener atomically.
// If any environment creation fails, all previously created environments are rolled back.
// Returns the created environments or an error if any step fails.
func (s *ListenerService) createEnvironmentsForListener(
	ctx context.Context,
	listener *models.Listener,
	envConfigs []models.EnvironmentConfig,
) ([]*models.GatewayEnvironment, error) {
	environments := make([]*models.GatewayEnvironment, 0, len(envConfigs))

	for _, ec := range envConfigs {
		// Check name uniqueness within listener
		nameExists, err := s.repo.EnvironmentExists(ctx, listener.ID, ec.Name)
		if err != nil {
			// Rollback already created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			return nil, fmt.Errorf("failed to check environment name uniqueness: %w", err)
		}
		if nameExists {
			// Rollback already created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			return nil, fmt.Errorf("environment name '%s' already exists on this listener", ec.Name)
		}

		// Check hostname uniqueness within listener
		hostnameExists, err := s.repo.HostnameExistsOnListener(ctx, listener.ID, ec.Hostname)
		if err != nil {
			// Rollback already created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			return nil, fmt.Errorf("failed to check hostname uniqueness: %w", err)
		}
		if hostnameExists {
			// Rollback already created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			return nil, fmt.Errorf("hostname '%s' already exists on this listener", ec.Hostname)
		}

		// Create environment entity
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
			// Rollback all previously created environments
			for _, env := range environments {
				_ = s.repo.DeleteEnvironment(ctx, env.ID)
			}
			return nil, fmt.Errorf("failed to create environment '%s': %w", ec.Name, err)
		}

		environments = append(environments, environment)

		s.logger.WithFields(map[string]interface{}{
			"environmentID": environment.ID,
			"listenerID":    listener.ID,
			"name":          environment.Name,
			"hostname":      environment.Hostname,
		}).Debug("Environment created")
	}

	return environments, nil
}
