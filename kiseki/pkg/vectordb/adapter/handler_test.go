package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/gen"
	"github.com/kizuna-org/akari/kiseki/pkg/config"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
	"github.com/labstack/echo/v4"
)

// mockVectorDBRepo is a mock implementation for testing
type mockVectorDBRepo struct{}

func (m *mockVectorDBRepo) Upsert(ctx context.Context, fragment entity.Fragment, denseVector []float32, sparseVector map[uint32]float32) error {
	return nil
}

func (m *mockVectorDBRepo) HybridSearch(ctx context.Context, characterID uuid.UUID, denseVector []float32, sparseVector map[uint32]float32, limit int) ([]entity.SearchResult, error) {
	// Return mock results
	return []entity.SearchResult{
		{
			Fragment: entity.Fragment{
				ID:          uuid.New(),
				CharacterID: characterID,
				Data:        "Test fragment",
				DType:       entity.DTypeText,
			},
			Score: 0.95,
		},
	}, nil
}

func (m *mockVectorDBRepo) Delete(ctx context.Context, fragmentID uuid.UUID) error {
	return nil
}

func (m *mockVectorDBRepo) EnsureCollection(ctx context.Context, characterID uuid.UUID) error {
	return nil
}

// mockKVSRepo is a mock implementation for testing
type mockKVSRepo struct{}

func (m *mockKVSRepo) IncrementAccess(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	return nil
}

func (m *mockKVSRepo) GetAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) (*entity.AccessInfo, error) {
	return &entity.AccessInfo{
		FragmentID:  fragmentID,
		CharacterID: characterID,
		AccessCount: 5,
	}, nil
}

func (m *mockKVSRepo) GetBatchAccessInfo(ctx context.Context, characterID uuid.UUID, fragmentIDs []uuid.UUID) (map[uuid.UUID]*entity.AccessInfo, error) {
	result := make(map[uuid.UUID]*entity.AccessInfo)
	for _, id := range fragmentIDs {
		result[id] = &entity.AccessInfo{
			FragmentID:  id,
			CharacterID: characterID,
			AccessCount: 5,
		}
	}
	return result, nil
}

func (m *mockKVSRepo) UpdateAccessTime(ctx context.Context, characterID uuid.UUID, fragmentID uuid.UUID) error {
	return nil
}

func (m *mockKVSRepo) InitializeAccessInfo(ctx context.Context, info entity.AccessInfo) error {
	return nil
}

func TestHandler_GetMemoryIO(t *testing.T) {
	// Setup
	cfg := config.Config{
		Score: config.ScoreConfig{
			Alpha:   0.5,
			Beta:    0.3,
			Gamma:   0.2,
			Epsilon: 0.1,
		},
	}
	interactor := usecase.NewMemoryInteractor(&mockVectorDBRepo{}, &mockKVSRepo{}, cfg)
	// For unit tests, taskInteractor can be nil since we're testing with vectors provided
	handler := NewHandler(interactor, nil)

	characterID := uuid.New()
	denseVector := make([]float32, 768)
	for i := range denseVector {
		denseVector[i] = 0.1
	}

	searchData := map[string]interface{}{
		"query":       "test query",
		"denseVector": denseVector,
		"limit":       5,
	}
	searchDataJSON, _ := json.Marshal(searchData)

	tests := []struct {
		name           string
		characterID    string
		dType          string
		data           string
		expectedStatus int
	}{
		{
			name:           "Valid request",
			characterID:    characterID.String(),
			dType:          "text",
			data:           string(searchDataJSON),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid character ID",
			characterID:    "invalid-uuid",
			dType:          "text",
			data:           string(searchDataJSON),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid data format",
			characterID:    characterID.String(),
			dType:          "text",
			data:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/characters/"+tt.characterID+"/memory", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			charID, _ := uuid.Parse(tt.characterID)
			params := gen.GetMemoryIOParams{
				DType: gen.DType(tt.dType),
				Data:  tt.data,
			}

			err := handler.GetMemoryIO(c, charID, params)

			if tt.expectedStatus == http.StatusOK {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestHandler_PutMemoryIO(t *testing.T) {
	// Setup
	cfg := config.Config{
		Score: config.ScoreConfig{
			Alpha:   0.5,
			Beta:    0.3,
			Gamma:   0.2,
			Epsilon: 0.1,
		},
	}
	interactor := usecase.NewMemoryInteractor(&mockVectorDBRepo{}, &mockKVSRepo{}, cfg)
	// For unit tests, taskInteractor can be nil
	handler := NewHandler(interactor, nil)

	characterID := uuid.New()
	denseVector := make([]float32, 768)
	for i := range denseVector {
		denseVector[i] = 0.1
	}

	storeData := map[string]interface{}{
		"content":     "test content",
		"denseVector": denseVector,
		"metadata": map[string]interface{}{
			"source": "test",
		},
	}

	// Convert storeData to proper union type
	dataBytes, _ := json.Marshal(storeData)
	var dataUnion gen.MemoryIORequest_Data
	_ = dataUnion.UnmarshalJSON(dataBytes)

	tests := []struct {
		name           string
		characterID    string
		requestBody    gen.MemoryIORequest
		expectedStatus int
	}{
		{
			name:        "Valid request",
			characterID: characterID.String(),
			requestBody: gen.MemoryIORequest{
				DType: gen.Text,
				Data:  dataUnion,
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:        "Invalid character ID",
			characterID: "invalid-uuid",
			requestBody: gen.MemoryIORequest{
				DType: gen.Text,
				Data:  dataUnion,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/characters/"+tt.characterID+"/memory", bytes.NewReader(bodyBytes))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			charID, _ := uuid.Parse(tt.characterID)

			err := handler.PutMemoryIO(c, charID)

			if tt.expectedStatus == http.StatusNoContent {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
