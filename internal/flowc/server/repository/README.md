# Repository Package

The repository package provides a data access abstraction layer for the FlowC control plane. It follows the **Repository Pattern** to decouple the service layer from the underlying data storage, enabling flexibility to swap storage backends without changing business logic.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Service Layer                            │
│                    (DeploymentService, etc.)                    │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Repository Interface                         │
│   DeploymentRepository + NodeMappingRepository + StatsRepository│
└────────────────────────────┬────────────────────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│    Memory     │    │   PostgreSQL  │    │    Redis      │
│  Repository   │    │  Repository   │    │  Repository   │
│ (implemented) │    │   (planned)   │    │  (planned)    │
└───────────────┘    └───────────────┘    └───────────────┘
```

## Interfaces

### Repository (Combined)

The main `Repository` interface combines all sub-interfaces:

```go
type Repository interface {
    DeploymentRepository
    NodeMappingRepository
    StatsRepository
    Close() error
    Ping(ctx context.Context) error
}
```

### DeploymentRepository

Handles CRUD operations for API deployments:

```go
type DeploymentRepository interface {
    Create(ctx context.Context, deployment *models.APIDeployment) error
    Get(ctx context.Context, id string) (*models.APIDeployment, error)
    Update(ctx context.Context, deployment *models.APIDeployment) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) ([]*models.APIDeployment, error)
    ListByStatus(ctx context.Context, status models.DeploymentStatus) ([]*models.APIDeployment, error)
    Count(ctx context.Context) (int, error)
    Exists(ctx context.Context, id string) (bool, error)
}
```

### NodeMappingRepository

Manages deployment-to-xDS-node-ID mappings:

```go
type NodeMappingRepository interface {
    SetNodeID(ctx context.Context, deploymentID, nodeID string) error
    GetNodeID(ctx context.Context, deploymentID string) (string, error)
    DeleteNodeID(ctx context.Context, deploymentID string) error
    GetDeploymentsByNodeID(ctx context.Context, nodeID string) ([]string, error)
}
```

### StatsRepository

Provides deployment statistics:

```go
type StatsRepository interface {
    GetStats(ctx context.Context) (*DeploymentStats, error)
}
```

## Usage

### Default In-Memory Repository

```go
// Use the default constructor (backward compatible)
service := NewDeploymentService(configManager, logger)
```

### Custom Repository

```go
// Create a custom repository
repo := repository.NewMemoryRepository()

// Or use the factory with configuration
config := &repository.RepositoryConfig{
    Type: repository.RepositoryTypeMemory,
}
repo, err := repository.NewRepository(config)
if err != nil {
    log.Fatal(err)
}

// Inject into service
service := NewDeploymentServiceWithRepository(configManager, logger, repo)
```

### Using the Factory

```go
config := &repository.RepositoryConfig{
    Type:             repository.RepositoryTypePostgres, // When implemented
    ConnectionString: "postgres://user:pass@localhost/flowc",
    MaxConnections:   10,
    ConnectTimeout:   "5s",
}

factory := repository.NewFactory(config)
repo, err := factory.Create()
if err != nil {
    log.Fatal(err)
}
defer repo.Close()
```

## Error Handling

The package defines standard errors for consistent error handling:

```go
var (
    ErrNotFound         = errors.New("resource not found")
    ErrAlreadyExists    = errors.New("resource already exists")
    ErrInvalidInput     = errors.New("invalid input")
    ErrConnectionFailed = errors.New("connection failed")
    ErrTransactionFailed = errors.New("transaction failed")
)
```

Usage in service layer:

```go
deployment, err := repo.Get(ctx, id)
if errors.Is(err, repository.ErrNotFound) {
    return nil, fmt.Errorf("deployment not found: %s", id)
}
```

## Implementations

### MemoryRepository (Implemented)

Thread-safe in-memory implementation using `sync.RWMutex`. Suitable for:
- Development and testing
- Single-instance deployments
- Ephemeral workloads

Features:
- Zero external dependencies
- Fast operations (O(1) for most operations)
- Data is lost on restart

### Future Implementations

The following implementations are planned:

| Type | Use Case | Status |
|------|----------|--------|
| `postgres` | Production with ACID guarantees | Planned |
| `mysql` | Production with ACID guarantees | Planned |
| `redis` | High-performance caching layer | Planned |
| `mongodb` | Document-oriented storage | Planned |

## Implementing a New Repository

To implement a new repository backend:

1. Create a new file (e.g., `postgres.go`)
2. Implement the `Repository` interface
3. Add the type to `RepositoryType` constants
4. Update the factory's `NewRepository` function

Example skeleton:

```go
type PostgresRepository struct {
    db *sql.DB
}

func NewPostgresRepository(connStr string) (*PostgresRepository, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
    }
    return &PostgresRepository{db: db}, nil
}

// Implement all Repository interface methods...
```

## Thread Safety

All repository implementations MUST be thread-safe:
- `MemoryRepository`: Uses `sync.RWMutex` for concurrent access
- Database implementations: Rely on connection pools and database-level locking

## Context Support

All methods accept `context.Context` for:
- Cancellation propagation
- Timeout handling
- Request tracing (future)

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

deployment, err := repo.Get(ctx, id)
```

## Testing

The in-memory repository is ideal for unit testing:

```go
func TestDeploymentService(t *testing.T) {
    repo := repository.NewMemoryRepository()
    service := NewDeploymentServiceWithRepository(configManager, logger, repo)
    
    // Test your service...
}
```

