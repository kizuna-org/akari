package entity

import (
	"time"

	"github.com/google/uuid"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeEmbedding TaskType = "embedding"
	// Future task types can be added here
	// TaskTypeCompletion TaskType = "completion"
	// TaskTypeSummarize  TaskType = "summarize"
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// Task represents a task to be processed asynchronously
type Task struct {
	ID          uuid.UUID              `json:"id"`
	CharacterID uuid.UUID              `json:"characterId"`
	Type        TaskType               `json:"type"`
	Status      TaskStatus             `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	RetryCount  int                    `json:"retryCount"`
	MaxRetries  int                    `json:"maxRetries"`
}

// EmbeddingTaskInput represents input for embedding generation task
type EmbeddingTaskInput struct {
	Text      string                 `json:"text"`
	Model     string                 `json:"model,omitempty"`     // e.g., "text-embedding-ada-002"
	StoreInDB bool                   `json:"storeInDb,omitempty"` // Whether to store result in vector DB
	DType     string                 `json:"dType,omitempty"`     // Data type for storage
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EmbeddingTaskOutput represents output from embedding generation task
type EmbeddingTaskOutput struct {
	DenseVector  []float32          `json:"denseVector"`
	SparseVector map[uint32]float32 `json:"sparseVector,omitempty"`
	FragmentID   *uuid.UUID         `json:"fragmentId,omitempty"` // If stored in DB
	Model        string             `json:"model"`
	TokenCount   int                `json:"tokenCount,omitempty"`
}

// NewTask creates a new task
func NewTask(characterID uuid.UUID, taskType TaskType, input map[string]interface{}) *Task {
	now := time.Now()
	return &Task{
		ID:          uuid.New(),
		CharacterID: characterID,
		Type:        taskType,
		Status:      TaskStatusPending,
		Input:       input,
		CreatedAt:   now,
		RetryCount:  0,
		MaxRetries:  3,
	}
}

// MarkProcessing marks the task as processing
func (t *Task) MarkProcessing() {
	t.Status = TaskStatusProcessing
	now := time.Now()
	t.StartedAt = &now
}

// MarkCompleted marks the task as completed
func (t *Task) MarkCompleted(output map[string]interface{}) {
	t.Status = TaskStatusCompleted
	t.Output = output
	now := time.Now()
	t.CompletedAt = &now
}

// MarkFailed marks the task as failed
func (t *Task) MarkFailed(err error) {
	t.Status = TaskStatusFailed
	t.Error = err.Error()
	now := time.Now()
	t.CompletedAt = &now
}

// CanRetry returns whether the task can be retried
func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// IncrementRetry increments the retry count
func (t *Task) IncrementRetry() {
	t.RetryCount++
	t.Status = TaskStatusPending
	t.StartedAt = nil
}
