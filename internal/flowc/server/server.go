package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/internal/flowc/server/handlers"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// APIServer represents the REST API server with declarative resource endpoints.
type APIServer struct {
	mux          *http.ServeMux
	server       *http.Server
	store        store.Store
	logger       *logger.EnvoyLogger
	port         int
	xdsPort      int
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	startTime    time.Time
}

// NewAPIServer creates a new API server instance with the declarative resource store.
// xdsPort is the gRPC xDS port used for bootstrap config generation.
func NewAPIServer(port, xdsPort int, readTimeout, writeTimeout, idleTimeout time.Duration, resourceStore store.Store, logger *logger.EnvoyLogger) *APIServer {
	mux := http.NewServeMux()

	s := &APIServer{
		mux:          mux,
		store:        resourceStore,
		logger:       logger,
		port:         port,
		xdsPort:      xdsPort,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		idleTimeout:  idleTimeout,
		startTime:    time.Now(),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes using Go 1.22+ method-based routing.
func (s *APIServer) setupRoutes() {
	rh := handlers.NewResourceHandler(s.store, s.logger)
	uh := handlers.NewUploadHandler(s.store, s.logger)
	bh := handlers.NewBootstrapHandler(s.store, "host.docker.internal", s.xdsPort, s.logger)
	dh := handlers.NewDeployHandler(s.store, "host.docker.internal", s.xdsPort, s.port, s.logger)

	// Health
	s.mux.HandleFunc("GET /health", rh.HealthCheck(s.startTime))

	// Root
	s.mux.HandleFunc("GET /", s.handleRoot)

	// --- Flat K8s-style resource endpoints ---

	// Gateway Profiles
	s.mux.HandleFunc("PUT /api/v1/gatewayprofiles/{name}", rh.HandlePut(resource.KindGatewayProfile))
	s.mux.HandleFunc("GET /api/v1/gatewayprofiles/{name}", rh.HandleGet(resource.KindGatewayProfile))
	s.mux.HandleFunc("GET /api/v1/gatewayprofiles", rh.HandleList(resource.KindGatewayProfile))
	s.mux.HandleFunc("DELETE /api/v1/gatewayprofiles/{name}", rh.HandleDelete(resource.KindGatewayProfile))

	// Gateways
	s.mux.HandleFunc("PUT /api/v1/gateways/{name}", rh.HandlePut(resource.KindGateway))
	s.mux.HandleFunc("GET /api/v1/gateways/{name}", rh.HandleGet(resource.KindGateway))
	s.mux.HandleFunc("GET /api/v1/gateways", rh.HandleList(resource.KindGateway))
	s.mux.HandleFunc("DELETE /api/v1/gateways/{name}", rh.HandleDelete(resource.KindGateway))

	// Listeners
	s.mux.HandleFunc("PUT /api/v1/listeners/{name}", rh.HandlePut(resource.KindListener))
	s.mux.HandleFunc("GET /api/v1/listeners/{name}", rh.HandleGet(resource.KindListener))
	s.mux.HandleFunc("GET /api/v1/listeners", rh.HandleList(resource.KindListener))
	s.mux.HandleFunc("DELETE /api/v1/listeners/{name}", rh.HandleDelete(resource.KindListener))

	// Environments
	s.mux.HandleFunc("PUT /api/v1/environments/{name}", rh.HandlePut(resource.KindEnvironment))
	s.mux.HandleFunc("GET /api/v1/environments/{name}", rh.HandleGet(resource.KindEnvironment))
	s.mux.HandleFunc("GET /api/v1/environments", rh.HandleList(resource.KindEnvironment))
	s.mux.HandleFunc("DELETE /api/v1/environments/{name}", rh.HandleDelete(resource.KindEnvironment))

	// APIs
	s.mux.HandleFunc("PUT /api/v1/apis/{name}", rh.HandlePut(resource.KindAPI))
	s.mux.HandleFunc("GET /api/v1/apis/{name}", rh.HandleGet(resource.KindAPI))
	s.mux.HandleFunc("GET /api/v1/apis", rh.HandleList(resource.KindAPI))
	s.mux.HandleFunc("DELETE /api/v1/apis/{name}", rh.HandleDelete(resource.KindAPI))

	// Deployments
	s.mux.HandleFunc("PUT /api/v1/deployments/{name}", rh.HandlePut(resource.KindDeployment))
	s.mux.HandleFunc("GET /api/v1/deployments/{name}", rh.HandleGet(resource.KindDeployment))
	s.mux.HandleFunc("GET /api/v1/deployments", rh.HandleList(resource.KindDeployment))
	s.mux.HandleFunc("DELETE /api/v1/deployments/{name}", rh.HandleDelete(resource.KindDeployment))

	// Gateway bootstrap and deployment instructions
	s.mux.HandleFunc("GET /api/v1/gateways/{name}/bootstrap", bh.HandleBootstrap)
	s.mux.HandleFunc("GET /api/v1/gateways/{name}/deploy", dh.HandleDeploy)

	// Bulk apply
	s.mux.HandleFunc("POST /api/v1/apply", rh.HandleApply)

	// ZIP upload convenience
	s.mux.HandleFunc("POST /api/v1/upload", uh.HandleUpload)
}

// handleRoot serves the API documentation.
func (s *APIServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := map[string]interface{}{
		"service":     "FlowC Control Plane",
		"version":     "3.0.0",
		"description": "Declarative Envoy xDS control plane with reconciliation-based architecture",
		"api_style":   "Flat K8s-style: PUT to create/update, GET/DELETE, POST /apply for bulk",
		"endpoints": map[string]interface{}{
			"health": "GET /health",
			"resources": map[string]string{
				"gatewayprofiles": "/api/v1/gatewayprofiles/{name}",
				"gateways":        "/api/v1/gateways/{name}",
				"listeners":       "/api/v1/listeners/{name}",
				"environments":    "/api/v1/environments/{name}",
				"apis":            "/api/v1/apis/{name}",
				"deployments":     "/api/v1/deployments/{name}",
			},
			"bulk_apply": "POST /api/v1/apply",
			"upload":     "POST /api/v1/upload",
		},
		"notes": []string{
			"All resources use PUT for idempotent create-or-update",
			"Hierarchy is expressed through spec reference fields (gatewayRef, listenerRef, etc.)",
			"Reconciler watches the store and generates xDS snapshots automatically",
			"Use If-Match header for optimistic concurrency control",
			"Use X-Managed-By header for ownership tracking",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers to all responses.
func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Managed-By, If-Match")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the API server.
func (s *APIServer) Start() error {
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.corsMiddleware(s.mux),
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

// Stop gracefully stops the API server.
func (s *APIServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping FlowC API server")

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}
