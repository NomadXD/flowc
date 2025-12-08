package service

import (
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/pkg/logger"
)

type GatewayService struct {
	configManager *cache.ConfigManager
	logger        *logger.EnvoyLogger
}

func NewGatewayService(configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *GatewayService {
	return &GatewayService{
		configManager: configManager,
		logger:        logger,
	}
}
