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

	// ErrGatewayHasDeployments is returned when attempting to delete a gateway that has active deployments.
	ErrGatewayHasDeployments = errors.New("gateway has active deployments")

	// ErrGatewayHasListeners is returned when attempting to delete a gateway that has active listeners.
	ErrGatewayHasListeners = errors.New("gateway has active listeners")

	// ErrListenerHasEnvironments is returned when attempting to delete a listener that has active environments.
	ErrListenerHasEnvironments = errors.New("listener has active environments")

	// ErrEnvironmentHasDeployments is returned when attempting to delete an environment that has active deployments.
	ErrEnvironmentHasDeployments = errors.New("environment has active deployments")

	// ErrListenerPortInUse is returned when attempting to create a listener with a port that's already in use.
	ErrListenerPortInUse = errors.New("listener port already in use")

	// ErrHostnameInUse is returned when attempting to create an environment with a hostname that's already in use.
	ErrHostnameInUse = errors.New("hostname already in use on listener")

	// ErrEnvironmentNameInUse is returned when attempting to create an environment with a name that's already in use.
	ErrEnvironmentNameInUse = errors.New("environment name already in use on listener")
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

// GatewayRepository defines the interface for gateway data access.
// All methods accept a context for cancellation, timeout, and tracing support.
// Implementations must be thread-safe.
type GatewayRepository interface {
	// CreateGateway stores a new gateway. Returns ErrAlreadyExists if the NodeID already exists.
	CreateGateway(ctx context.Context, gateway *models.Gateway) error

	// GetGateway retrieves a gateway by ID. Returns ErrNotFound if the gateway does not exist.
	GetGateway(ctx context.Context, id string) (*models.Gateway, error)

	// GetGatewayByNodeID retrieves a gateway by its node ID. Returns ErrNotFound if not found.
	GetGatewayByNodeID(ctx context.Context, nodeID string) (*models.Gateway, error)

	// UpdateGateway modifies an existing gateway. Returns ErrNotFound if the gateway does not exist.
	UpdateGateway(ctx context.Context, gateway *models.Gateway) error

	// DeleteGateway removes a gateway by ID. Returns ErrNotFound if the gateway does not exist.
	DeleteGateway(ctx context.Context, id string) error

	// ListGateways retrieves all gateways. Returns an empty slice if no gateways exist.
	ListGateways(ctx context.Context) ([]*models.Gateway, error)

	// GatewayExists checks if a gateway with the given node ID exists.
	GatewayExists(ctx context.Context, nodeID string) (bool, error)

	// CountGateways returns the total number of gateways.
	CountGateways(ctx context.Context) (int, error)
}

// ListenerRepository defines the interface for listener data access.
// All methods accept a context for cancellation, timeout, and tracing support.
// Implementations must be thread-safe.
type ListenerRepository interface {
	// CreateListener stores a new listener. Returns ErrListenerPortInUse if the port is already in use.
	CreateListener(ctx context.Context, listener *models.Listener) error

	// GetListener retrieves a listener by ID. Returns ErrNotFound if the listener does not exist.
	GetListener(ctx context.Context, id string) (*models.Listener, error)

	// GetListenerByGatewayAndPort retrieves a listener by gateway ID and port. Returns ErrNotFound if not found.
	GetListenerByGatewayAndPort(ctx context.Context, gatewayID string, port uint32) (*models.Listener, error)

	// UpdateListener modifies an existing listener. Returns ErrNotFound if the listener does not exist.
	UpdateListener(ctx context.Context, listener *models.Listener) error

	// DeleteListener removes a listener by ID. Returns ErrNotFound if the listener does not exist.
	DeleteListener(ctx context.Context, id string) error

	// ListListenersByGateway retrieves all listeners for a gateway. Returns an empty slice if none exist.
	ListListenersByGateway(ctx context.Context, gatewayID string) ([]*models.Listener, error)

	// ListenerExists checks if a listener with the given gateway ID and port exists.
	ListenerExists(ctx context.Context, gatewayID string, port uint32) (bool, error)

	// CountListenersByGateway returns the number of listeners for a gateway.
	CountListenersByGateway(ctx context.Context, gatewayID string) (int, error)
}

// EnvironmentRepository defines the interface for gateway environment data access.
// All methods accept a context for cancellation, timeout, and tracing support.
// Implementations must be thread-safe.
type EnvironmentRepository interface {
	// CreateEnvironment stores a new environment. Returns ErrEnvironmentNameInUse if name is in use.
	// Returns ErrHostnameInUse if hostname is already in use on the listener.
	CreateEnvironment(ctx context.Context, env *models.GatewayEnvironment) error

	// GetEnvironment retrieves an environment by ID. Returns ErrNotFound if it does not exist.
	GetEnvironment(ctx context.Context, id string) (*models.GatewayEnvironment, error)

	// GetEnvironmentByListenerAndName retrieves an environment by listener ID and name.
	// Returns ErrNotFound if not found.
	GetEnvironmentByListenerAndName(ctx context.Context, listenerID, name string) (*models.GatewayEnvironment, error)

	// UpdateEnvironment modifies an existing environment. Returns ErrNotFound if it does not exist.
	UpdateEnvironment(ctx context.Context, env *models.GatewayEnvironment) error

	// DeleteEnvironment removes an environment by ID. Returns ErrNotFound if it does not exist.
	DeleteEnvironment(ctx context.Context, id string) error

	// ListEnvironmentsByListener retrieves all environments for a listener.
	// Returns an empty slice if none exist.
	ListEnvironmentsByListener(ctx context.Context, listenerID string) ([]*models.GatewayEnvironment, error)

	// EnvironmentExists checks if an environment with the given listener ID and name exists.
	EnvironmentExists(ctx context.Context, listenerID, name string) (bool, error)

	// HostnameExistsOnListener checks if a hostname is already in use on a listener.
	HostnameExistsOnListener(ctx context.Context, listenerID, hostname string) (bool, error)

	// CountEnvironmentsByListener returns the number of environments for a listener.
	CountEnvironmentsByListener(ctx context.Context, listenerID string) (int, error)
}

// EnvironmentMappingRepository defines the interface for deployment-to-environment ID mapping.
// This is used to track which environment each deployment belongs to.
type EnvironmentMappingRepository interface {
	// SetEnvironmentID associates an environment ID with a deployment.
	SetEnvironmentID(ctx context.Context, deploymentID, environmentID string) error

	// GetEnvironmentID retrieves the environment ID for a deployment. Returns ErrNotFound if no mapping exists.
	GetEnvironmentID(ctx context.Context, deploymentID string) (string, error)

	// DeleteEnvironmentID removes the environment ID mapping for a deployment.
	DeleteEnvironmentID(ctx context.Context, deploymentID string) error

	// GetDeploymentsByEnvironment retrieves all deployment IDs associated with an environment.
	GetDeploymentsByEnvironment(ctx context.Context, environmentID string) ([]string, error)
}

// Repository combines all repository interfaces into a single interface.
// This is the main interface that should be used by services.
type Repository interface {
	DeploymentRepository
	NodeMappingRepository
	StatsRepository
	GatewayRepository
	ListenerRepository
	EnvironmentRepository
	EnvironmentMappingRepository

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
