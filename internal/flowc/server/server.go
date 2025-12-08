package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/server/handlers"
	service "github.com/flowc-labs/flowc/internal/flowc/server/services"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// APIServer represents the REST API server
type APIServer struct {
	mux          *http.ServeMux
	server       *http.Server
	services     *service.Services
	handlers     *handlers.Handlers
	logger       *logger.EnvoyLogger
	port         int
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
}

// NewAPIServer creates a new API server instance
func NewAPIServer(port int, readTimeout, writeTimeout, idleTimeout time.Duration, configManager *cache.ConfigManager, logger *logger.EnvoyLogger) *APIServer {
	// Create deployment service
	services := service.NewServices(configManager, logger)

	// Create handlers
	handlers := handlers.NewHandlers(services, logger)

	// Create ServeMux
	mux := http.NewServeMux()

	server := &APIServer{
		mux:          mux,
		services:     services,
		handlers:     handlers,
		logger:       logger,
		port:         port,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		idleTimeout:  idleTimeout,
	}

	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes using Go 1.22 HTTP mux with method routing
func (s *APIServer) setupRoutes() {
	// Health check endpoint
	s.mux.HandleFunc("GET /health", s.handlers.HealthCheck)

	// Root endpoint with API documentation
	s.mux.HandleFunc("GET /", s.handleRoot)

	// API v1 routes with method-specific routing
	// Deployment routes
	s.mux.HandleFunc("POST /api/v1/deployments", s.handlers.DeployAPI)
	s.mux.HandleFunc("GET /api/v1/deployments", s.handlers.ListDeployments)
	s.mux.HandleFunc("GET /api/v1/deployments/{id}", s.handlers.GetDeployment)
	s.mux.HandleFunc("PUT /api/v1/deployments/{id}", s.handlers.UpdateDeployment)
	s.mux.HandleFunc("DELETE /api/v1/deployments/{id}", s.handlers.DeleteDeployment)
	s.mux.HandleFunc("GET /api/v1/deployments/stats", s.handlers.GetDeploymentStats)

	// Validation routes
	s.mux.HandleFunc("POST /api/v1/validate", s.handlers.ValidateZip)
}

// handleRoot serves the API documentation
func (s *APIServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := map[string]interface{}{
		"service":     "FlowC API Gateway",
		"version":     "1.0.0",
		"description": "REST API for deploying APIs via zip files containing OpenAPI and FlowC specifications",
		"endpoints": map[string]interface{}{
			"health": "GET /health",
			"deployments": map[string]interface{}{
				"create": "POST /api/v1/deployments",
				"list":   "GET /api/v1/deployments",
				"get":    "GET /api/v1/deployments/{id}",
				"update": "PUT /api/v1/deployments/{id}",
				"delete": "DELETE /api/v1/deployments/{id}",
				"stats":  "GET /api/v1/deployments/stats",
			},
			"validation": "POST /api/v1/validate",
		},
	}

	s.handlers.WriteJSONResponse(w, http.StatusOK, response)
}

// Start starts the API server
func (s *APIServer) Start() error {
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.mux,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
		IdleTimeout:  s.idleTimeout,
	}

	s.logger.WithFields(map[string]interface{}{
		"port": s.port,
	}).Info("Starting FlowC API server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	return nil
}

// Stop gracefully stops the API server
func (s *APIServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping FlowC API server")

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}
