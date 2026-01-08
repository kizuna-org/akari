package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/config"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
)

// mockVectorDBRepository is a mock implementation for testing
type mockVectorDBRepository struct {
	upsertFunc           func(ctx context.Context, fragment entity.Fragment, denseVector []float32, sparseVector map[uint32]float32) error
	hybridSearchFunc     func(ctx context.Context, characterID uuid.UUID, denseVector []float32, sparseVector map[uint32]float32, limit int) ([]entity.SearchResult, error)
	deleteFunc           func(ctx context.Context, fragmentID uuid.UUID) error
	ensureCollectionFunc func(ctx context.Context, characterID uuid.UUID) error
}

func (m *mockVectorDBRepository) Upsert(ctx context.Context, fragment entity.Fragment, denseVector []float32, sparseVector map[uint32]float32) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, fragment, denseVector, sparseVector)
	}
	return nil
}

func (m *mockVectorDBRepository) HybridSearch(ctx context.Context, characterID uuid.UUID, denseVector []float32, sparseVector map[uint32]float32, limit int) ([]entity.SearchResult, error) {
	if m.hybridSearchFunc != nil {
		return m.hybridSearchFunc(ctx, characterID, denseVector, sparseVector, limit)
	}
	return []entity.SearchResult{}, nil
}

func (m *mockVectorDBRepository) Delete(ctx context.Context, fragmentID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, fragmentID)
	}
	return nil
}

func (m *mockVectorDBRepository) EnsureCollection(ctx context.Context, characterID uuid.UUID) error {
	if m.ensureCollectionFunc != nil {
		return m.ensureCollectionFunc(ctx, characterID)
	}
	return nil
}

// mockKVSRepository is a mock implementation for testing
type mockKVSRepository struct {
	incrementAccessFunc      func(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error
	getAccessInfoFunc        func(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) (*entity.AccessInfo, error)
	getBatchAccessInfoFunc   func(ctx context.Context, characterID uuid.UUID, fragmentIDs []uuid.UUID) (map[uuid.UUID]*entity.AccessInfo, error)
	updateAccessTimeFunc     func(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error
	initializeAccessInfoFunc func(ctx context.Context, info entity.AccessInfo) error
}

func (m *mockKVSRepository) IncrementAccess(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	if m.incrementAccessFunc != nil {
		return m.incrementAccessFunc(ctx, characterID, fragmentID)
	}
	return nil
}

func (m *mockKVSRepository) GetAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) (*entity.AccessInfo, error) {
	if m.getAccessInfoFunc != nil {
		return m.getAccessInfoFunc(ctx, characterID, fragmentID)
	}
	return &entity.AccessInfo{}, nil
}

func (m *mockKVSRepository) GetBatchAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentIDs []uuid.UUID) (map[uuid.UUID]*entity.AccessInfo, error) {
	if m.getBatchAccessInfoFunc != nil {
		return m.getBatchAccessInfoFunc(ctx, characterID, fragmentIDs)
	}
	return make(map[uuid.UUID]*entity.AccessInfo), nil
}

func (m *mockKVSRepository) UpdateAccessTime(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	if m.updateAccessTimeFunc != nil {
		return m.updateAccessTimeFunc(ctx, characterID, fragmentID)
	}
	return nil
}

func (m *mockKVSRepository) InitializeAccessInfo(ctx context.Context, info entity.AccessInfo) error {
	if m.initializeAccessInfoFunc != nil {
		return m.initializeAccessInfoFunc(ctx, info)
	}
	return nil
}

func TestMemoryInteractor_GetMemory(t *testing.T) {
	characterID := uuid.New()
	fragmentID := uuid.New()

	tests := []struct {
		name              string
		input             GetMemoryInput
		mockSearchResults []entity.SearchResult
		mockAccessInfo    map[uuid.UUID]*entity.AccessInfo
		wantMinResults    int
		wantMaxResults    int
		wantErr           bool
	}{
		{
			name: "successful search with results",
			input: GetMemoryInput{
				CharacterID:  characterID,
				Query:        "test query",
				DenseVector:  []float32{0.1, 0.2, 0.3},
				SparseVector: map[uint32]float32{1: 0.5},
				Limit:        5,
			},
			mockSearchResults: []entity.SearchResult{
				{
					Fragment: entity.Fragment{
						ID:          fragmentID,
						CharacterID: characterID,
						Data:        "test data",
						DType:       entity.DTypeText,
					},
					SemanticScore: 0.9,
				},
			},
			mockAccessInfo: map[uuid.UUID]*entity.AccessInfo{
				fragmentID: {
					FragmentID:     fragmentID,
					CharacterID:    characterID,
					AccessCount:    5,
					LastAccessedAt: time.Now().Add(-1 * time.Hour),
				},
			},
			wantMinResults: 1,
			wantMaxResults: 5,
			wantErr:        false,
		},
		{
			name: "empty search results",
			input: GetMemoryInput{
				CharacterID:  characterID,
				Query:        "nonexistent",
				DenseVector:  []float32{0.1, 0.2, 0.3},
				SparseVector: map[uint32]float32{},
				Limit:        5,
			},
			mockSearchResults: []entity.SearchResult{},
			mockAccessInfo:    map[uuid.UUID]*entity.AccessInfo{},
			wantMinResults:    0,
			wantMaxResults:    0,
			wantErr:           false,
		},
		{
			name: "default limit when zero",
			input: GetMemoryInput{
				CharacterID:  characterID,
				Query:        "test",
				DenseVector:  []float32{0.1},
				SparseVector: map[uint32]float32{},
				Limit:        0, // Should default to 10
			},
			mockSearchResults: []entity.SearchResult{},
			mockAccessInfo:    map[uuid.UUID]*entity.AccessInfo{},
			wantMinResults:    0,
			wantMaxResults:    0,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			vectorDBRepo := &mockVectorDBRepository{
				hybridSearchFunc: func(ctx context.Context, characterID uuid.UUID, denseVector []float32, sparseVector map[uint32]float32, limit int) ([]entity.SearchResult, error) {
					return tt.mockSearchResults, nil
				},
			}

			kvsRepo := &mockKVSRepository{
				getBatchAccessInfoFunc: func(ctx context.Context, characterID uuid.UUID, fragmentIDs []uuid.UUID) (map[uuid.UUID]*entity.AccessInfo, error) {
					return tt.mockAccessInfo, nil
				},
			}

			cfg := config.Config{
				Score: config.ScoreConfig{
					Alpha:   0.5,
					Beta:    0.3,
					Gamma:   0.2,
					Epsilon: 0.1,
				},
			}

			interactor := NewMemoryInteractor(vectorDBRepo, kvsRepo, cfg)

			// Execute
			got, err := interactor.GetMemory(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMemory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify results
			if len(got.Fragments) < tt.wantMinResults || len(got.Fragments) > tt.wantMaxResults {
				t.Errorf("GetMemory() returned %v results, want between %v and %v", len(got.Fragments), tt.wantMinResults, tt.wantMaxResults)
			}
		})
	}
}

func TestMemoryInteractor_PutMemory(t *testing.T) {
	characterID := uuid.New()

	tests := []struct {
		name    string
		input   PutMemoryInput
		wantErr bool
	}{
		{
			name: "successful put",
			input: PutMemoryInput{
				CharacterID:  characterID,
				Data:         "test data",
				DType:        entity.DTypeText,
				DenseVector:  []float32{0.1, 0.2, 0.3},
				SparseVector: map[uint32]float32{1: 0.5},
				Metadata:     map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "put with empty metadata",
			input: PutMemoryInput{
				CharacterID:  characterID,
				Data:         "test data 2",
				DType:        entity.DTypeText,
				DenseVector:  []float32{0.1, 0.2},
				SparseVector: map[uint32]float32{},
				Metadata:     nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			vectorDBRepo := &mockVectorDBRepository{
				ensureCollectionFunc: func(ctx context.Context, characterID uuid.UUID) error {
					return nil
				},
				upsertFunc: func(ctx context.Context, fragment entity.Fragment, denseVector []float32, sparseVector map[uint32]float32) error {
					return nil
				},
			}

			kvsRepo := &mockKVSRepository{
				initializeAccessInfoFunc: func(ctx context.Context, info entity.AccessInfo) error {
					return nil
				},
			}

			cfg := config.Config{
				Score: config.ScoreConfig{
					Alpha:   0.5,
					Beta:    0.3,
					Gamma:   0.2,
					Epsilon: 0.1,
				},
			}

			interactor := NewMemoryInteractor(vectorDBRepo, kvsRepo, cfg)

			// Execute
			got, err := interactor.PutMemory(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutMemory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify output
			if got.Fragment.CharacterID != tt.input.CharacterID {
				t.Errorf("PutMemory() CharacterID = %v, want %v", got.Fragment.CharacterID, tt.input.CharacterID)
			}
			if got.Fragment.Data != tt.input.Data {
				t.Errorf("PutMemory() Data = %v, want %v", got.Fragment.Data, tt.input.Data)
			}
			if got.Fragment.DType != tt.input.DType {
				t.Errorf("PutMemory() DType = %v, want %v", got.Fragment.DType, tt.input.DType)
			}
		})
	}
}
