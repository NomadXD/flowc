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

	// Gateway routes - gateways must be registered before listeners can be added
	s.mux.HandleFunc("POST /api/v1/gateways", s.handlers.CreateGateway)
	s.mux.HandleFunc("GET /api/v1/gateways", s.handlers.ListGateways)
	s.mux.HandleFunc("GET /api/v1/gateways/{id}", s.handlers.GetGateway)
	s.mux.HandleFunc("PUT /api/v1/gateways/{id}", s.handlers.UpdateGateway)
	s.mux.HandleFunc("DELETE /api/v1/gateways/{id}", s.handlers.DeleteGateway)
	s.mux.HandleFunc("GET /api/v1/gateways/{id}/apis", s.handlers.GetGatewayAPIs)

	// Listener routes - listeners must be created within a gateway
	s.mux.HandleFunc("POST /api/v1/gateways/{gateway_id}/listeners", s.handlers.CreateListener)
	s.mux.HandleFunc("GET /api/v1/gateways/{gateway_id}/listeners", s.handlers.ListListeners)
	s.mux.HandleFunc("GET /api/v1/gateways/{gateway_id}/listeners/{port}", s.handlers.GetListener)
	s.mux.HandleFunc("PUT /api/v1/gateways/{gateway_id}/listeners/{port}", s.handlers.UpdateListener)
	s.mux.HandleFunc("DELETE /api/v1/gateways/{gateway_id}/listeners/{port}", s.handlers.DeleteListener)

	// Environment routes - environments must be created within a listener
	s.mux.HandleFunc("POST /api/v1/gateways/{gateway_id}/listeners/{port}/environments", s.handlers.CreateEnvironment)
	s.mux.HandleFunc("GET /api/v1/gateways/{gateway_id}/listeners/{port}/environments", s.handlers.ListEnvironments)
	s.mux.HandleFunc("GET /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}", s.handlers.GetEnvironment)
	s.mux.HandleFunc("PUT /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}", s.handlers.UpdateEnvironment)
	s.mux.HandleFunc("DELETE /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}", s.handlers.DeleteEnvironment)
	s.mux.HandleFunc("GET /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}/apis", s.handlers.GetEnvironmentAPIs)

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
		"version":     "2.0.0",
		"description": "REST API for deploying APIs via zip files containing OpenAPI and FlowC specifications",
		"hierarchy":   "Gateway -> Listener (port) -> Environment (hostname/SNI) -> API Deployments",
		"endpoints": map[string]interface{}{
			"health": "GET /health",
			"gateways": map[string]interface{}{
				"create":   "POST /api/v1/gateways",
				"list":     "GET /api/v1/gateways",
				"get":      "GET /api/v1/gateways/{id}",
				"update":   "PUT /api/v1/gateways/{id}",
				"delete":   "DELETE /api/v1/gateways/{id}?force=true",
				"listAPIs": "GET /api/v1/gateways/{id}/apis",
			},
			"listeners": map[string]interface{}{
				"create": "POST /api/v1/gateways/{gateway_id}/listeners",
				"list":   "GET /api/v1/gateways/{gateway_id}/listeners",
				"get":    "GET /api/v1/gateways/{gateway_id}/listeners/{port}",
				"update": "PUT /api/v1/gateways/{gateway_id}/listeners/{port}",
				"delete": "DELETE /api/v1/gateways/{gateway_id}/listeners/{port}?force=true",
			},
			"environments": map[string]interface{}{
				"create":   "POST /api/v1/gateways/{gateway_id}/listeners/{port}/environments",
				"list":     "GET /api/v1/gateways/{gateway_id}/listeners/{port}/environments",
				"get":      "GET /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}",
				"update":   "PUT /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}",
				"delete":   "DELETE /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}?force=true",
				"listAPIs": "GET /api/v1/gateways/{gateway_id}/listeners/{port}/environments/{name}/apis",
			},
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
		"notes": []string{
			"Gateways represent physical Envoy proxy instances (identified by node_id)",
			"Listeners are port bindings within a gateway",
			"Environments use hostname-based SNI for filter chain matching",
			"APIs are deployed to specific environments within listeners",
			"flowc.yaml must specify gateway_id (or node_id), port, and environment name",
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
