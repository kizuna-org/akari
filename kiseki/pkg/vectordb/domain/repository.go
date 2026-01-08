package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
)

// VectorDBRepository defines the interface for vector database operations
type VectorDBRepository interface {
	// Upsert inserts or updates a fragment in the vector database
	Upsert(ctx context.Context, fragment entity.Fragment, denseVector []float32, sparseVector map[uint32]float32) error

	// HybridSearch performs hybrid search combining dense and sparse vectors
	// Returns fragments with their semantic scores
	HybridSearch(ctx context.Context, characterID uuid.UUID, denseVector []float32, sparseVector map[uint32]float32, limit int) ([]entity.SearchResult, error)

	// Delete removes a fragment from the vector database
	Delete(ctx context.Context, fragmentID uuid.UUID) error

	// EnsureCollection ensures that the collection for a character exists
	EnsureCollection(ctx context.Context, characterID uuid.UUID) error
}

// KVSRepository defines the interface for key-value store operations
type KVSRepository interface {
	// IncrementAccess increments the access count for a fragment
	IncrementAccess(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error

	// GetAccessInfo retrieves access information for a fragment
	GetAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) (*entity.AccessInfo, error)

	// GetBatchAccessInfo retrieves access information for multiple fragments
	GetBatchAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentIDs []uuid.UUID) (map[uuid.UUID]*entity.AccessInfo, error)

	// UpdateAccessTime updates the last access time for a fragment
	UpdateAccessTime(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error

	// InitializeAccessInfo initializes access information for a new fragment
	InitializeAccessInfo(ctx context.Context, info entity.AccessInfo) error
}
