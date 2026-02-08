package adapter

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/gen"
	"github.com/kizuna-org/akari/kiseki/pkg/task/usecase"
	"github.com/labstack/echo/v4"
)

// Handler handles task-related HTTP requests
type Handler struct {
	taskInteractor    *usecase.TaskInteractor
	pollingInteractor *usecase.PollingInteractor
}

// NewHandler creates a new task handler
func NewHandler(taskInteractor *usecase.TaskInteractor, pollingInteractor *usecase.PollingInteractor) *Handler {
	return &Handler{
		taskInteractor:    taskInteractor,
		pollingInteractor: pollingInteractor,
	}
}

// PostMemoryPolling handles POST /characters/{characterId}/task
// External services submit completed task results and receive new tasks
func (h *Handler) PostMemoryPolling(ctx echo.Context, characterID gen.CharacterIdPath) error {
	// Parse character ID
	charUUID, err := uuid.Parse(characterID.String())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_CHARACTER_ID",
			Message: "Invalid character ID format",
		})
	}

	// Parse request body
	var req gen.MemoryPollingRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST_BODY",
			Message: fmt.Sprintf("Invalid request body: %v", err),
		})
	}

	// Convert request items to completed tasks
	completedTasks := make([]usecase.CompletedTaskItem, 0, len(req.Items))
	for _, item := range req.Items {
		completedTasks = append(completedTasks, usecase.CompletedTaskItem{
			TaskID: item.TaskId,
			DType:  string(item.DType),
			Data:   item.Data,
		})
	}

	// Handle polling (process completed tasks and get new ones)
	input := usecase.HandlePollingInput{
		CharacterID:    charUUID,
		CompletedTasks: completedTasks,
	}

	output, err := h.pollingInteractor.HandlePolling(ctx.Request().Context(), input)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "POLLING_FAILED",
			Message: fmt.Sprintf("Failed to handle polling: %v", err),
		})
	}

	// Group new tasks by type
	tasksByType := make(map[string][]gen.PollingResponseItem)
	for _, task := range output.NewTasks {
		// Convert data to appropriate union format
		var dataUnion gen.PollingResponseItem_Data
		if taskData, ok := task.Data.(map[string]interface{}); ok {
			dataUnion.FromPollingResponseItemData3(taskData)
		}
		
		item := gen.PollingResponseItem{
			TaskId: task.TaskID,
			DType:  gen.DType(task.DType),
			Data:   dataUnion,
			Meta:   task.Meta,
		}

		tasksByType[task.Type] = append(tasksByType[task.Type], item)
	}

	// Convert to response groups
	groups := make([]gen.PollingResponseGroup, 0, len(tasksByType))
	for tType, items := range tasksByType {
		groups = append(groups, gen.PollingResponseGroup{
			TType: tType,
			Items: items,
		})
	}

	response := gen.MemoryPollingResponse{
		Items: groups,
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetTask retrieves a task by ID (custom endpoint, not in OpenAPI yet)
func (h *Handler) GetTask(ctx echo.Context) error {
	// Parse task ID from path parameter
	taskIDStr := ctx.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_TASK_ID",
			Message: "Invalid task ID format",
		})
	}

	// Get task
	input := usecase.GetTaskInput{
		TaskID: taskID,
	}

	output, err := h.taskInteractor.GetTask(ctx.Request().Context(), input)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, gen.Error{
			Code:    "TASK_NOT_FOUND",
			Message: fmt.Sprintf("Task not found: %v", err),
		})
	}

	// Convert task to response
	response := map[string]interface{}{
		"id":          output.Task.ID.String(),
		"characterId": output.Task.CharacterID.String(),
		"type":        string(output.Task.Type),
		"status":      string(output.Task.Status),
		"input":       output.Task.Input,
		"createdAt":   output.Task.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if output.Task.Output != nil {
		response["output"] = output.Task.Output
	}
	if output.Task.Error != "" {
		response["error"] = output.Task.Error
	}
	if output.Task.StartedAt != nil {
		response["startedAt"] = output.Task.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if output.Task.CompletedAt != nil {
		response["completedAt"] = output.Task.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListTasks lists tasks for a character (custom endpoint, not in OpenAPI yet)
func (h *Handler) ListTasks(ctx echo.Context) error {
	// Parse character ID from path parameter
	characterIDStr := ctx.Param("characterId")
	characterID, err := uuid.Parse(characterIDStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_CHARACTER_ID",
			Message: "Invalid character ID format",
		})
	}

	// Get limit from query parameter
	limit := 100
	if limitStr := ctx.QueryParam("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// List tasks
	input := usecase.ListTasksInput{
		CharacterID: characterID,
		Limit:       limit,
	}

	output, err := h.taskInteractor.ListTasks(ctx.Request().Context(), input)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "LIST_TASKS_FAILED",
			Message: fmt.Sprintf("Failed to list tasks: %v", err),
		})
	}

	// Convert tasks to response
	tasks := make([]map[string]interface{}, len(output.Tasks))
	for i, task := range output.Tasks {
		taskMap := map[string]interface{}{
			"id":        task.ID.String(),
			"type":      string(task.Type),
			"status":    string(task.Status),
			"createdAt": task.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if task.CompletedAt != nil {
			taskMap["completedAt"] = task.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		tasks[i] = taskMap
	}

	response := map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}

	return ctx.JSON(http.StatusOK, response)
}
