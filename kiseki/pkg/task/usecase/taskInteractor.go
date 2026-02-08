package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
)

// TaskInteractor handles task-related operations
type TaskInteractor struct {
	taskRepo domain.TaskRepository
}

// NewTaskInteractor creates a new task interactor
func NewTaskInteractor(taskRepo domain.TaskRepository) *TaskInteractor {
	return &TaskInteractor{
		taskRepo: taskRepo,
	}
}

// CreateTaskInput represents input for creating a task
type CreateTaskInput struct {
	CharacterID uuid.UUID
	Type        entity.TaskType
	Input       map[string]interface{}
}

// CreateTaskOutput represents output from creating a task
type CreateTaskOutput struct {
	Task *entity.Task
}

// CreateTask creates a new task and enqueues it
func (i *TaskInteractor) CreateTask(ctx context.Context, input CreateTaskInput) (*CreateTaskOutput, error) {
	// Create task entity
	task := entity.NewTask(input.CharacterID, input.Type, input.Input)

	// Enqueue task
	if err := i.taskRepo.Enqueue(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	return &CreateTaskOutput{
		Task: task,
	}, nil
}

// GetTaskInput represents input for getting a task
type GetTaskInput struct {
	TaskID uuid.UUID
}

// GetTaskOutput represents output from getting a task
type GetTaskOutput struct {
	Task *entity.Task
}

// GetTask retrieves a task by ID
func (i *TaskInteractor) GetTask(ctx context.Context, input GetTaskInput) (*GetTaskOutput, error) {
	task, err := i.taskRepo.Get(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &GetTaskOutput{
		Task: task,
	}, nil
}

// ListTasksInput represents input for listing tasks
type ListTasksInput struct {
	CharacterID uuid.UUID
	Limit       int
}

// ListTasksOutput represents output from listing tasks
type ListTasksOutput struct {
	Tasks []*entity.Task
}

// ListTasks lists tasks for a character
func (i *TaskInteractor) ListTasks(ctx context.Context, input ListTasksInput) (*ListTasksOutput, error) {
	if input.Limit <= 0 {
		input.Limit = 100
	}

	tasks, err := i.taskRepo.ListByCharacter(ctx, input.CharacterID, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return &ListTasksOutput{
		Tasks: tasks,
	}, nil
}
