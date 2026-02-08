package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
	"github.com/redis/go-redis/v9"
)

const (
	taskQueueKey   = "kiseki:task:queue"
	taskKeyPrefix  = "kiseki:task:"
	tasksByCharKey = "kiseki:tasks:character:"
	taskExpiration = 7 * 24 * time.Hour // Keep tasks for 7 days
)

// Repository implements TaskRepository using Redis
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new Redis task repository
func NewRepository(client *redis.Client) *Repository {
	return &Repository{
		client: client,
	}
}

// Enqueue adds a task to the queue
func (r *Repository) Enqueue(ctx context.Context, task *entity.Task) error {
	// Serialize task to JSON
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Store task data
	taskKey := taskKey(task.ID)
	if err := r.client.Set(ctx, taskKey, taskJSON, taskExpiration).Err(); err != nil {
		return fmt.Errorf("failed to store task: %w", err)
	}

	// Add to queue (sorted set with timestamp as score for FIFO)
	score := float64(task.CreatedAt.Unix())
	if err := r.client.ZAdd(ctx, taskQueueKey, redis.Z{
		Score:  score,
		Member: task.ID.String(),
	}).Err(); err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	// Add to character's task list
	charTasksKey := tasksByCharacterKey(task.CharacterID)
	if err := r.client.ZAdd(ctx, charTasksKey, redis.Z{
		Score:  score,
		Member: task.ID.String(),
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to character tasks: %w", err)
	}

	return nil
}

// Dequeue retrieves the next pending task from the queue
func (r *Repository) Dequeue(ctx context.Context) (*entity.Task, error) {
	// Get the oldest task from queue
	result, err := r.client.ZPopMin(ctx, taskQueueKey, 1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No tasks available
		}
		return nil, fmt.Errorf("failed to dequeue task: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No tasks available
	}

	// Parse task ID
	taskID, err := uuid.Parse(result[0].Member.(string))
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	// Get task data
	task, err := r.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// Get retrieves a task by ID
func (r *Repository) Get(ctx context.Context, taskID uuid.UUID) (*entity.Task, error) {
	taskKey := taskKey(taskID)

	taskJSON, err := r.client.Get(ctx, taskKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task entity.Task
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// Update updates a task's status and data
func (r *Repository) Update(ctx context.Context, task *entity.Task) error {
	// Serialize task to JSON
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Update task data
	taskKey := taskKey(task.ID)
	if err := r.client.Set(ctx, taskKey, taskJSON, taskExpiration).Err(); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// ListByCharacter retrieves all tasks for a character
func (r *Repository) ListByCharacter(ctx context.Context, characterID uuid.UUID, limit int) ([]*entity.Task, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	charTasksKey := tasksByCharacterKey(characterID)

	// Get task IDs (newest first)
	taskIDs, err := r.client.ZRevRange(ctx, charTasksKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Retrieve each task
	tasks := make([]*entity.Task, 0, len(taskIDs))
	for _, idStr := range taskIDs {
		taskID, err := uuid.Parse(idStr)
		if err != nil {
			continue // Skip invalid IDs
		}

		task, err := r.Get(ctx, taskID)
		if err != nil {
			continue // Skip tasks that couldn't be retrieved
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Delete removes a task from storage
func (r *Repository) Delete(ctx context.Context, taskID uuid.UUID) error {
	// Get task first to remove from character's list
	task, err := r.Get(ctx, taskID)
	if err != nil {
		return err
	}

	// Remove from character's task list
	charTasksKey := tasksByCharacterKey(task.CharacterID)
	if err := r.client.ZRem(ctx, charTasksKey, taskID.String()).Err(); err != nil {
		return fmt.Errorf("failed to remove from character tasks: %w", err)
	}

	// Delete task data
	taskKey := taskKey(taskID)
	if err := r.client.Del(ctx, taskKey).Err(); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// Helper functions
func taskKey(taskID uuid.UUID) string {
	return fmt.Sprintf("%s%s", taskKeyPrefix, taskID.String())
}

func tasksByCharacterKey(characterID uuid.UUID) string {
	return fmt.Sprintf("%s%s", tasksByCharKey, characterID.String())
}
