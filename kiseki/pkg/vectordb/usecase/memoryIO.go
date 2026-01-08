package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/config"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
)

// MemoryInteractor handles memory I/O operations
type MemoryInteractor struct {
	vectorDBRepo domain.VectorDBRepository
	kvsRepo      domain.KVSRepository
	scorer       *Scorer
	config       config.Config
}

// NewMemoryInteractor creates a new memory interactor
func NewMemoryInteractor(
	vectorDBRepo domain.VectorDBRepository,
	kvsRepo domain.KVSRepository,
	config config.Config,
) *MemoryInteractor {
	return &MemoryInteractor{
		vectorDBRepo: vectorDBRepo,
		kvsRepo:      kvsRepo,
		scorer:       NewScorer(config.Score),
		config:       config,
	}
}

// GetMemoryInput represents the input for GetMemory operation
type GetMemoryInput struct {
	CharacterID  uuid.UUID
	Query        string
	DenseVector  []float32
	SparseVector map[uint32]float32
	Limit        int
}

// GetMemoryOutput represents the output for GetMemory operation
type GetMemoryOutput struct {
	Fragments []entity.SearchResult
}

// GetMemory performs hybrid search and returns rescored results
// This is the main search functionality for memory retrieval
func (m *MemoryInteractor) GetMemory(ctx context.Context, input GetMemoryInput) (*GetMemoryOutput, error) {
	if input.Limit <= 0 {
		input.Limit = 10 // Default limit
	}

	// Step 1: Get candidates from Qdrant with higher limit for rescoring
	candidateLimit := input.Limit * 3 // Get 3x candidates for better rescoring
	results, err := m.vectorDBRepo.HybridSearch(
		ctx,
		input.CharacterID,
		input.DenseVector,
		input.SparseVector,
		candidateLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector database: %w", err)
	}

	if len(results) == 0 {
		return &GetMemoryOutput{Fragments: []entity.SearchResult{}}, nil
	}

	// Step 2: Get access information from Redis for all candidates
	fragmentIDs := make([]uuid.UUID, len(results))
	for i, result := range results {
		fragmentIDs[i] = result.Fragment.ID
	}

	accessInfoMap, err := m.kvsRepo.GetBatchAccessInfo(ctx, input.CharacterID, fragmentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get access info: %w", err)
	}

	// Convert map key from UUID to string for scorer
	accessInfoStrMap := make(map[string]*entity.AccessInfo)
	for id, info := range accessInfoMap {
		accessInfoStrMap[id.String()] = info
	}

	// Step 3: Rescore results with popularity and time scores
	results = m.scorer.RescoreResults(results, accessInfoStrMap)

	// Step 4: Sort by final score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Step 5: Trim to requested limit
	if len(results) > input.Limit {
		results = results[:input.Limit]
	}

	// Step 6: Update access information for returned fragments
	for _, result := range results {
		// Increment access count and update timestamp asynchronously
		// We don't wait for this to complete to avoid slowing down the response
		go func(fragmentID uuid.UUID) {
			ctx := context.Background()
			_ = m.kvsRepo.IncrementAccess(ctx, input.CharacterID, fragmentID)
		}(result.Fragment.ID)
	}

	return &GetMemoryOutput{
		Fragments: results,
	}, nil
}

// PutMemoryInput represents the input for PutMemory operation
type PutMemoryInput struct {
	CharacterID  uuid.UUID
	Data         string
	DType        entity.DType
	DenseVector  []float32
	SparseVector map[uint32]float32
	Metadata     map[string]interface{}
}

// PutMemoryOutput represents the output for PutMemory operation
type PutMemoryOutput struct {
	Fragment entity.Fragment
}

// PutMemory stores a new memory fragment
func (m *MemoryInteractor) PutMemory(ctx context.Context, input PutMemoryInput) (*PutMemoryOutput, error) {
	// Create fragment
	now := time.Now()
	fragment := entity.Fragment{
		ID:          uuid.New(),
		CharacterID: input.CharacterID,
		Data:        input.Data,
		DType:       input.DType,
		Metadata:    input.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Step 1: Ensure collection exists
	if err := m.vectorDBRepo.EnsureCollection(ctx, input.CharacterID); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Step 2: Store in Qdrant
	if err := m.vectorDBRepo.Upsert(ctx, fragment, input.DenseVector, input.SparseVector); err != nil {
		return nil, fmt.Errorf("failed to store fragment in vector database: %w", err)
	}

	// Step 3: Initialize access info in Redis
	accessInfo := entity.AccessInfo{
		FragmentID:      fragment.ID,
		CharacterID:     fragment.CharacterID,
		AccessCount:     0,
		LastAccessedAt:  now,
		FirstAccessedAt: now,
	}
	if err := m.kvsRepo.InitializeAccessInfo(ctx, accessInfo); err != nil {
		return nil, fmt.Errorf("failed to initialize access info: %w", err)
	}

	return &PutMemoryOutput{
		Fragment: fragment,
	}, nil
}

// DeleteMemoryInput represents the input for DeleteMemory operation
type DeleteMemoryInput struct {
	CharacterID uuid.UUID
	FragmentID  uuid.UUID
}

// DeleteMemory deletes a memory fragment
func (m *MemoryInteractor) DeleteMemory(ctx context.Context, input DeleteMemoryInput) error {
	// Note: Current implementation limitation - Delete requires additional context
	// In a full implementation, we would need to track which collection a fragment belongs to
	return fmt.Errorf("delete operation not yet implemented")
}
