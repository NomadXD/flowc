package service

import (
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/pkg/logger"
)

type Services struct {
	DeploymentService *DeploymentService
	GatewayService    *GatewayService
}

func NewServices(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *Services {
	deploymentService := NewDeploymentService(configManager, logger)
	gatewayService := NewGatewayService(configManager, logger)
	return &Services{
		DeploymentService: deploymentService,
		GatewayService:    gatewayService,
	}
}
