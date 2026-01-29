package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
	vectordbEntity "github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
	vectordbUsecase "github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
)

// PollingInteractor handles polling protocol operations
type PollingInteractor struct {
	taskRepo         domain.TaskRepository
	memoryInteractor *vectordbUsecase.MemoryInteractor
}

// NewPollingInteractor creates a new polling interactor
func NewPollingInteractor(
	taskRepo domain.TaskRepository,
	memoryInteractor *vectordbUsecase.MemoryInteractor,
) *PollingInteractor {
	return &PollingInteractor{
		taskRepo:         taskRepo,
		memoryInteractor: memoryInteractor,
	}
}

// HandlePollingInput represents input for polling
type HandlePollingInput struct {
	CharacterID    uuid.UUID
	CompletedTasks []CompletedTaskItem
}

// CompletedTaskItem represents a completed task from external service
type CompletedTaskItem struct {
	TaskID string
	DType  string
	Data   interface{}
}

// HandlePollingOutput represents output from polling
type HandlePollingOutput struct {
	NewTasks []NewTaskItem
}

// NewTaskItem represents a new task to be processed
type NewTaskItem struct {
	TaskID string
	Type   string
	DType  string
	Data   interface{}
	Meta   map[string]interface{}
}

// HandlePolling processes completed tasks and returns new pending tasks
func (i *PollingInteractor) HandlePolling(ctx context.Context, input HandlePollingInput) (*HandlePollingOutput, error) {
	// 1. Process completed tasks
	if err := i.processCompletedTasks(ctx, input.CompletedTasks); err != nil {
		return nil, fmt.Errorf("failed to process completed tasks: %w", err)
	}

	// 2. Get pending tasks for the character
	pendingTasks, err := i.getPendingTasks(ctx, input.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending tasks: %w", err)
	}

	// 3. Convert to output format
	newTasks := make([]NewTaskItem, 0, len(pendingTasks))
	for _, task := range pendingTasks {
		newTasks = append(newTasks, NewTaskItem{
			TaskID: task.ID.String(),
			Type:   string(task.Type),
			DType:  "text", // Default, can be extracted from task.Input if needed
			Data:   task.Input,
			Meta: map[string]interface{}{
				"createdAt":   task.CreatedAt,
				"characterId": task.CharacterID.String(),
			},
		})
	}

	return &HandlePollingOutput{
		NewTasks: newTasks,
	}, nil
}

// processCompletedTasks processes results from external service
func (i *PollingInteractor) processCompletedTasks(ctx context.Context, completedTasks []CompletedTaskItem) error {
	for _, item := range completedTasks {
		taskID, err := uuid.Parse(item.TaskID)
		if err != nil {
			// Log error but continue processing other tasks
			continue
		}

		// Get task
		task, err := i.taskRepo.Get(ctx, taskID)
		if err != nil {
			// Task not found, skip
			continue
		}

		// Parse result data based on task type
		switch task.Type {
		case entity.TaskTypeEmbedding:
			if err := i.processEmbeddingResult(ctx, task, item.Data); err != nil {
				// Mark task as failed
				task.MarkFailed(err)
				_ = i.taskRepo.Update(ctx, task)
				continue
			}
		default:
			// Unknown task type, mark as completed anyway
			outputMap, _ := item.Data.(map[string]interface{})
			if outputMap == nil {
				outputMap = map[string]interface{}{"result": item.Data}
			}
			task.MarkCompleted(outputMap)
		}

		// Update task status
		if err := i.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to update task %s: %w", taskID, err)
		}
	}

	return nil
}

// processEmbeddingResult processes embedding task result and stores in VectorDB
func (i *PollingInteractor) processEmbeddingResult(ctx context.Context, task *entity.Task, data interface{}) error {
	// Parse embedding result
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding result: %w", err)
	}

	var embeddingOutput struct {
		DenseVector  []float32          `json:"denseVector"`
		SparseVector map[uint32]float32 `json:"sparseVector,omitempty"`
		Model        string             `json:"model,omitempty"`
		TokenCount   int                `json:"tokenCount,omitempty"`
	}

	if err := json.Unmarshal(dataBytes, &embeddingOutput); err != nil {
		return fmt.Errorf("failed to unmarshal embedding result: %w", err)
	}

	// Validate vectors
	if len(embeddingOutput.DenseVector) == 0 {
		return fmt.Errorf("dense vector is required")
	}

	// Get original input
	inputBytes, err := json.Marshal(task.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal task input: %w", err)
	}

	var embeddingInput entity.EmbeddingTaskInput
	if err := json.Unmarshal(inputBytes, &embeddingInput); err != nil {
		return fmt.Errorf("failed to unmarshal embedding input: %w", err)
	}

	// Store in VectorDB if requested
	if embeddingInput.StoreInDB {
		putInput := vectordbUsecase.PutMemoryInput{
			CharacterID:  task.CharacterID,
			Data:         embeddingInput.Text,
			DType:        vectordbEntity.DType(embeddingInput.DType),
			DenseVector:  embeddingOutput.DenseVector,
			SparseVector: embeddingOutput.SparseVector,
			Metadata:     embeddingInput.Metadata,
		}

		putOutput, err := i.memoryInteractor.PutMemory(ctx, putInput)
		if err != nil {
			return fmt.Errorf("failed to store in vector db: %w", err)
		}

		// Store fragment ID in output
		fragmentID := putOutput.Fragment.ID
		task.MarkCompleted(map[string]interface{}{
			"denseVector":  embeddingOutput.DenseVector,
			"sparseVector": embeddingOutput.SparseVector,
			"fragmentId":   fragmentID.String(),
			"model":        embeddingOutput.Model,
			"tokenCount":   embeddingOutput.TokenCount,
		})
	} else {
		// Just return vectors without storing
		task.MarkCompleted(map[string]interface{}{
			"denseVector":  embeddingOutput.DenseVector,
			"sparseVector": embeddingOutput.SparseVector,
			"model":        embeddingOutput.Model,
			"tokenCount":   embeddingOutput.TokenCount,
		})
	}

	return nil
}

// getPendingTasks retrieves pending tasks for a character
func (i *PollingInteractor) getPendingTasks(ctx context.Context, characterID uuid.UUID) ([]*entity.Task, error) {
	// List all tasks for the character
	allTasks, err := i.taskRepo.ListByCharacter(ctx, characterID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter pending tasks
	pendingTasks := make([]*entity.Task, 0)
	for _, task := range allTasks {
		if task.Status == entity.TaskStatusPending {
			pendingTasks = append(pendingTasks, task)
		}
	}

	return pendingTasks, nil
}
