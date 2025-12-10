package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/server/repository"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	listenerresource "github.com/flowc-labs/flowc/internal/flowc/xds/resources/listener"
	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/google/uuid"
)

// EnvironmentService manages environment lifecycle within listeners.
// Environments use hostname-based SNI for filter chain matching, allowing
// multiple isolated environments to share the same listener port.
type EnvironmentService struct {
	configManager *cache.ConfigManager
	logger        *logger.EnvoyLogger
	repo          repository.Repository
}

// NewEnvironmentService creates a new environment service with a default in-memory repository.
func NewEnvironmentService(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *EnvironmentService {
	return NewEnvironmentServiceWithRepository(configManager, logger, repository.NewDefaultRepository())
}

// NewEnvironmentServiceWithRepository creates an environment service with a custom repository.
func NewEnvironmentServiceWithRepository(configManager *cache.ConfigManager, logger *logger.EnvoyLogger, repo repository.Repository) *EnvironmentService {
	return &EnvironmentService{
		configManager: configManager,
		logger:        logger,
		repo:          repo,
	}
}

// CreateEnvironment creates a new environment within a listener.
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, listenerID string, req *models.CreateEnvironmentRequest) (*models.GatewayEnvironment, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}

	// Validate listener exists
	_, err := s.repo.GetListener(ctx, listenerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("listener not found: %s", listenerID)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}

	// Check if name is already in use for this listener
	exists, err := s.repo.EnvironmentExists(ctx, listenerID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check environment existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("environment name '%s' is already in use on this listener", req.Name)
	}

	// Check if hostname is already in use for this listener
	hostnameExists, err := s.repo.HostnameExistsOnListener(ctx, listenerID, req.Hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to check hostname existence: %w", err)
	}
	if hostnameExists {
		return nil, fmt.Errorf("hostname '%s' is already in use on this listener", req.Hostname)
	}

	env := &models.GatewayEnvironment{
		ID:          uuid.New().String(),
		ListenerID:  listenerID,
		Name:        req.Name,
		Hostname:    req.Hostname,
		Description: req.Description,
		HTTPFilters: req.HTTPFilters,
		Labels:      req.Labels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store environment in repository
	if err := s.repo.CreateEnvironment(ctx, env); err != nil {
		if errors.Is(err, repository.ErrEnvironmentNameInUse) {
			return nil, fmt.Errorf("environment name '%s' is already in use on this listener", req.Name)
		}
		if errors.Is(err, repository.ErrHostnameInUse) {
			return nil, fmt.Errorf("hostname '%s' is already in use on this listener", req.Hostname)
		}
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	// Get the listener to retrieve gateway info and regenerate xDS listener
	listenerModel, err := s.repo.GetListener(ctx, listenerID)
	if err != nil {
		// Rollback: delete the environment we just created
		s.repo.DeleteEnvironment(ctx, env.ID)
		return nil, fmt.Errorf("failed to get listener after creating environment: %w", err)
	}

	// Get the gateway to retrieve the NodeID
	gateway, err := s.repo.GetGateway(ctx, listenerModel.GatewayID)
	if err != nil {
		// Rollback: delete the environment we just created
		s.repo.DeleteEnvironment(ctx, env.ID)
		return nil, fmt.Errorf("failed to get gateway after creating environment: %w", err)
	}

	// Regenerate the xDS listener with the new environment
	if err := s.updateListenerSnapshot(ctx, gateway.NodeID, listenerModel); err != nil {
		// Rollback: delete the environment we just created
		s.repo.DeleteEnvironment(ctx, env.ID)
		return nil, fmt.Errorf("failed to update xDS snapshot: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"environmentID": env.ID,
		"listenerID":    listenerID,
		"name":          env.Name,
		"hostname":      env.Hostname,
	}).Info("Environment created successfully")

	return env, nil
}

// GetEnvironment retrieves an environment by ID.
func (s *EnvironmentService) GetEnvironment(ctx context.Context, id string) (*models.GatewayEnvironment, error) {
	env, err := s.repo.GetEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("environment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}
	return env, nil
}

// GetEnvironmentByListenerAndName retrieves an environment by listener ID and name.
func (s *EnvironmentService) GetEnvironmentByListenerAndName(ctx context.Context, listenerID, name string) (*models.GatewayEnvironment, error) {
	env, err := s.repo.GetEnvironmentByListenerAndName(ctx, listenerID, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("environment '%s' not found on listener %s", name, listenerID)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}
	return env, nil
}

// ListEnvironmentsByListener retrieves all environments for a listener.
func (s *EnvironmentService) ListEnvironmentsByListener(ctx context.Context, listenerID string) ([]*models.GatewayEnvironment, error) {
	// Validate listener exists
	_, err := s.repo.GetListener(ctx, listenerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("listener not found: %s", listenerID)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}

	envs, err := s.repo.ListEnvironmentsByListener(ctx, listenerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	return envs, nil
}

// UpdateEnvironment updates an existing environment's configuration.
// Only the fields provided in the request will be updated.
// Note: Name cannot be changed after creation.
func (s *EnvironmentService) UpdateEnvironment(ctx context.Context, id string, req *models.UpdateEnvironmentRequest) (*models.GatewayEnvironment, error) {
	env, err := s.repo.GetEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("environment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// Apply partial updates - only update fields that are provided
	if req.Hostname != nil {
		// Check if new hostname is already in use (by another environment)
		if *req.Hostname != env.Hostname {
			hostnameExists, err := s.repo.HostnameExistsOnListener(ctx, env.ListenerID, *req.Hostname)
			if err != nil {
				return nil, fmt.Errorf("failed to check hostname existence: %w", err)
			}
			if hostnameExists {
				return nil, fmt.Errorf("hostname '%s' is already in use on this listener", *req.Hostname)
			}
		}
		env.Hostname = *req.Hostname
	}
	if req.Description != nil {
		env.Description = *req.Description
	}
	if req.HTTPFilters != nil {
		env.HTTPFilters = req.HTTPFilters
	}
	if req.Labels != nil {
		env.Labels = req.Labels
	}
	env.UpdatedAt = time.Now()

	if err := s.repo.UpdateEnvironment(ctx, env); err != nil {
		if errors.Is(err, repository.ErrHostnameInUse) {
			return nil, fmt.Errorf("hostname is already in use on this listener")
		}
		return nil, fmt.Errorf("failed to update environment: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"environmentID": env.ID,
		"listenerID":    env.ListenerID,
		"name":          env.Name,
	}).Info("Environment updated successfully")

	return env, nil
}

// DeleteEnvironment removes an environment from a listener.
// If force is false and the environment has active deployments, the deletion will fail.
// If force is true, all deployments will be removed first.
func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, id string, force bool) error {
	env, err := s.repo.GetEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("environment not found: %s", id)
		}
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Check for deployments on this environment
	deploymentIDs, err := s.repo.GetDeploymentsByEnvironment(ctx, env.ID)
	if err != nil {
		return fmt.Errorf("failed to check deployments: %w", err)
	}

	if len(deploymentIDs) > 0 {
		if !force {
			return fmt.Errorf("environment has %d active deployment(s); use force=true to delete anyway", len(deploymentIDs))
		}

		// Force delete: remove all deployments first
		s.logger.WithFields(map[string]interface{}{
			"environmentID": id,
			"name":          env.Name,
			"deployments":   len(deploymentIDs),
		}).Warn("Force deleting environment with active deployments")

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
	}

	// Delete environment from repository
	if err := s.repo.DeleteEnvironment(ctx, id); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"environmentID": id,
		"listenerID":    env.ListenerID,
		"name":          env.Name,
		"force":         force,
	}).Info("Environment deleted successfully")

	return nil
}

// GetEnvironmentAPIs retrieves all API deployments associated with an environment.
func (s *EnvironmentService) GetEnvironmentAPIs(ctx context.Context, id string) ([]*models.APIDeployment, error) {
	// Validate environment exists
	_, err := s.repo.GetEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("environment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	deploymentIDs, err := s.repo.GetDeploymentsByEnvironment(ctx, id)
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

// CountEnvironmentAPIs returns the number of API deployments on an environment.
func (s *EnvironmentService) CountEnvironmentAPIs(ctx context.Context, id string) (int, error) {
	// Validate environment exists
	_, err := s.repo.GetEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, fmt.Errorf("environment not found: %s", id)
		}
		return 0, fmt.Errorf("failed to get environment: %w", err)
	}

	deploymentIDs, err := s.repo.GetDeploymentsByEnvironment(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("failed to get deployments: %w", err)
	}

	return len(deploymentIDs), nil
}

// updateListenerSnapshot regenerates and updates the xDS listener resource for a listener.
// This creates a listener with filter chains for all environments on the listener.
func (s *EnvironmentService) updateListenerSnapshot(ctx context.Context, nodeID string, listenerModel *models.Listener) error {
	// Get all environments for this listener
	environments, err := s.repo.ListEnvironmentsByListener(ctx, listenerModel.ID)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	// Build filter chain configurations for each environment
	filterChains := make([]*listenerresource.FilterChainConfig, 0, len(environments))
	for _, env := range environments {
		// Create route config name for this environment
		routeConfigName := fmt.Sprintf("route_%s_%s", listenerModel.ID, env.Name)

		filterChains = append(filterChains, &listenerresource.FilterChainConfig{
			Name:            env.Name,
			Hostname:        env.Hostname,
			HTTPFilters:     env.HTTPFilters,
			RouteConfigName: routeConfigName,
			// TODO: Map models.TLSConfig to listenerresource.TLSConfig if listener has TLS
		})
	}

	// Create listener configuration
	listenerConfig := &listenerresource.ListenerConfig{
		Name:         fmt.Sprintf("listener_%d", listenerModel.Port),
		Port:         listenerModel.Port,
		Address:      listenerModel.Address,
		FilterChains: filterChains,
		HTTP2:        listenerModel.HTTP2,
		AccessLog:    listenerModel.AccessLog,
	}

	// Generate xDS listener resource
	xdsListener, err := listenerresource.CreateListenerWithFilterChains(listenerConfig)
	if err != nil {
		return fmt.Errorf("failed to create xDS listener: %w", err)
	}

	// Update the xDS snapshot
	if err := s.configManager.AddListener(nodeID, listenerConfig.Name, xdsListener); err != nil {
		return fmt.Errorf("failed to add listener to snapshot: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"nodeID":       nodeID,
		"listenerID":   listenerModel.ID,
		"port":         listenerModel.Port,
		"environments": len(environments),
	}).Info("Updated xDS listener snapshot after environment change")

	return nil
}
