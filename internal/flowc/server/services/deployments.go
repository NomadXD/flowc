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
	"github.com/google/uuid"
)

// DeploymentService manages API deployments and their lifecycle
type DeploymentService struct {
	configManager *cache.ConfigManager
	bundleLoader  *loader.BundleLoader
	logger        *logger.EnvoyLogger
	repo          repository.Repository
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

// DeployAPI deploys an API from a zip file
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

	// Create deployment record using helper
	deploymentID := uuid.New().String()
	deployment := deploymentBundle.ToAPIDeployment(deploymentID)
	deployment.Status = string(models.StatusDeploying)

	// Node ID for xDS
	nodeID := "test-envoy-node"

	// Store deployment and node ID mapping using repository
	if err := s.repo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to store deployment: %w", err)
	}

	if err := s.repo.SetNodeID(ctx, deployment.ID, nodeID); err != nil {
		// Rollback deployment creation on failure
		_ = s.repo.Delete(ctx, deployment.ID)
		return nil, fmt.Errorf("failed to store node ID mapping: %w", err)
	}

	// Generate xDS resources using the translator architecture
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), s.logger)
	strategies, err := factory.CreateStrategySet(translator.DefaultStrategyConfig(), deployment)
	if err != nil {
		s.updateDeploymentStatus(ctx, deployment.ID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}

	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), s.logger)
	if err != nil {
		s.updateDeploymentStatus(ctx, deployment.ID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create composite translator: %w", err)
	}

	// Translate using APIDeployment + IR + nodeID
	xdsResources, err := compositeTranslator.Translate(ctx, deployment, deploymentBundle.IR, nodeID)
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

	if err := s.configManager.DeployAPI(nodeID, cacheDeployment); err != nil {
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
		"deploymentID": deployment.ID,
		"apiName":      deployment.Name,
		"apiVersion":   deployment.Version,
		"context":      deployment.Context,
	}).Info("API deployment completed successfully")

	return deployment, nil
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

	// Remove deployment record and node ID mapping from repository
	if err := s.repo.Delete(ctx, deploymentID); err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	if err := s.repo.DeleteNodeID(ctx, deploymentID); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"deploymentID": deploymentID,
			"error":        err.Error(),
		}).Warn("Failed to delete node ID mapping")
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

	// Generate new xDS resources using translator
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), s.logger)
	strategies, err := factory.CreateStrategySet(translator.DefaultStrategyConfig(), deployment)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}

	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), s.logger)
	if err != nil {
		s.updateDeploymentStatus(ctx, deploymentID, models.StatusFailed)
		return nil, fmt.Errorf("failed to create composite translator: %w", err)
	}

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
