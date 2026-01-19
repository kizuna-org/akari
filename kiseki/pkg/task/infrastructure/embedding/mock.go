package embedding

import (
	"context"
	"fmt"
	"math/rand"
)

// MockService is a mock implementation of EmbeddingService for testing
type MockService struct {
	vectorSize int
}

// NewMockService creates a new mock embedding service
func NewMockService(vectorSize int) *MockService {
	return &MockService{
		vectorSize: vectorSize,
	}
}

// GenerateEmbedding generates mock dense and sparse vectors
func (s *MockService) GenerateEmbedding(ctx context.Context, text string, model string) ([]float32, map[uint32]float32, int, error) {
	if text == "" {
		return nil, nil, 0, fmt.Errorf("text is empty")
	}

	// Generate mock dense vector
	denseVector := make([]float32, s.vectorSize)
	for i := range denseVector {
		denseVector[i] = rand.Float32()
	}

	// Generate mock sparse vector (10 random indices)
	sparseVector := make(map[uint32]float32)
	for i := 0; i < 10; i++ {
		idx := uint32(rand.Intn(s.vectorSize))
		sparseVector[idx] = rand.Float32()
	}

	// Mock token count (rough estimate: words * 1.3)
	tokenCount := len(text) / 4 // Rough character to token ratio

	return denseVector, sparseVector, tokenCount, nil
}
