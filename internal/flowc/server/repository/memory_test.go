package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/pkg/types"
)

func createTestDeployment(id string) *models.APIDeployment {
	return &models.APIDeployment{
		ID:        id,
		Name:      "test-api",
		Version:   "1.0.0",
		Context:   "/api/v1",
		Status:    string(models.StatusDeployed),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: types.FlowCMetadata{
			Name:    "test-api",
			Version: "1.0.0",
			Context: "/api/v1",
		},
	}
}

func TestMemoryRepository_Create(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	deployment := createTestDeployment("test-1")

	// Test successful creation
	err := repo.Create(ctx, deployment)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test duplicate creation
	err = repo.Create(ctx, deployment)
	if err != ErrAlreadyExists {
		t.Errorf("Expected ErrAlreadyExists, got: %v", err)
	}

	// Test nil deployment
	err = repo.Create(ctx, nil)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for nil deployment, got: %v", err)
	}

	// Test empty ID
	emptyID := createTestDeployment("")
	err = repo.Create(ctx, emptyID)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for empty ID, got: %v", err)
	}
}

func TestMemoryRepository_Get(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	deployment := createTestDeployment("test-1")
	_ = repo.Create(ctx, deployment)

	// Test successful get
	retrieved, err := repo.Get(ctx, "test-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != deployment.ID {
		t.Errorf("Expected ID %s, got %s", deployment.ID, retrieved.ID)
	}

	// Test not found
	_, err = repo.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestMemoryRepository_Update(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	deployment := createTestDeployment("test-1")
	_ = repo.Create(ctx, deployment)

	// Test successful update
	deployment.Version = "2.0.0"
	err := repo.Update(ctx, deployment)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, _ := repo.Get(ctx, "test-1")
	if retrieved.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", retrieved.Version)
	}

	// Test update nonexistent
	nonexistent := createTestDeployment("nonexistent")
	err = repo.Update(ctx, nonexistent)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}

	// Test nil deployment
	err = repo.Update(ctx, nil)
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput, got: %v", err)
	}
}

func TestMemoryRepository_Delete(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	deployment := createTestDeployment("test-1")
	_ = repo.Create(ctx, deployment)

	// Test successful delete
	err := repo.Delete(ctx, "test-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = repo.Get(ctx, "test-1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}

	// Test delete nonexistent
	err = repo.Delete(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestMemoryRepository_List(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test empty list
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d items", len(list))
	}

	// Add deployments
	_ = repo.Create(ctx, createTestDeployment("test-1"))
	_ = repo.Create(ctx, createTestDeployment("test-2"))
	_ = repo.Create(ctx, createTestDeployment("test-3"))

	// Test list with items
	list, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 items, got %d", len(list))
	}
}

func TestMemoryRepository_ListByStatus(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Add deployments with different statuses
	deployed1 := createTestDeployment("test-1")
	deployed1.Status = string(models.StatusDeployed)
	_ = repo.Create(ctx, deployed1)

	deployed2 := createTestDeployment("test-2")
	deployed2.Status = string(models.StatusDeployed)
	_ = repo.Create(ctx, deployed2)

	failed := createTestDeployment("test-3")
	failed.Status = string(models.StatusFailed)
	_ = repo.Create(ctx, failed)

	// Test filter by deployed status
	list, err := repo.ListByStatus(ctx, models.StatusDeployed)
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 deployed items, got %d", len(list))
	}

	// Test filter by failed status
	list, err = repo.ListByStatus(ctx, models.StatusFailed)
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 failed item, got %d", len(list))
	}
}

func TestMemoryRepository_Count(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test empty count
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}

	// Add deployments
	_ = repo.Create(ctx, createTestDeployment("test-1"))
	_ = repo.Create(ctx, createTestDeployment("test-2"))

	// Test count
	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestMemoryRepository_Exists(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	_ = repo.Create(ctx, createTestDeployment("test-1"))

	// Test exists
	exists, err := repo.Exists(ctx, "test-1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected exists to be true")
	}

	// Test not exists
	exists, err = repo.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected exists to be false")
	}
}

func TestMemoryRepository_NodeID(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test set and get
	err := repo.SetNodeID(ctx, "dep-1", "node-1")
	if err != nil {
		t.Fatalf("SetNodeID failed: %v", err)
	}

	nodeID, err := repo.GetNodeID(ctx, "dep-1")
	if err != nil {
		t.Fatalf("GetNodeID failed: %v", err)
	}
	if nodeID != "node-1" {
		t.Errorf("Expected node-1, got %s", nodeID)
	}

	// Test not found
	_, err = repo.GetNodeID(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}

	// Test delete
	err = repo.DeleteNodeID(ctx, "dep-1")
	if err != nil {
		t.Fatalf("DeleteNodeID failed: %v", err)
	}

	_, err = repo.GetNodeID(ctx, "dep-1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}

	// Test invalid input
	err = repo.SetNodeID(ctx, "", "node-1")
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for empty deployment ID, got: %v", err)
	}

	err = repo.SetNodeID(ctx, "dep-1", "")
	if err != ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput for empty node ID, got: %v", err)
	}
}

func TestMemoryRepository_GetDeploymentsByNodeID(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Set up mappings
	_ = repo.SetNodeID(ctx, "dep-1", "node-1")
	_ = repo.SetNodeID(ctx, "dep-2", "node-1")
	_ = repo.SetNodeID(ctx, "dep-3", "node-2")

	// Test get deployments by node
	deps, err := repo.GetDeploymentsByNodeID(ctx, "node-1")
	if err != nil {
		t.Fatalf("GetDeploymentsByNodeID failed: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 deployments for node-1, got %d", len(deps))
	}

	deps, err = repo.GetDeploymentsByNodeID(ctx, "node-2")
	if err != nil {
		t.Fatalf("GetDeploymentsByNodeID failed: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("Expected 1 deployment for node-2, got %d", len(deps))
	}

	// Test nonexistent node
	deps, err = repo.GetDeploymentsByNodeID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetDeploymentsByNodeID failed: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 deployments, got %d", len(deps))
	}
}

func TestMemoryRepository_GetStats(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test empty stats
	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("Expected total 0, got %d", stats.Total)
	}

	// Add deployments with different statuses
	deployed := createTestDeployment("test-1")
	deployed.Status = string(models.StatusDeployed)
	_ = repo.Create(ctx, deployed)

	failed := createTestDeployment("test-2")
	failed.Status = string(models.StatusFailed)
	_ = repo.Create(ctx, failed)

	pending := createTestDeployment("test-3")
	pending.Status = string(models.StatusPending)
	_ = repo.Create(ctx, pending)

	// Test stats
	stats, err = repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}
	if stats.Deployed != 1 {
		t.Errorf("Expected deployed 1, got %d", stats.Deployed)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected failed 1, got %d", stats.Failed)
	}
	if stats.Pending != 1 {
		t.Errorf("Expected pending 1, got %d", stats.Pending)
	}
}

func TestMemoryRepository_ContextCancellation(t *testing.T) {
	repo := NewMemoryRepository()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test that cancelled context returns error
	_, err := repo.Get(ctx, "test")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	_, err = repo.List(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	err = repo.Create(ctx, createTestDeployment("test"))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestMemoryRepository_CloseAndPing(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test ping
	err := repo.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test close
	err = repo.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewRepository_Factory(t *testing.T) {
	// Test default config
	repo, err := NewRepository(nil)
	if err != nil {
		t.Fatalf("NewRepository with nil config failed: %v", err)
	}
	if _, ok := repo.(*MemoryRepository); !ok {
		t.Error("Expected MemoryRepository for nil config")
	}

	// Test memory type
	repo, err = NewRepository(&RepositoryConfig{Type: RepositoryTypeMemory})
	if err != nil {
		t.Fatalf("NewRepository with memory config failed: %v", err)
	}
	if _, ok := repo.(*MemoryRepository); !ok {
		t.Error("Expected MemoryRepository")
	}

	// Test unsupported types return error
	_, err = NewRepository(&RepositoryConfig{Type: RepositoryTypePostgres})
	if err == nil {
		t.Error("Expected error for unimplemented postgres repository")
	}

	// Test unknown type
	_, err = NewRepository(&RepositoryConfig{Type: "unknown"})
	if err == nil {
		t.Error("Expected error for unknown repository type")
	}
}
