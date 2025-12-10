package service

import (
	"github.com/flowc-labs/flowc/internal/flowc/server/repository"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// Services is a container for all service instances.
// It ensures that services share a single repository instance for data consistency.
type Services struct {
	DeploymentService  *DeploymentService
	GatewayService     *GatewayService
	ListenerService    *ListenerService
	EnvironmentService *EnvironmentService
}

// NewServices creates a new Services container with a shared repository.
// This ensures that all services use the same repository instance, which is required
// for consistent data access across gateway, listener, environment, and deployment operations.
func NewServices(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *Services {
	// Create a shared repository instance for all services
	repo := repository.NewDefaultRepository()

	// Create services with the shared repository
	deploymentService := NewDeploymentServiceWithRepository(configManager, logger, repo)
	gatewayService := NewGatewayServiceWithRepository(configManager, logger, repo)
	listenerService := NewListenerServiceWithRepository(configManager, logger, repo)
	environmentService := NewEnvironmentServiceWithRepository(configManager, logger, repo)

	return &Services{
		DeploymentService:  deploymentService,
		GatewayService:     gatewayService,
		ListenerService:    listenerService,
		EnvironmentService: environmentService,
	}
}

// NewServicesWithRepository creates a new Services container with a custom repository.
// This is useful for testing or when using a different storage backend.
func NewServicesWithRepository(configManager *cache.ConfigManager, logger *logger.EnvoyLogger, repo repository.Repository) *Services {
	deploymentService := NewDeploymentServiceWithRepository(configManager, logger, repo)
	gatewayService := NewGatewayServiceWithRepository(configManager, logger, repo)
	listenerService := NewListenerServiceWithRepository(configManager, logger, repo)
	environmentService := NewEnvironmentServiceWithRepository(configManager, logger, repo)

	return &Services{
		DeploymentService:  deploymentService,
		GatewayService:     gatewayService,
		ListenerService:    listenerService,
		EnvironmentService: environmentService,
	}
}
