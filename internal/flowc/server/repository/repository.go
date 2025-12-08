// Package repository provides data access abstractions for the FlowC control plane.
// It follows the repository pattern to decouple the service layer from the underlying
// data storage, allowing for different implementations (in-memory, RDBMS, Redis, MongoDB, etc.).
package repository

import (
	"context"
	"errors"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
)

// Common errors returned by repository implementations.
// These errors should be used by all implementations to ensure consistent error handling.
var (
	// ErrNotFound is returned when the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when attempting to create a resource that already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput is returned when the input data is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrConnectionFailed is returned when the repository cannot connect to the underlying storage.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrTransactionFailed is returned when a transaction cannot be completed.
	ErrTransactionFailed = errors.New("transaction failed")
)

// DeploymentRepository defines the interface for deployment data access.
// All methods accept a context for cancellation, timeout, and tracing support.
// Implementations must be thread-safe.
type DeploymentRepository interface {
	// Create stores a new deployment. Returns ErrAlreadyExists if the deployment ID already exists.
	Create(ctx context.Context, deployment *models.APIDeployment) error

	// Get retrieves a deployment by ID. Returns ErrNotFound if the deployment does not exist.
	Get(ctx context.Context, id string) (*models.APIDeployment, error)

	// Update modifies an existing deployment. Returns ErrNotFound if the deployment does not exist.
	Update(ctx context.Context, deployment *models.APIDeployment) error

	// Delete removes a deployment by ID. Returns ErrNotFound if the deployment does not exist.
	Delete(ctx context.Context, id string) error

	// List retrieves all deployments. Returns an empty slice if no deployments exist.
	List(ctx context.Context) ([]*models.APIDeployment, error)

	// ListByStatus retrieves deployments filtered by status.
	ListByStatus(ctx context.Context, status models.DeploymentStatus) ([]*models.APIDeployment, error)

	// Count returns the total number of deployments.
	Count(ctx context.Context) (int, error)

	// Exists checks if a deployment with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)
}

// NodeMappingRepository defines the interface for deployment-to-node ID mapping.
// This is used to track which xDS node ID is associated with each deployment.
type NodeMappingRepository interface {
	// SetNodeID associates a node ID with a deployment.
	SetNodeID(ctx context.Context, deploymentID, nodeID string) error

	// GetNodeID retrieves the node ID for a deployment. Returns ErrNotFound if no mapping exists.
	GetNodeID(ctx context.Context, deploymentID string) (string, error)

	// DeleteNodeID removes the node ID mapping for a deployment.
	DeleteNodeID(ctx context.Context, deploymentID string) error

	// GetDeploymentsByNodeID retrieves all deployment IDs associated with a node ID.
	GetDeploymentsByNodeID(ctx context.Context, nodeID string) ([]string, error)
}

// DeploymentStats contains statistics about deployments.
type DeploymentStats struct {
	Total     int `json:"total"`
	Deployed  int `json:"deployed"`
	Failed    int `json:"failed"`
	Pending   int `json:"pending"`
	Updating  int `json:"updating"`
	Deploying int `json:"deploying"`
}

// StatsRepository defines the interface for retrieving deployment statistics.
type StatsRepository interface {
	// GetStats retrieves deployment statistics.
	GetStats(ctx context.Context) (*DeploymentStats, error)
}

// Repository combines all repository interfaces into a single interface.
// This is the main interface that should be used by services.
type Repository interface {
	DeploymentRepository
	NodeMappingRepository
	StatsRepository

	// Close releases any resources held by the repository.
	Close() error

	// Ping checks if the underlying storage is accessible.
	Ping(ctx context.Context) error
}

// TransactionalRepository extends Repository with transaction support.
// Not all implementations may support transactions.
type TransactionalRepository interface {
	Repository

	// BeginTx starts a new transaction and returns a Repository scoped to that transaction.
	BeginTx(ctx context.Context) (Repository, error)

	// Commit commits the current transaction.
	Commit() error

	// Rollback aborts the current transaction.
	Rollback() error
}

// RepositoryConfig holds configuration for repository implementations.
type RepositoryConfig struct {
	// Type specifies the repository implementation type.
	Type RepositoryType `yaml:"type" json:"type"`

	// Connection string for database-backed repositories.
	ConnectionString string `yaml:"connection_string" json:"connection_string"`

	// MaxConnections is the maximum number of connections for connection pools.
	MaxConnections int `yaml:"max_connections" json:"max_connections"`

	// ConnectTimeout is the timeout for establishing connections.
	ConnectTimeout string `yaml:"connect_timeout" json:"connect_timeout"`

	// Additional implementation-specific options.
	Options map[string]interface{} `yaml:"options" json:"options"`
}

// RepositoryType represents the type of repository implementation.
type RepositoryType string

const (
	// RepositoryTypeMemory represents an in-memory repository.
	RepositoryTypeMemory RepositoryType = "memory"

	// RepositoryTypePostgres represents a PostgreSQL repository.
	RepositoryTypePostgres RepositoryType = "postgres"

	// RepositoryTypeMySQL represents a MySQL repository.
	RepositoryTypeMySQL RepositoryType = "mysql"

	// RepositoryTypeRedis represents a Redis repository.
	RepositoryTypeRedis RepositoryType = "redis"

	// RepositoryTypeMongoDB represents a MongoDB repository.
	RepositoryTypeMongoDB RepositoryType = "mongodb"
)

// DefaultConfig returns the default repository configuration (in-memory).
func DefaultConfig() *RepositoryConfig {
	return &RepositoryConfig{
		Type:           RepositoryTypeMemory,
		MaxConnections: 10,
		ConnectTimeout: "5s",
		Options:        make(map[string]interface{}),
	}
}
