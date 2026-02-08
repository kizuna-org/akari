package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/kizuna-org/akari/kiseki/pkg/task/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
	vdbEntity "github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
	vdbUsecase "github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
)

// Worker processes tasks from the queue
type Worker struct {
	taskRepo         domain.TaskRepository
	embeddingService domain.EmbeddingService
	memoryInteractor *vdbUsecase.MemoryInteractor
	pollInterval     time.Duration
	stopCh           chan struct{}
}

// NewWorker creates a new task worker
func NewWorker(
	taskRepo domain.TaskRepository,
	embeddingService domain.EmbeddingService,
	memoryInteractor *vdbUsecase.MemoryInteractor,
	pollInterval time.Duration,
) *Worker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second // Default poll interval
	}

	return &Worker{
		taskRepo:         taskRepo,
		embeddingService: embeddingService,
		memoryInteractor: memoryInteractor,
		pollInterval:     pollInterval,
		stopCh:           make(chan struct{}),
	}
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	slog.Info("Task worker starting", "pollInterval", w.pollInterval)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Task worker stopped by context")
			return
		case <-w.stopCh:
			slog.Info("Task worker stopped")
			return
		case <-ticker.C:
			if err := w.processNextTask(ctx); err != nil {
				slog.Error("Failed to process task", "error", err)
			}
		}
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.stopCh)
}

// processNextTask processes the next task from the queue
func (w *Worker) processNextTask(ctx context.Context) error {
	// Dequeue next task
	task, err := w.taskRepo.Dequeue(ctx)
	if err != nil {
		return fmt.Errorf("failed to dequeue task: %w", err)
	}

	if task == nil {
		// No tasks available
		return nil
	}

	slog.Info("Processing task", "taskId", task.ID, "type", task.Type, "characterId", task.CharacterID)

	// Mark task as processing
	task.MarkProcessing()
	if err := w.taskRepo.Update(ctx, task); err != nil {
		slog.Error("Failed to mark task as processing", "taskId", task.ID, "error", err)
	}

	// Process task based on type
	var processErr error
	switch task.Type {
	case entity.TaskTypeEmbedding:
		processErr = w.processEmbeddingTask(ctx, task)
	default:
		processErr = fmt.Errorf("unknown task type: %s", task.Type)
	}

	// Update task status
	if processErr != nil {
		slog.Error("Task processing failed", "taskId", task.ID, "error", processErr)
		
		if task.CanRetry() {
			task.IncrementRetry()
			slog.Info("Requeueing task for retry", "taskId", task.ID, "retryCount", task.RetryCount)
			if err := w.taskRepo.Enqueue(ctx, task); err != nil {
				slog.Error("Failed to requeue task", "taskId", task.ID, "error", err)
			}
		} else {
			task.MarkFailed(processErr)
			if err := w.taskRepo.Update(ctx, task); err != nil {
				slog.Error("Failed to mark task as failed", "taskId", task.ID, "error", err)
			}
		}
		return processErr
	}

	slog.Info("Task completed successfully", "taskId", task.ID)
	return nil
}

// processEmbeddingTask processes an embedding generation task
func (w *Worker) processEmbeddingTask(ctx context.Context, task *entity.Task) error {
	// Parse input
	var input entity.EmbeddingTaskInput
	inputJSON, err := json.Marshal(task.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}
	if err := json.Unmarshal(inputJSON, &input); err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	// Validate input
	if input.Text == "" {
		return fmt.Errorf("text is required")
	}

	// Generate embedding
	slog.Info("Generating embedding", "taskId", task.ID, "textLength", len(input.Text))
	denseVector, sparseVector, tokenCount, err := w.embeddingService.GenerateEmbedding(ctx, input.Text, input.Model)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Prepare output
	output := entity.EmbeddingTaskOutput{
		DenseVector:  denseVector,
		SparseVector: sparseVector,
		Model:        input.Model,
		TokenCount:   tokenCount,
	}

	// Store in vector DB if requested
	if input.StoreInDB && w.memoryInteractor != nil {
		slog.Info("Storing embedding in vector DB", "taskId", task.ID)
		
		dtype := vdbEntity.DTypeText
		if input.DType != "" {
			dtype = vdbEntity.DType(input.DType)
		}

		putInput := vdbUsecase.PutMemoryInput{
			CharacterID:  task.CharacterID,
			Data:         input.Text,
			DType:        dtype,
			DenseVector:  denseVector,
			SparseVector: sparseVector,
			Metadata:     input.Metadata,
		}

		putOutput, err := w.memoryInteractor.PutMemory(ctx, putInput)
		if err != nil {
			return fmt.Errorf("failed to store in vector DB: %w", err)
		}

		output.FragmentID = &putOutput.Fragment.ID
		slog.Info("Embedding stored successfully", "taskId", task.ID, "fragmentId", putOutput.Fragment.ID)
	}

	// Convert output to map
	outputMap := make(map[string]interface{})
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}
	if err := json.Unmarshal(outputJSON, &outputMap); err != nil {
		return fmt.Errorf("failed to unmarshal output: %w", err)
	}

	// Mark task as completed
	task.MarkCompleted(outputMap)
	if err := w.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}
