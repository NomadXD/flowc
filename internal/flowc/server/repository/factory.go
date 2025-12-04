package repository

import (
	"fmt"
)

// Factory creates repository instances based on configuration.
type Factory struct {
	config *RepositoryConfig
}

// NewFactory creates a new repository factory with the given configuration.
func NewFactory(config *RepositoryConfig) *Factory {
	if config == nil {
		config = DefaultConfig()
	}
	return &Factory{
		config: config,
	}
}

// Create creates a new repository instance based on the factory's configuration.
func (f *Factory) Create() (Repository, error) {
	return NewRepository(f.config)
}

// NewRepository creates a new repository instance based on the provided configuration.
// This is the main entry point for creating repositories.
func NewRepository(config *RepositoryConfig) (Repository, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case RepositoryTypeMemory, "":
		return NewMemoryRepository(), nil

	case RepositoryTypePostgres:
		return nil, fmt.Errorf("postgres repository not implemented: use 'memory' for now")

	case RepositoryTypeMySQL:
		return nil, fmt.Errorf("mysql repository not implemented: use 'memory' for now")

	case RepositoryTypeRedis:
		return nil, fmt.Errorf("redis repository not implemented: use 'memory' for now")

	case RepositoryTypeMongoDB:
		return nil, fmt.Errorf("mongodb repository not implemented: use 'memory' for now")

	default:
		return nil, fmt.Errorf("unknown repository type: %s", config.Type)
	}
}

// MustNewRepository creates a new repository or panics if an error occurs.
// This is useful for initialization in main() where errors should be fatal.
func MustNewRepository(config *RepositoryConfig) Repository {
	repo, err := NewRepository(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create repository: %v", err))
	}
	return repo
}

// NewDefaultRepository creates a new in-memory repository with default configuration.
// This is a convenience function for simple use cases.
func NewDefaultRepository() Repository {
	return NewMemoryRepository()
}
