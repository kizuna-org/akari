package adapter

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/gen"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain/entity"
	"github.com/kizuna-org/akari/kiseki/pkg/character/usecase"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Handler handles character-related HTTP requests
type Handler struct {
	interactor *usecase.CharacterInteractor
}

// NewHandler creates a new character handler
func NewHandler(interactor *usecase.CharacterInteractor) *Handler {
	return &Handler{
		interactor: interactor,
	}
}

// ListCharacters handles GET /characters
func (h *Handler) ListCharacters(ctx echo.Context) error {
	output, err := h.interactor.ListCharacters(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to list characters",
		})
	}

	response := gen.CharacterListResponse{
		Items: make([]gen.Character, len(output.Characters)),
	}

	for i, char := range output.Characters {
		response.Items[i] = toGenCharacter(char)
	}

	return ctx.JSON(http.StatusOK, response)
}

// CreateCharacter handles POST /characters
func (h *Handler) CreateCharacter(ctx echo.Context) error {
	var req gen.CreateCharacterRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	if req.Name == "" {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Character name is required",
		})
	}

	output, err := h.interactor.CreateCharacter(ctx.Request().Context(), usecase.CreateCharacterInput{
		Name: req.Name,
	})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create character",
		})
	}

	return ctx.JSON(http.StatusCreated, toGenCharacter(output.Character))
}

// GetCharacter handles GET /characters/{characterId}
func (h *Handler) GetCharacter(ctx echo.Context, characterID gen.CharacterIdPath) error {
	id, err := uuid.Parse(characterID.String())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Invalid character ID",
		})
	}

	output, err := h.interactor.GetCharacter(ctx.Request().Context(), usecase.GetCharacterInput{
		ID: id,
	})
	if err != nil {
		return ctx.JSON(http.StatusNotFound, gen.Error{
			Code:    "NOT_FOUND",
			Message: "Character not found",
		})
	}

	return ctx.JSON(http.StatusOK, toGenCharacter(output.Character))
}

// UpdateCharacter handles PUT /characters/{characterId}
func (h *Handler) UpdateCharacter(ctx echo.Context, characterID gen.CharacterIdPath) error {
	id, err := uuid.Parse(characterID.String())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Invalid character ID",
		})
	}

	var req gen.UpdateCharacterRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	if req.Name == nil || *req.Name == "" {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Character name is required",
		})
	}

	output, err := h.interactor.UpdateCharacter(ctx.Request().Context(), usecase.UpdateCharacterInput{
		ID:   id,
		Name: *req.Name,
	})
	if err != nil {
		if err.Error() == "failed to get character: character not found" {
			return ctx.JSON(http.StatusNotFound, gen.Error{
				Code:    "NOT_FOUND",
				Message: "Character not found",
			})
		}
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to update character",
		})
	}

	return ctx.JSON(http.StatusOK, toGenCharacter(output.Character))
}

// DeleteCharacter handles DELETE /characters/{characterId}
func (h *Handler) DeleteCharacter(ctx echo.Context, characterID gen.CharacterIdPath) error {
	id, err := uuid.Parse(characterID.String())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{
			Code:    "INVALID_REQUEST",
			Message: "Invalid character ID",
		})
	}

	err = h.interactor.DeleteCharacter(ctx.Request().Context(), usecase.DeleteCharacterInput{
		ID: id,
	})
	if err != nil {
		if err.Error() == "character not found" || err.Error() == "failed to check character existence: character not found" {
			return ctx.JSON(http.StatusNotFound, gen.Error{
				Code:    "NOT_FOUND",
				Message: "Character not found",
			})
		}
		return ctx.JSON(http.StatusInternalServerError, gen.Error{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to delete character",
		})
	}

	return ctx.NoContent(http.StatusNoContent)
}

// toGenCharacter converts entity.Character to gen.Character
func toGenCharacter(char *entity.Character) gen.Character {
	return gen.Character{
		Id:        openapi_types.UUID(char.ID),
		Name:      char.Name,
		CreatedAt: char.CreatedAt,
		UpdatedAt: char.UpdatedAt,
	}
}
