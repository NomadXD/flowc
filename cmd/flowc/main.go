package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	apiServer "github.com/flowc-labs/flowc/internal/flowc/server"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	"github.com/flowc-labs/flowc/internal/flowc/xds/handlers"
	"github.com/flowc-labs/flowc/internal/flowc/xds/server"
	"github.com/flowc-labs/flowc/pkg/logger"
)

func main() {
	// Create logger
	log := logger.NewDefaultEnvoyLogger()
	log.Info("Starting FlowC XDS Control Plane...")

	// Create XDS server on port 18000
	log.Info("Creating XDS server on port 18000")
	xdsServer := server.NewXDSServer(18000)

	// Create configuration manager
	log.Info("Creating configuration manager")
	configManager := cache.NewConfigManager(xdsServer.GetCache(), xdsServer.GetLogger())

	// Create XDS handlers for generating test configuration
	log.Info("Creating XDS handlers")
	xdsHandlers := handlers.NewXDSHandlers(xdsServer.GetLogger())

	// Create test configuration at startup
	log.Info("Creating test configuration...")
	if err := createTestConfiguration(configManager, xdsHandlers, log); err != nil {
		log.WithError(err).Fatal("Failed to create test configuration")
	}
	log.Info("Test configuration created successfully")

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal")
		cancel()
		xdsServer.Stop()
	}()

	// Create REST API server on port 8080
	log.Info("Creating REST API server on port 8080")
	restAPIServer := apiServer.NewAPIServer(8080, configManager, xdsHandlers, log)

	// Start the XDS server in a goroutine
	log.Info("Starting XDS server...")
	go func() {
		if err := xdsServer.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start XDS server")
		}
	}()

	// Start the REST API server in a goroutine
	log.Info("Starting REST API server...")
	go func() {
		if err := restAPIServer.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start REST API server")
		}
	}()

	// Give the servers a moment to start
	time.Sleep(100 * time.Millisecond)

	log.Info("XDS server started successfully on port 18000")
	log.Info("REST API server started successfully on port 8080")
	log.Info("Test configuration deployed with node ID: test-envoy-node")
	log.Info("API endpoints available at http://localhost:8080")
	log.Info("Use Ctrl+C to stop the servers")

	// Keep the main goroutine alive
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := restAPIServer.Stop(shutdownCtx); err != nil {
		log.WithError(err).Error("Failed to gracefully stop REST API server")
	}

	log.Info("Servers shutdown complete")
}

// createTestConfiguration creates a complete test configuration using the handlers
func createTestConfiguration(configManager *cache.ConfigManager, xdsHandlers *handlers.XDSHandlers, log *logger.EnvoyLogger) error {
	nodeID := "test-envoy-node"

	log.Info("Creating test configuration for Envoy proxy")

	// Create test resources using the handlers
	cluster := xdsHandlers.CreateBasicCluster(
		handlers.ClusterName,
		handlers.UpstreamHost,
		handlers.UpstreamPort,
	)

	// endpoint := xdsHandlers.CreateBasicEndpoint(
	// 	handlers.ClusterName,
	// 	handlers.UpstreamHost,
	// 	handlers.UpstreamPort,
	// ) // Not needed for static cluster

	listener := xdsHandlers.CreateBasicListener(
		handlers.ListenerName,
		handlers.RouteName,
		handlers.ListenerPort,
	)

	route := xdsHandlers.CreateBasicRoute(
		handlers.RouteName,
		handlers.ClusterName,
		"/",
	)

	// Deploy the complete API configuration atomically
	// Note: Using STATIC cluster, so we don't need separate EDS endpoints
	deployment := &cache.APIDeployment{
		Clusters:  []*clusterv3.Cluster{cluster},
		Endpoints: []*endpointv3.ClusterLoadAssignment{}, // Empty - using static cluster
		Listeners: []*listenerv3.Listener{listener},
		Routes:    []*routev3.RouteConfiguration{route},
	}

	if err := configManager.DeployAPI(nodeID, deployment); err != nil {
		return fmt.Errorf("failed to deploy test API: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"nodeID":       nodeID,
		"clusterName":  handlers.ClusterName,
		"listenerPort": handlers.ListenerPort,
		"upstreamHost": handlers.UpstreamHost,
		"upstreamPort": handlers.UpstreamPort,
	}).Info("Test configuration deployed successfully")

	return nil
}
