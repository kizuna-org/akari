package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
)

// TaskRepository defines the interface for task persistence
type TaskRepository interface {
	// Enqueue adds a task to the queue
	Enqueue(ctx context.Context, task *entity.Task) error

	// Dequeue retrieves the next pending task from the queue
	Dequeue(ctx context.Context) (*entity.Task, error)

	// Get retrieves a task by ID
	Get(ctx context.Context, taskID uuid.UUID) (*entity.Task, error)

	// Update updates a task's status and data
	Update(ctx context.Context, task *entity.Task) error

	// ListByCharacter retrieves all tasks for a character
	ListByCharacter(ctx context.Context, characterID uuid.UUID, limit int) ([]*entity.Task, error)

	// Delete removes a task from storage
	Delete(ctx context.Context, taskID uuid.UUID) error
}

// EmbeddingService defines the interface for embedding generation
type EmbeddingService interface {
	// GenerateEmbedding generates dense and sparse vectors for text
	GenerateEmbedding(ctx context.Context, text string, model string) (denseVector []float32, sparseVector map[uint32]float32, tokenCount int, err error)
}
