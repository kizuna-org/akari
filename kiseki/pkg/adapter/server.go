package adapter

import (
	"net/http"
	"time"

	"github.com/kizuna-org/akari/kiseki/gen"
	characterAdapter "github.com/kizuna-org/akari/kiseki/pkg/character/adapter"
	vectordbAdapter "github.com/kizuna-org/akari/kiseki/pkg/vectordb/adapter"
	"github.com/labstack/echo/v4"
)

// Server implements the full ServerInterface by composing individual handlers
type Server struct {
	characterHandler *characterAdapter.Handler
	memoryHandler    *vectordbAdapter.Handler
}

// NewServer creates a new server with all handlers
func NewServer(characterHandler *characterAdapter.Handler, memoryHandler *vectordbAdapter.Handler) *Server {
	return &Server{
		characterHandler: characterHandler,
		memoryHandler:    memoryHandler,
	}
}

// Character endpoints - delegate to character handler
func (s *Server) ListCharacters(ctx echo.Context) error {
	return s.characterHandler.ListCharacters(ctx)
}

func (s *Server) CreateCharacter(ctx echo.Context) error {
	return s.characterHandler.CreateCharacter(ctx)
}

func (s *Server) DeleteCharacter(ctx echo.Context, characterId gen.CharacterIdPath) error {
	return s.characterHandler.DeleteCharacter(ctx, characterId)
}

func (s *Server) GetCharacter(ctx echo.Context, characterId gen.CharacterIdPath) error {
	return s.characterHandler.GetCharacter(ctx, characterId)
}

func (s *Server) UpdateCharacter(ctx echo.Context, characterId gen.CharacterIdPath) error {
	return s.characterHandler.UpdateCharacter(ctx, characterId)
}

// Memory endpoints - delegate to memory handler
func (s *Server) GetMemoryIO(ctx echo.Context, characterId gen.CharacterIdPath, params gen.GetMemoryIOParams) error {
	return s.memoryHandler.GetMemoryIO(ctx, characterId, params)
}

func (s *Server) PutMemoryIO(ctx echo.Context, characterId gen.CharacterIdPath) error {
	return s.memoryHandler.PutMemoryIO(ctx, characterId)
}

func (s *Server) PostMemorySleep(ctx echo.Context, characterId gen.CharacterIdPath) error {
	return ctx.JSON(http.StatusNotImplemented, gen.Error{
		Code:    "NOT_IMPLEMENTED",
		Message: "Memory sleep not yet implemented",
	})
}

func (s *Server) PostMemoryPolling(ctx echo.Context, characterId gen.CharacterIdPath) error {
	return ctx.JSON(http.StatusNotImplemented, gen.Error{
		Code:    "NOT_IMPLEMENTED",
		Message: "Memory polling not yet implemented",
	})
}

// Health check
func (s *Server) GetMemoryHealth(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, gen.HealthResponse{
		Status:    gen.Healthy,
		Timestamp: timePtr(time.Now()),
		Version:   stringPtr("0.1.0"),
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}
