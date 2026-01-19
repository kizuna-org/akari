package adapter

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/gen"
	"github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
	"github.com/kizuna-org/akari/kiseki/pkg/task/usecase"
	"github.com/labstack/echo/v4"
)

// CreateTask creates a new task (custom endpoint for direct task creation)
func (h *Handler) CreateTask(ctx echo.Context) error {
	// Parse character ID from path parameter
	characterIDStr := ctx.Param("characterId")
	characterID, err := uuid.Parse(characterIDStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_CHARACTER_ID",
			Message: "Invalid character ID format",
		})
	}

	// Parse request body
	var reqBody map[string]interface{}
	if err := ctx.Bind(&reqBody); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST_BODY",
			Message: fmt.Sprintf("Invalid request body: %v", err),
		})
	}

	// Get task type
	taskTypeStr, ok := reqBody["taskType"].(string)
	if !ok || taskTypeStr == "" {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "MISSING_TASK_TYPE",
			Message: "Task type is required",
		})
	}

	taskType := entity.TaskType(taskTypeStr)

	// Validate task type and input
	switch taskType {
	case entity.TaskTypeEmbedding:
		if text, ok := reqBody["text"].(string); !ok || text == "" {
			return ctx.JSON(http.StatusBadRequest, gen.Error{
				Code:    "MISSING_TEXT",
				Message: "Text is required for embedding task",
			})
		}
	default:
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_TASK_TYPE",
			Message: fmt.Sprintf("Unknown task type: %s", taskType),
		})
	}

	// Create task
	input := usecase.CreateTaskInput{
		CharacterID: characterID,
		Type:        taskType,
		Input:       reqBody,
	}

	output, err := h.taskInteractor.CreateTask(ctx.Request().Context(), input)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "TASK_CREATION_FAILED",
			Message: fmt.Sprintf("Failed to create task: %v", err),
		})
	}

	// Convert task to response
	response := map[string]interface{}{
		"taskId":      output.Task.ID.String(),
		"status":      string(output.Task.Status),
		"type":        string(output.Task.Type),
		"createdAt":   output.Task.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"characterId": output.Task.CharacterID.String(),
	}

	return ctx.JSON(http.StatusAccepted, response)
}
