package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/gen"
	taskEntity "github.com/kizuna-org/akari/kiseki/pkg/task/domain/entity"
	taskUsecase "github.com/kizuna-org/akari/kiseki/pkg/task/usecase"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
	"github.com/labstack/echo/v4"
)

// Handler handles memory-related HTTP requests
type Handler struct {
	memoryInteractor *usecase.MemoryInteractor
	taskInteractor   *taskUsecase.TaskInteractor
}

// NewHandler creates a new memory handler
func NewHandler(memoryInteractor *usecase.MemoryInteractor, taskInteractor *taskUsecase.TaskInteractor) *Handler {
	return &Handler{
		memoryInteractor: memoryInteractor,
		taskInteractor:   taskInteractor,
	}
}

// GetMemoryIO handles GET /characters/{characterId}/memory
func (h *Handler) GetMemoryIO(ctx echo.Context, characterID gen.CharacterIdPath, params gen.GetMemoryIOParams) error {
	// Parse character ID
	charUUID, err := uuid.Parse(characterID.String())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_CHARACTER_ID",
			Message: "Invalid character ID format",
		})
	}

	// Parse data as JSON to extract vectors
	// Expected format: {"query": "...", "denseVector": [...], "sparseVector": {...}}
	var searchData struct {
		Query        string             `json:"query"`
		DenseVector  []float32          `json:"denseVector"`
		SparseVector map[uint32]float32 `json:"sparseVector"`
		Limit        *int               `json:"limit,omitempty"`
	}

	if err := json.Unmarshal([]byte(params.Data), &searchData); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_DATA_FORMAT",
			Message: fmt.Sprintf("Invalid data format: %v", err),
		})
	}

	// Validate vectors
	if len(searchData.DenseVector) == 0 {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "MISSING_DENSE_VECTOR",
			Message: "Dense vector is required",
		})
	}

	// Set default limit
	limit := 10
	if searchData.Limit != nil && *searchData.Limit > 0 {
		limit = *searchData.Limit
	}

	// Call usecase
	input := usecase.GetMemoryInput{
		CharacterID:  charUUID,
		Query:        searchData.Query,
		DenseVector:  searchData.DenseVector,
		SparseVector: searchData.SparseVector,
		Limit:        limit,
	}

	output, err := h.memoryInteractor.GetMemory(ctx.Request().Context(), input)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "SEARCH_FAILED",
			Message: fmt.Sprintf("Failed to search memory: %v", err),
		})
	}

	// Convert to response format
	items := make([]gen.DataType, len(output.Fragments))
	for i, fragment := range output.Fragments {
		// Marshal data to JSON for DataType_Data union type
		dataBytes, err := json.Marshal(fragment.Fragment.Data)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, gen.Error{
				Code:    "MARSHAL_ERROR",
				Message: fmt.Sprintf("Failed to marshal fragment data: %v", err),
			})
		}

		var dataUnion gen.DataType_Data
		if err := dataUnion.UnmarshalJSON(dataBytes); err != nil {
			return ctx.JSON(http.StatusInternalServerError, gen.Error{
				Code:    "UNMARSHAL_ERROR",
				Message: fmt.Sprintf("Failed to unmarshal fragment data: %v", err),
			})
		}

		items[i] = gen.DataType{
			DType: gen.DType(fragment.Fragment.DType),
			Data:  dataUnion,
		}
	}

	response := gen.MemoryIOResponse{
		Items: items,
		Meta: map[string]interface{}{
			"count":       len(items),
			"characterId": characterID.String(),
		},
	}

	return ctx.JSON(http.StatusOK, response)
}

// PutMemoryIO handles PUT /characters/{characterId}/memory
// Supports two modes:
// 1. With vectors (synchronous): Store immediately with provided vectors
// 2. Without vectors (asynchronous): Create embedding task, store when complete
func (h *Handler) PutMemoryIO(ctx echo.Context, characterID gen.CharacterIdPath) error {
	// Parse character ID
	charUUID, err := uuid.Parse(characterID.String())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_CHARACTER_ID",
			Message: "Invalid character ID format",
		})
	}

	// Parse request body
	var req gen.MemoryIORequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST_BODY",
			Message: fmt.Sprintf("Invalid request body: %v", err),
		})
	}

	// Parse data as JSON to extract content and optional vectors
	var storeData struct {
		Content      string                 `json:"content"`
		DenseVector  []float32              `json:"denseVector,omitempty"`
		SparseVector map[uint32]float32     `json:"sparseVector,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
		Model        string                 `json:"model,omitempty"` // For async embedding
	}

	// Convert req.Data to JSON string for parsing
	dataBytes, err := json.Marshal(req.Data)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_DATA_FORMAT",
			Message: fmt.Sprintf("Invalid data format: %v", err),
		})
	}

	if err := json.Unmarshal(dataBytes, &storeData); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_DATA_FORMAT",
			Message: fmt.Sprintf("Invalid data format: %v", err),
		})
	}

	// Validate required fields
	if storeData.Content == "" {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "MISSING_CONTENT",
			Message: "Content is required",
		})
	}

	// Check if vectors are provided (synchronous mode)
	if len(storeData.DenseVector) > 0 {
		// Synchronous mode: Store immediately
		input := usecase.PutMemoryInput{
			CharacterID:  charUUID,
			Data:         storeData.Content,
			DType:        entity.DType(req.DType),
			DenseVector:  storeData.DenseVector,
			SparseVector: storeData.SparseVector,
			Metadata:     storeData.Metadata,
		}

		_, err = h.memoryInteractor.PutMemory(ctx.Request().Context(), input)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, gen.Error{
				Code:    "STORE_FAILED",
				Message: fmt.Sprintf("Failed to store memory: %v", err),
			})
		}

		return ctx.NoContent(http.StatusNoContent)
	}

	// Asynchronous mode: Create embedding task
	// Task will generate vectors and store in VectorDB automatically
	taskInput := map[string]interface{}{
		"taskType":  "embedding",
		"text":      storeData.Content,
		"storeInDb": true,
		"dType":     string(req.DType),
		"metadata":  storeData.Metadata,
	}
	
	if storeData.Model != "" {
		taskInput["model"] = storeData.Model
	}

	taskCreateInput := taskUsecase.CreateTaskInput{
		CharacterID: charUUID,
		Type:        taskEntity.TaskTypeEmbedding,
		Input:       taskInput,
	}

	taskOutput, err := h.taskInteractor.CreateTask(ctx.Request().Context(), taskCreateInput)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "TASK_CREATION_FAILED",
			Message: fmt.Sprintf("Failed to create embedding task: %v", err),
		})
	}

	// Return accepted with task ID
	return ctx.JSON(http.StatusAccepted, map[string]interface{}{
		"message": "Embedding task created. Memory will be stored when task completes.",
		"taskId":  taskOutput.Task.ID.String(),
		"status":  string(taskOutput.Task.Status),
	})
}
