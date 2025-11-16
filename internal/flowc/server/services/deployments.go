package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/server/loader"
	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/internal/flowc/xds/translator"
	"github.com/flowc-labs/flowc/pkg/bundle"
	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/flowc-labs/flowc/pkg/openapi"
	"github.com/google/uuid"
)

// DeploymentService manages API deployments and their lifecycle
type DeploymentService struct {
	configManager *cache.ConfigManager
	bundleLoader  *loader.BundleLoader
	logger        *logger.EnvoyLogger
	deployments   map[string]*models.APIDeployment
	nodeIDs       map[string]string // Maps deployment ID to internal node ID
	mutex         sync.RWMutex
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *DeploymentService {
	return &DeploymentService{
		configManager: configManager,
		bundleLoader:  loader.NewBundleLoader(),
		logger:        logger,
		deployments:   make(map[string]*models.APIDeployment),
		nodeIDs:       make(map[string]string),
	}
}

// DeployAPI deploys an API from a zip file
func (s *DeploymentService) DeployAPI(zipData []byte, description string) (*models.APIDeployment, error) {
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
		"flowCMetadata": deploymentBundle.FlowCMetadata,
		"specSize":      len(deploymentBundle.Spec),
		"apiType":       deploymentBundle.GetAPIType(),
	}).Info("Loaded deployment bundle")

	// Create deployment record with unique node ID for xDS
	uniqueNodeID := "test-envoy-node"

	// Load OpenAPI spec from raw spec data if it's a REST API (for backward compatibility with APIDeployment model)
	var openAPISpec models.OpenAPISpec
	if deploymentBundle.IsRESTAPI() {
		// Parse OpenAPI spec from raw spec data for the deployment model
		// This is only needed for the APIDeployment model which still uses OpenAPISpec
		ctx := context.Background()
		openAPIManager := openapi.NewOpenAPIManager()
		spec, err := openAPIManager.LoadFromData(ctx, deploymentBundle.Spec)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
		}
		openAPISpec = *spec
	}

	deployment := &models.APIDeployment{
		ID:          uuid.New().String(),
		Name:        deploymentBundle.FlowCMetadata.Name,
		Version:     deploymentBundle.FlowCMetadata.Version,
		Context:     deploymentBundle.FlowCMetadata.Context,
		Status:      string(models.StatusDeploying),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    *deploymentBundle.FlowCMetadata,
		OpenAPISpec: openAPISpec,
	}

	// Store deployment and node ID mapping
	s.mutex.Lock()
	s.deployments[deployment.ID] = deployment
	s.nodeIDs[deployment.ID] = uniqueNodeID
	s.mutex.Unlock()

	// Generate xDS resources using the translator architecture
	// Use the IR-based model (OpenAPISpec is no longer needed)
	deploymentModel := translator.NewDeploymentModelWithIR(
		deploymentBundle.FlowCMetadata,
		deploymentBundle.IR, // Always populated
		deployment.ID,
	).WithNodeID(uniqueNodeID)

	// Use CompositeTranslator to generate xDS resources. This is the core of the translator architecture.
	// It orchestrates the different strategies to generate the xDS resources.
	// The factory creates the different strategies based on the configuration.
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), s.logger)
	// The strategy set is a collection of all the strategies.
	strategies, err := factory.CreateStrategySet(translator.DefaultStrategyConfig(), deploymentModel)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		s.updateDeployment(deployment)
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}
	// The composite translator is the one that orchestrates the different strategies.
	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), s.logger)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		s.updateDeployment(deployment)
		return nil, fmt.Errorf("failed to create composite translator: %w", err)
	}
	// The translate method is the one that generates the xDS resources.
	xdsResources, err := compositeTranslator.Translate(context.Background(), deploymentModel)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		s.updateDeployment(deployment)
		return nil, fmt.Errorf("failed to translate deployment to xDS resources: %w", err)
	}

	cacheDeployment := &cache.APIDeployment{
		Clusters:  xdsResources.Clusters,
		Endpoints: xdsResources.Endpoints, // Empty for static clusters
		Listeners: xdsResources.Listeners,
		Routes:    xdsResources.Routes,
	}

	if err := s.configManager.DeployAPI(uniqueNodeID, cacheDeployment); err != nil {
		deployment.Status = string(models.StatusFailed)
		s.updateDeployment(deployment)
		return nil, fmt.Errorf("failed to deploy to xDS cache: %w", err)
	}

	// Update deployment status
	deployment.Status = string(models.StatusDeployed)
	deployment.UpdatedAt = time.Now()
	s.updateDeployment(deployment)

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
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	deployment, exists := s.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	return deployment, nil
}

// ListDeployments returns all deployments
func (s *DeploymentService) ListDeployments() []*models.APIDeployment {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	deployments := make([]*models.APIDeployment, 0, len(s.deployments))
	for _, deployment := range s.deployments {
		deployments = append(deployments, deployment)
	}

	return deployments
}

// DeleteDeployment removes a deployment and its xDS resources
func (s *DeploymentService) DeleteDeployment(deploymentID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	deployment, exists := s.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	nodeID, nodeExists := s.nodeIDs[deploymentID]
	if !nodeExists {
		return fmt.Errorf("node ID not found for deployment: %s", deploymentID)
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
		"apiName":      deployment.Name,
		"nodeID":       nodeID,
	}).Info("Deleting API deployment")

	// Remove from xDS cache
	s.configManager.RemoveNode(nodeID)

	// Remove deployment record and node ID mapping
	delete(s.deployments, deploymentID)
	delete(s.nodeIDs, deploymentID)

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
	}).Info("API deployment deleted successfully")

	return nil
}

// UpdateDeployment updates an existing deployment
func (s *DeploymentService) UpdateDeployment(deploymentID string, zipData []byte) (*models.APIDeployment, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	deployment, exists := s.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	nodeID, nodeExists := s.nodeIDs[deploymentID]
	if !nodeExists {
		return nil, fmt.Errorf("node ID not found for deployment: %s", deploymentID)
	}

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
		"apiName":      deployment.Name,
		"nodeID":       nodeID,
	}).Info("Updating API deployment")

	// Set status to updating
	deployment.Status = string(models.StatusUpdating)
	deployment.UpdatedAt = time.Now()

	// Parse new zip file
	bundle, err := s.bundleLoader.LoadBundle(zipData)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		return nil, fmt.Errorf("failed to parse zip file: %w", err)
	}

	// Update deployment with new data
	deployment.Version = bundle.FlowCMetadata.Version
	deployment.Metadata = *bundle.FlowCMetadata

	// Load OpenAPI spec from raw spec data if it's a REST API (for backward compatibility with APIDeployment model)
	if bundle.IsRESTAPI() {
		ctx := context.Background()
		openAPIManager := openapi.NewOpenAPIManager()
		spec, err := openAPIManager.LoadFromData(ctx, bundle.Spec)
		if err != nil {
			deployment.Status = string(models.StatusFailed)
			return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
		}
		deployment.OpenAPISpec = *spec
	}

	// Generate new xDS resources using translator
	// Use the IR-based model (OpenAPISpec is no longer needed)
	deploymentModel := translator.NewDeploymentModelWithIR(
		bundle.FlowCMetadata,
		bundle.IR, // Always populated
		deployment.ID,
	).WithNodeID(nodeID)

	// The factory creates the different strategies based on the configuration.
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), s.logger)
	// The strategy set is a collection of all the strategies.
	strategies, err := factory.CreateStrategySet(translator.DefaultStrategyConfig(), deploymentModel)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}
	// The composite translator is the one that orchestrates the different strategies.
	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), s.logger)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		return nil, fmt.Errorf("failed to create composite translator: %w", err)
	}
	// The translate method is the one that generates the xDS resources.
	xdsResources, err := compositeTranslator.Translate(context.Background(), deploymentModel)
	if err != nil {
		deployment.Status = string(models.StatusFailed)
		return nil, fmt.Errorf("failed to translate deployment to xDS resources: %w", err)
	}

	// Deploy updated resources to xDS cache using bulk deployment
	cacheDeployment := &cache.APIDeployment{
		Clusters:  xdsResources.Clusters,
		Endpoints: xdsResources.Endpoints, // Empty for static clusters
		Listeners: xdsResources.Listeners,
		Routes:    xdsResources.Routes,
	}

	if err := s.configManager.DeployAPI(nodeID, cacheDeployment); err != nil {
		deployment.Status = string(models.StatusFailed)
		return nil, fmt.Errorf("failed to deploy to xDS cache: %w", err)
	}

	// Update status
	deployment.Status = string(models.StatusDeployed)
	deployment.UpdatedAt = time.Now()

	s.logger.WithFields(map[string]interface{}{
		"deploymentID": deploymentID,
		"apiName":      deployment.Name,
		"newVersion":   deployment.Version,
	}).Info("API deployment updated successfully")

	return deployment, nil
}

// generateXDSResources is deprecated in favor of the translator architecture
// Left here for reference but no longer used
// func (s *DeploymentService) generateXDSResources(deployment *models.APIDeployment, _ []*models.APIRoute) (*models.XDSResources, error) {
// 	// This method has been replaced by the translator pattern
// 	// See internal/flowc/xds/translator package for the new implementation
// 	return nil, fmt.Errorf("deprecated method - use translator architecture instead")
// }

// updateDeployment updates a deployment in the internal store
func (s *DeploymentService) updateDeployment(deployment *models.APIDeployment) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deployments[deployment.ID] = deployment
}

// GetDeploymentStats returns statistics about deployments
func (s *DeploymentService) GetDeploymentStats() map[string]int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := map[string]int{
		"total":    len(s.deployments),
		"deployed": 0,
		"failed":   0,
		"pending":  0,
		"updating": 0,
	}

	for _, deployment := range s.deployments {
		switch deployment.Status {
		case string(models.StatusDeployed):
			stats["deployed"]++
		case string(models.StatusFailed):
			stats["failed"]++
		case string(models.StatusPending):
			stats["pending"]++
		case string(models.StatusUpdating):
			stats["updating"]++
		}
	}

	return stats
}

// GetOpenAPIRouter returns the OpenAPI router for a deployment
// func (s *DeploymentService) GetOpenAPIRouter(deploymentID string) (routers.Router, bool) {
// 	s.mutex.RLock()
// 	defer s.mutex.RUnlock()

// 	router, exists := s.routers[deploymentID]
// 	return router, exists
// }

// ValidateRequestForDeployment validates a request against a specific deployment's OpenAPI spec
// func (s *DeploymentService) ValidateRequestForDeployment(ctx context.Context, deploymentID string, req *http.Request) error {
// 	router, exists := s.GetOpenAPIRouter(deploymentID)
// 	if !exists {
// 		return fmt.Errorf("no OpenAPI router found for deployment %s", deploymentID)
// 	}

// 	return s.openAPIManager.ValidateRequest(ctx, req, router)
// }
