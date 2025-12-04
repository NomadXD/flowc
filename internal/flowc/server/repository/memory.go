package repository

import (
	"context"
	"sync"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
)

// MemoryRepository is an in-memory implementation of the Repository interface.
// It uses sync.RWMutex for thread-safe operations.
// This implementation is suitable for development, testing, and single-instance deployments.
type MemoryRepository struct {
	deployments map[string]*models.APIDeployment
	nodeIDs     map[string]string // Maps deployment ID to node ID
	mutex       sync.RWMutex
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		deployments: make(map[string]*models.APIDeployment),
		nodeIDs:     make(map[string]string),
	}
}

// Ensure MemoryRepository implements Repository interface.
var _ Repository = (*MemoryRepository)(nil)

// Create stores a new deployment.
func (r *MemoryRepository) Create(ctx context.Context, deployment *models.APIDeployment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deployment == nil {
		return ErrInvalidInput
	}

	if deployment.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[deployment.ID]; exists {
		return ErrAlreadyExists
	}

	// Store a copy to prevent external modifications
	r.deployments[deployment.ID] = copyDeployment(deployment)
	return nil
}

// Get retrieves a deployment by ID.
func (r *MemoryRepository) Get(ctx context.Context, id string) (*models.APIDeployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployment, exists := r.deployments[id]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modifications
	return copyDeployment(deployment), nil
}

// Update modifies an existing deployment.
func (r *MemoryRepository) Update(ctx context.Context, deployment *models.APIDeployment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deployment == nil || deployment.ID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[deployment.ID]; !exists {
		return ErrNotFound
	}

	r.deployments[deployment.ID] = copyDeployment(deployment)
	return nil
}

// Delete removes a deployment by ID.
func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[id]; !exists {
		return ErrNotFound
	}

	delete(r.deployments, id)
	return nil
}

// List retrieves all deployments.
func (r *MemoryRepository) List(ctx context.Context) ([]*models.APIDeployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployments := make([]*models.APIDeployment, 0, len(r.deployments))
	for _, deployment := range r.deployments {
		deployments = append(deployments, copyDeployment(deployment))
	}

	return deployments, nil
}

// ListByStatus retrieves deployments filtered by status.
func (r *MemoryRepository) ListByStatus(ctx context.Context, status models.DeploymentStatus) ([]*models.APIDeployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployments := make([]*models.APIDeployment, 0)
	for _, deployment := range r.deployments {
		if deployment.Status == string(status) {
			deployments = append(deployments, copyDeployment(deployment))
		}
	}

	return deployments, nil
}

// Count returns the total number of deployments.
func (r *MemoryRepository) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.deployments), nil
}

// Exists checks if a deployment with the given ID exists.
func (r *MemoryRepository) Exists(ctx context.Context, id string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.deployments[id]
	return exists, nil
}

// SetNodeID associates a node ID with a deployment.
func (r *MemoryRepository) SetNodeID(ctx context.Context, deploymentID, nodeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deploymentID == "" || nodeID == "" {
		return ErrInvalidInput
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.nodeIDs[deploymentID] = nodeID
	return nil
}

// GetNodeID retrieves the node ID for a deployment.
func (r *MemoryRepository) GetNodeID(ctx context.Context, deploymentID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	nodeID, exists := r.nodeIDs[deploymentID]
	if !exists {
		return "", ErrNotFound
	}

	return nodeID, nil
}

// DeleteNodeID removes the node ID mapping for a deployment.
func (r *MemoryRepository) DeleteNodeID(ctx context.Context, deploymentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.nodeIDs, deploymentID)
	return nil
}

// GetDeploymentsByNodeID retrieves all deployment IDs associated with a node ID.
func (r *MemoryRepository) GetDeploymentsByNodeID(ctx context.Context, nodeID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deploymentIDs := make([]string, 0)
	for depID, nID := range r.nodeIDs {
		if nID == nodeID {
			deploymentIDs = append(deploymentIDs, depID)
		}
	}

	return deploymentIDs, nil
}

// GetStats retrieves deployment statistics.
func (r *MemoryRepository) GetStats(ctx context.Context) (*DeploymentStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := &DeploymentStats{
		Total: len(r.deployments),
	}

	for _, deployment := range r.deployments {
		switch deployment.Status {
		case string(models.StatusDeployed):
			stats.Deployed++
		case string(models.StatusFailed):
			stats.Failed++
		case string(models.StatusPending):
			stats.Pending++
		case string(models.StatusUpdating):
			stats.Updating++
		case string(models.StatusDeploying):
			stats.Deploying++
		}
	}

	return stats, nil
}

// Close releases any resources held by the repository.
// For the in-memory implementation, this is a no-op.
func (r *MemoryRepository) Close() error {
	return nil
}

// Ping checks if the underlying storage is accessible.
// For the in-memory implementation, this always succeeds.
func (r *MemoryRepository) Ping(ctx context.Context) error {
	return ctx.Err()
}

// copyDeployment creates a shallow copy of a deployment.
// Note: This does not deep copy nested structures like Metadata.
// For production use with mutations, consider implementing deep copy.
func copyDeployment(d *models.APIDeployment) *models.APIDeployment {
	if d == nil {
		return nil
	}

	return &models.APIDeployment{
		ID:        d.ID,
		Name:      d.Name,
		Version:   d.Version,
		Context:   d.Context,
		Status:    d.Status,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		Metadata:  d.Metadata,
	}
}
