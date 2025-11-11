package translator

import (
	"context"
	"fmt"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/flowc-labs/flowc/internal/flowc/xds/resources/cluster"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// =============================================================================
// DEPLOYMENT STRATEGIES (Cluster Generation)
// These are the original translators refactored as deployment strategies
// =============================================================================

// BasicDeploymentStrategy implements basic 1:1 deployment
type BasicDeploymentStrategy struct {
	options *TranslatorOptions
	logger  *logger.EnvoyLogger
}

func NewBasicDeploymentStrategy(options *TranslatorOptions, log *logger.EnvoyLogger) *BasicDeploymentStrategy {
	if options == nil {
		options = DefaultTranslatorOptions()
	}
	return &BasicDeploymentStrategy{
		options: options,
		logger:  log,
	}
}

func (s *BasicDeploymentStrategy) Name() string {
	return "basic"
}

func (s *BasicDeploymentStrategy) Validate(model *DeploymentModel) error {
	if model == nil || model.Metadata == nil {
		return fmt.Errorf("model or metadata is nil")
	}
	if model.Metadata.Name == "" {
		return fmt.Errorf("deployment name is required")
	}
	if model.Metadata.Version == "" {
		return fmt.Errorf("deployment version is required")
	}
	if model.Metadata.Upstream.Host == "" {
		return fmt.Errorf("upstream host is required")
	}
	if model.Metadata.Upstream.Port == 0 {
		return fmt.Errorf("upstream port is required")
	}
	return nil
}

func (s *BasicDeploymentStrategy) GenerateClusters(ctx context.Context, model *DeploymentModel) ([]*clusterv3.Cluster, error) {
	if err := s.Validate(model); err != nil {
		return nil, err
	}

	upstream := model.Metadata.Upstream
	scheme := upstream.Scheme
	if scheme == "" {
		scheme = "http"
	}

	clusterName := s.generateClusterName(model.Metadata.Name, model.Metadata.Version)

	return []*clusterv3.Cluster{
		cluster.CreateClusterWithScheme(clusterName, upstream.Host, upstream.Port, scheme),
	}, nil
}

func (s *BasicDeploymentStrategy) GetClusterNames(model *DeploymentModel) []string {
	return []string{
		s.generateClusterName(model.Metadata.Name, model.Metadata.Version),
	}
}

func (s *BasicDeploymentStrategy) generateClusterName(name, version string) string {
	return fmt.Sprintf("%s-%s-cluster", name, version)
}

// =============================================================================

// CanaryDeploymentStrategy implements canary deployment
type CanaryDeploymentStrategy struct {
	canaryConfig *CanaryConfig
	options      *TranslatorOptions
	logger       *logger.EnvoyLogger
}

func NewCanaryDeploymentStrategy(canaryConfig *CanaryConfig, options *TranslatorOptions, log *logger.EnvoyLogger) *CanaryDeploymentStrategy {
	if options == nil {
		options = DefaultTranslatorOptions()
	}
	return &CanaryDeploymentStrategy{
		canaryConfig: canaryConfig,
		options:      options,
		logger:       log,
	}
}

func (s *CanaryDeploymentStrategy) Name() string {
	return "canary"
}

func (s *CanaryDeploymentStrategy) Validate(model *DeploymentModel) error {
	// Basic validation
	if model == nil || model.Metadata == nil {
		return fmt.Errorf("model or metadata is nil")
	}

	// Canary-specific validation
	if s.canaryConfig == nil {
		return fmt.Errorf("canary configuration is required")
	}
	if s.canaryConfig.BaselineVersion == "" {
		return fmt.Errorf("baseline version is required")
	}
	if s.canaryConfig.CanaryVersion == "" {
		return fmt.Errorf("canary version is required")
	}
	if s.canaryConfig.CanaryWeight < 0 || s.canaryConfig.CanaryWeight > 100 {
		return fmt.Errorf("canary weight must be between 0 and 100")
	}

	return nil
}

func (s *CanaryDeploymentStrategy) GenerateClusters(ctx context.Context, model *DeploymentModel) ([]*clusterv3.Cluster, error) {
	if err := s.Validate(model); err != nil {
		return nil, err
	}

	upstream := model.Metadata.Upstream
	scheme := upstream.Scheme
	if scheme == "" {
		scheme = "http"
	}

	// Generate clusters for both baseline and canary
	baselineCluster := cluster.CreateClusterWithScheme(
		s.generateClusterName(model.Metadata.Name, s.canaryConfig.BaselineVersion),
		upstream.Host,
		upstream.Port,
		scheme,
	)

	canaryCluster := cluster.CreateClusterWithScheme(
		s.generateClusterName(model.Metadata.Name, s.canaryConfig.CanaryVersion),
		upstream.Host,
		upstream.Port,
		scheme,
	)

	return []*clusterv3.Cluster{baselineCluster, canaryCluster}, nil
}

func (s *CanaryDeploymentStrategy) GetClusterNames(model *DeploymentModel) []string {
	return []string{
		s.generateClusterName(model.Metadata.Name, s.canaryConfig.BaselineVersion),
		s.generateClusterName(model.Metadata.Name, s.canaryConfig.CanaryVersion),
	}
}

func (s *CanaryDeploymentStrategy) generateClusterName(name, version string) string {
	return fmt.Sprintf("%s-%s-cluster", name, version)
}

// =============================================================================

// BlueGreenDeploymentStrategy implements blue-green deployment
type BlueGreenDeploymentStrategy struct {
	blueGreenConfig *BlueGreenConfig
	options         *TranslatorOptions
	logger          *logger.EnvoyLogger
}

func NewBlueGreenDeploymentStrategy(blueGreenConfig *BlueGreenConfig, options *TranslatorOptions, log *logger.EnvoyLogger) *BlueGreenDeploymentStrategy {
	if options == nil {
		options = DefaultTranslatorOptions()
	}
	return &BlueGreenDeploymentStrategy{
		blueGreenConfig: blueGreenConfig,
		options:         options,
		logger:          log,
	}
}

func (s *BlueGreenDeploymentStrategy) Name() string {
	return "blue-green"
}

func (s *BlueGreenDeploymentStrategy) Validate(model *DeploymentModel) error {
	if model == nil || model.Metadata == nil {
		return fmt.Errorf("model or metadata is nil")
	}

	if s.blueGreenConfig == nil {
		return fmt.Errorf("blue-green configuration is required")
	}
	if s.blueGreenConfig.ActiveVersion == "" {
		return fmt.Errorf("active version is required")
	}
	if s.blueGreenConfig.StandbyVersion == "" {
		return fmt.Errorf("standby version is required")
	}

	return nil
}

func (s *BlueGreenDeploymentStrategy) GenerateClusters(ctx context.Context, model *DeploymentModel) ([]*clusterv3.Cluster, error) {
	if err := s.Validate(model); err != nil {
		return nil, err
	}

	upstream := model.Metadata.Upstream
	scheme := upstream.Scheme
	if scheme == "" {
		scheme = "http"
	}

	// Generate clusters for both active and standby
	activeCluster := cluster.CreateClusterWithScheme(
		s.generateClusterName(model.Metadata.Name, s.blueGreenConfig.ActiveVersion, "active"),
		upstream.Host,
		upstream.Port,
		scheme,
	)

	standbyCluster := cluster.CreateClusterWithScheme(
		s.generateClusterName(model.Metadata.Name, s.blueGreenConfig.StandbyVersion, "standby"),
		upstream.Host,
		upstream.Port,
		scheme,
	)

	return []*clusterv3.Cluster{activeCluster, standbyCluster}, nil
}

func (s *BlueGreenDeploymentStrategy) GetClusterNames(model *DeploymentModel) []string {
	// Return active cluster first (primary)
	return []string{
		s.generateClusterName(model.Metadata.Name, s.blueGreenConfig.ActiveVersion, "active"),
		s.generateClusterName(model.Metadata.Name, s.blueGreenConfig.StandbyVersion, "standby"),
	}
}

func (s *BlueGreenDeploymentStrategy) generateClusterName(name, version, environment string) string {
	return fmt.Sprintf("%s-%s-%s-cluster", name, version, environment)
}

// =============================================================================

// ExternalDeploymentStrategy delegates cluster generation to external service
type ExternalDeploymentStrategy struct {
	externalConfig *ExternalTranslatorConfig
	options        *TranslatorOptions
	logger         *logger.EnvoyLogger
	// Reuse the ExternalTranslator implementation
	externalTranslator *ExternalTranslator
}

func NewExternalDeploymentStrategy(externalConfig *ExternalTranslatorConfig, options *TranslatorOptions, log *logger.EnvoyLogger) (*ExternalDeploymentStrategy, error) {
	externalTranslator, err := NewExternalTranslator(externalConfig, options, log)
	if err != nil {
		return nil, err
	}

	return &ExternalDeploymentStrategy{
		externalConfig:     externalConfig,
		options:            options,
		logger:             log,
		externalTranslator: externalTranslator,
	}, nil
}

func (s *ExternalDeploymentStrategy) Name() string {
	return "external"
}

func (s *ExternalDeploymentStrategy) Validate(model *DeploymentModel) error {
	return s.externalTranslator.Validate(model)
}

func (s *ExternalDeploymentStrategy) GenerateClusters(ctx context.Context, model *DeploymentModel) ([]*clusterv3.Cluster, error) {
	// Call external translator
	resources, err := s.externalTranslator.Translate(ctx, model)
	if err != nil {
		return nil, err
	}

	return resources.Clusters, nil
}

func (s *ExternalDeploymentStrategy) GetClusterNames(model *DeploymentModel) []string {
	// We don't know cluster names ahead of time with external strategy
	// This will be populated after translation
	return []string{}
}
