package server

import (
	"context"
	"fmt"
	"net"
	"time"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/flowc-labs/flowc/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// XDSServer represents the XDS control plane server
type XDSServer struct {
	grpcServer *grpc.Server
	cache      cachev3.SnapshotCache
	server     serverv3.Server
	logger     *logger.EnvoyLogger
	port       int
}

// NewXDSServer creates a new XDS server instance
func NewXDSServer(port int) *XDSServer {
	envoyLogger := logger.NewDefaultEnvoyLogger()

	// Create a snapshot cache
	snapshotCache := cachev3.NewSnapshotCache(true, cachev3.IDHash{}, envoyLogger)

	// Create the XDS server
	xdsServer := serverv3.NewServer(context.Background(), snapshotCache, nil)

	// Configure gRPC server with keepalive settings
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	return &XDSServer{
		grpcServer: grpcServer,
		cache:      snapshotCache,
		server:     xdsServer,
		logger:     envoyLogger,
		port:       port,
	}
}

// RegisterServices registers all XDS services with the gRPC server
func (s *XDSServer) RegisterServices() {
	// Register the XDS services
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(s.grpcServer, s.server)
}

// Start starts the XDS server
func (s *XDSServer) Start() error {
	s.RegisterServices()

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	s.logger.WithFields(map[string]interface{}{"port": s.port}).Info("Starting XDS server")

	// Start serving
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the XDS server
func (s *XDSServer) Stop() {
	s.logger.Info("Stopping XDS server")
	s.grpcServer.GracefulStop()
}

// GetCache returns the snapshot cache for external configuration updates
func (s *XDSServer) GetCache() cachev3.SnapshotCache {
	return s.cache
}

// GetLogger returns the logger instance
func (s *XDSServer) GetLogger() *logger.EnvoyLogger {
	return s.logger
}
