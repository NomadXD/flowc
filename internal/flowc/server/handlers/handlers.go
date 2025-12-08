package handlers

import (
	"time"

	services "github.com/flowc-labs/flowc/internal/flowc/server/services"
	"github.com/flowc-labs/flowc/pkg/logger"
)

type Handlers struct {
	services  *services.Services
	logger    *logger.EnvoyLogger
	startTime time.Time
}

func NewHandlers(services *services.Services, logger *logger.EnvoyLogger) *Handlers {
	return &Handlers{
		services:  services,
		logger:    logger,
		startTime: time.Now(),
	}
}
