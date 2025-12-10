package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/server/loader"
	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/server/repository"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/internal/flowc/xds/translator"
	"github.com/flowc-labs/flowc/pkg/bundle"
	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/google/uuid"
)

// DeploymentService manages API deployments and their lifecycle
type DeploymentService struct {
	configManager *cache.ConfigManager
	bundleLoader  *loader.BundleLoader
	logger        *logger.EnvoyLogger
	repo          repository.Repository
}

// DeploymentTarget contains the resolved hierarchy for a deployment
type DeploymentTarget struct {
	Gateway     *models.Gateway
	Listener    *models.Listener
	Environment *models.GatewayEnvironment
}

// NewDeploymentService creates a new deployment service with the default in-memory repository
func NewDeploymentService(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *DeploymentService {
	return NewDeploymentServiceWithRepository(configManager, logger, repository.NewDefaultRepository())
}

// NewDeploymentServiceWithRepository creates a new deployment service with a custom repository
func NewDeploymentServiceWithRepository(configManager *cache.ConfigManager, logger *logger.EnvoyLogger, repo repository.Repository) *DeploymentService {
	return &DeploymentService{
		configManager: configManager,
		bundleLoader:  loader.NewBundleLoader(),
		logger:        logger,
		repo:          repo,
	}
}

// DeployAPI deploys an API from a zip file to a specific gateway environment
func (s *DeploymentService) DeployAPI(zipData []byte, description string) (*models.APIDeployment, error) {
	ctx := context.Background()

	s.logger.WithFields(map[string]interface{}{
		"zipSize":     len(zipData),
		"description": description,
	}).Info("Starting API deployment")

	// Validate zip file
	if err := bundle.ValidateZip(zipData); err != nil {
		return nil, fmt.Errorf("zip validation failed: %w", err)
	}

	// Parse zip file
	deploymentBundle, err := s.bundleLoader.LoadBundle(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse zip file: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"name":    deploymentBundle.FlowCMetadata.Name,
		"version": deploymentBundle.FlowCMetadata.Version,
		"apiType": deploymentBundle.GetAPIType(),
	}).Info("Loaded deployment bundle")

	// Resolve the deployment target (gateway → listener → environment)
	target, err := s.resolveDeploymentTarget(ctx, &deploymentBundle.FlowCMetadata.Gateway)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve deployment target: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"gatewayID":     target.Gateway.ID,
		"gatewayName":   target.Gateway.Name,
		"listenerPort":  target.Listener.Port,
		"environmentID": target.Environment.ID,
		"envName":       target.Environment.Name,
	}).Debug("Deployment target resolved")

	// Create deployment record
	deploymentID := uuid.New().String()
	deployment := deploymentBundle.ToAPIDeployment(deploymentID)
	deployment.Status = string(models.StatusDeploying)

	// Store deployment and mappings
	if err := s.repo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to store deployment: %w", err)
	}

	// Store node ID mapping (for xDS targeting)
	if err := s.repo.SetNodeID(ctx, deployment.ID, target.Gateway.NodeID); err != nil {
		_ = s.repo.Delete(ctx, deployment.ID)
		return nil, fmt.Errorf("failed to store node ID mapping: %w", err)
	}

	// Store environment ID mapping (for querying by environment)
	if err := s.repo.SetEnvironmentID(ctx, deployment.ID, target.Environment.ID); err != nil {
		_ = s.repo.Delete(ctx, deployment.ID)
		_ = s.repo.DeleteNodeID(ctx, deployment.ID)
		return nil, fmt.Errorf("failed to store environment ID mapping: %w", err)
	}

	// Resolve strategy configuration using gateway defaults and API config
	// Precedence: API config (flowc.yaml) > Gateway defaults > Built-in defaults
	resolver := translator.NewConfigResolver(target.Gateway.Defaults, s.logger)
	resolvedConfig := resolver.Resolve(deploymentBundle.FlowCMetadata.Strategy)

	// Generate xDS resources using the translator architecture with resolved config
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), s.logger)
	strategies, err := factory.CreateStrategySet(resolvedConfig, deployment)
	if err != nil {
		s.updateDeploymentStatus(ctx, deployment.ID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}

	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), s.logger)
	if err != nil {
		s.updateDeploymentStatus(ctx, deployment.ID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create composite translator: %w", err)
	}

	// Set translation context for environment-aware xDS generation
	translationContext := &translator.TranslationContext{
		Gateway:     target.Gateway,
		Listener:    target.Listener,
		Environment: target.Environment,
	}
	compositeTranslator.SetTranslationContext(translationContext)

	// Translate using APIDeployment + IR + nodeID
	xdsResources, err := compositeTranslator.Translate(ctx, deployment, deploymentBundle.IR, target.Gateway.NodeID)
	if err != nil {
		s.updateDeploymentStatus(ctx, deployment.ID, models.StatusFailed)
		return nil, fmt.Errorf("failed to translate deployment to xDS resources: %w", err)
	}

	cacheDeployment := &cache.APIDeployment{
		Clusters:  xdsResources.Clusters,
		Endpoints: xdsResources.Endpoints,
		Listeners: xdsResources.Listeners,
		Routes:    xdsResources.Routes,
	}

	if err := s.configManager.DeployAPI(target.Gateway.NodeID, cacheDeployment); err != nil {
		s.updateDeploymentStatus(ctx, deployment.ID, models.StatusFailed)
		return nil, fmt.Errorf("failed to deploy to xDS cache: %w", err)
	}

	// Update deployment status
	deployment.Status = string(models.StatusDeployed)
	deployment.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, deployment); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deployment.ID,
			"error":        err.Error(),
		}).Error("Failed to update deployment status after successful deploy")
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID":  deployment.ID,
		"apiName":       deployment.Name,
		"apiVersion":    deployment.Version,
		"context":       deployment.Context,
		"gatewayID":     target.Gateway.ID,
		"listenerPort":  target.Listener.Port,
		"environmentID": target.Environment.ID,
	}).Info("API deployment completed successfully")

	return deployment, nil
}

// resolveDeploymentTarget resolves the deployment hierarchy from GatewayConfig
func (s *DeploymentService) resolveDeploymentTarget(ctx context.Context, gwConfig *types.GatewayConfig) (*DeploymentTarget, error) {
	// Validate required fields
	if gwConfig.Port == 0 {
		return nil, fmt.Errorf("gateway.port is required in flowc.yaml")
	}
	if gwConfig.Environment == "" {
		return nil, fmt.Errorf("gateway.environment is required in flowc.yaml")
	}

	// Get gateway by ID or NodeID
	var gateway *models.Gateway
	var err error

	if gwConfig.GatewayID != "" {
		gateway, err = s.repo.GetGateway(ctx, gwConfig.GatewayID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("gateway with id '%s' not found; create the gateway first using POST /api/v1/gateways", gwConfig.GatewayID)
			}
			return nil, fmt.Errorf("failed to get gateway: %w", err)
		}
	} else if gwConfig.NodeID != "" {
		gateway, err = s.repo.GetGatewayByNodeID(ctx, gwConfig.NodeID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("gateway with node_id '%s' not found; create the gateway first using POST /api/v1/gateways", gwConfig.NodeID)
			}
			return nil, fmt.Errorf("failed to get gateway: %w", err)
		}
	} else {
		return nil, fmt.Errorf("gateway.gateway_id or gateway.node_id is required in flowc.yaml")
	}

	// Get listener by port
	listener, err := s.repo.GetListenerByGatewayAndPort(ctx, gateway.ID, gwConfig.Port)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("listener on port %d not found on gateway '%s'; create the listener first using POST /api/v1/gateways/%s/listeners", gwConfig.Port, gateway.Name, gateway.ID)
		}
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}

	// Get environment by name
	environment, err := s.repo.GetEnvironmentByListenerAndName(ctx, listener.ID, gwConfig.Environment)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("environment '%s' not found on listener port %d; create the environment first using POST /api/v1/gateways/%s/listeners/%d/environments", gwConfig.Environment, gwConfig.Port, gateway.ID, gwConfig.Port)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return &DeploymentTarget{
		Gateway:     gateway,
		Listener:    listener,
		Environment: environment,
	}, nil
}

// GetDeployment retrieves a deployment by ID
func (s *DeploymentService) GetDeployment(deploymentID string) (*models.APIDeployment, error) {
	ctx := context.Background()

	deployment, err := s.repo.Get(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("deployment not found: %s", deploymentID)
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return deployment, nil
}

// ListDeployments returns all deployments
func (s *DeploymentService) ListDeployments() []*models.APIDeployment {
	ctx := context.Background()

	deployments, err := s.repo.List(ctx)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to list deployments")
		return []*models.APIDeployment{}
	}

	return deployments
}

// DeleteDeployment removes a deployment and its xDS resources
func (s *DeploymentService) DeleteDeployment(deploymentID string) error {
	ctx := context.Background()

	deployment, err := s.repo.Get(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("deployment not found: %s", deploymentID)
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	nodeID, err := s.repo.GetNodeID(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("node ID not found for deployment: %s", deploymentID)
		}
		return fmt.Errorf("failed to get node ID: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
		"apiName":      deployment.Name,
		"nodeID":       nodeID,
	}).Info("Deleting API deployment")

	// Remove from xDS cache
	s.configManager.RemoveNode(nodeID)

	// Remove deployment record and all mappings from repository
	if err := s.repo.Delete(ctx, deploymentID); err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	if err := s.repo.DeleteNodeID(ctx, deploymentID); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deploymentID,
			"error":        err.Error(),
		}).Warn("Failed to delete node ID mapping")
	}

	if err := s.repo.DeleteEnvironmentID(ctx, deploymentID); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deploymentID,
			"error":        err.Error(),
		}).Warn("Failed to delete environment ID mapping")
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
	}).Info("API deployment deleted successfully")

	return nil
}

// UpdateDeployment updates an existing deployment
func (s *DeploymentService) UpdateDeployment(deploymentID string, zipData []byte) (*models.APIDeployment, error) {
	ctx := context.Background()

	deployment, err := s.repo.Get(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("deployment not found: %s", deploymentID)
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	nodeID, err := s.repo.GetNodeID(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("node ID not found for deployment: %s", deploymentID)
		}
		return nil, fmt.Errorf("failed to get node ID: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
		"apiName":      deployment.Name,
		"nodeID":       nodeID,
	}).Info("Updating API deployment")

	// Set status to updating
	deployment.Status = string(models.StatusUpdating)
	deployment.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to update deployment status: %w", err)
	}

	// Parse new zip file
	deploymentBundle, err := s.bundleLoader.LoadBundle(zipData)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to parse zip file: %w", err)
	}

	// Update deployment with new data
	deployment.Version = deploymentBundle.FlowCMetadata.Version
	deployment.Context = deploymentBundle.FlowCMetadata.Context
	deployment.Metadata = *deploymentBundle.FlowCMetadata

	// Get gateway for strategy defaults
	gateway, err := s.repo.GetGatewayByNodeID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
			return nil, fmt.Errorf("gateway with node_id '%s' no longer exists", nodeID)
		}
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to get gateway: %w", err)
	}

	// Resolve strategy configuration using gateway defaults and API config
	resolver := translator.NewConfigResolver(gateway.Defaults, s.logger)
	resolvedConfig := resolver.Resolve(deploymentBundle.FlowCMetadata.Strategy)

	// Generate new xDS resources using translator with resolved config
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), s.logger)
	strategies, err := factory.CreateStrategySet(resolvedConfig, deployment)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}

	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), s.logger)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create composite translator: %w", err)
	}

	// Get deployment target for translation context
	target, err := s.resolveDeploymentTarget(ctx, &deploymentBundle.FlowCMetadata.Gateway)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to resolve deployment target: %w", err)
	}

	// Set translation context for environment-aware xDS generation
	translationContext := &translator.TranslationContext{
		Gateway:     target.Gateway,
		Listener:    target.Listener,
		Environment: target.Environment,
	}
	compositeTranslator.SetTranslationContext(translationContext)

	// Translate using APIDeployment + IR + nodeID
	xdsResources, err := compositeTranslator.Translate(ctx, deployment, deploymentBundle.IR, nodeID)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to translate deployment to xDS resources: %w", err)
	}

	// Deploy updated resources to xDS cache
	cacheDeployment := &cache.APIDeployment{
		Clusters:  xdsResources.Clusters,
		Endpoints: xdsResources.Endpoints,
		Listeners: xdsResources.Listeners,
		Routes:    xdsResources.Routes,
	}

	if err := s.configManager.DeployAPI(nodeID, cacheDeployment); err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to deploy to xDS cache: %w", err)
	}

	// Update status
	deployment.Status = string(models.StatusDeployed)
	deployment.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, deployment); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deploymentID,
			"error":        err.Error(),
		}).Error("Failed to update deployment status after successful update")
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
		"apiName":      deployment.Name,
		"newVersion":   deployment.Version,
	}).Info("API deployment updated successfully")

	return deployment, nil
}

// updateDeploymentStatus is a helper to update deployment status on failure
func (s *DeploymentService) updateDeploymentStatus(ctx context.Context, deploymentID string, status models.DeploymentStatus) {
	deployment, err := s.repo.Get(ctx, deploymentID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deploymentID,
			"error":        err.Error(),
		}).Error("Failed to get deployment for status update")
		return
	}

	deployment.Status = string(status)
	deployment.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, deployment); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deploymentID,
			"status":       status,
			"error":        err.Error(),
		}).Error("Failed to update deployment status")
	}
}

// GetDeploymentStats returns statistics about deployments
func (s *DeploymentService) GetDeploymentStats() map[string]int {
	ctx := context.Background()

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get deployment stats")
		return map[string]int{
			"total":     0,
			"deployed":  0,
			"failed":    0,
			"pending":   0,
			"updating":  0,
			"deploying": 0,
		}
	}

	return map[string]int{
		"total":     stats.Total,
		"deployed":  stats.Deployed,
		"failed":    stats.Failed,
		"pending":   stats.Pending,
		"updating":  stats.Updating,
		"deploying": stats.Deploying,
	}
}

// GetRepository returns the underlying repository.
// This can be useful for advanced use cases or testing.
func (s *DeploymentService) GetRepository() repository.Repository {
	return s.repo
}
