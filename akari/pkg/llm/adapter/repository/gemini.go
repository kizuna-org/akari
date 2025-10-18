package repository

import (
	"context"

	"github.com/kizuna-org/akari/pkg/llm/infrastructure"
	"google.golang.org/genai"
)

type AkariContext = genai.Content
type AkariFunction = infrastructure.Function

type GeminiRepository interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*AkariContext,
		message string,
		functions []AkariFunction,
	) ([]*string, []*genai.Part, error)
}

type GeminiRepositoryImpl struct {
	geminiModel infrastructure.GeminiModel
}

func NewGeminiRepository(
	geminiModel infrastructure.GeminiModel,
) GeminiRepository {
	return &GeminiRepositoryImpl{
		geminiModel: geminiModel,
	}
}

func (g *GeminiRepositoryImpl) SendChatMessage(
	ctx context.Context,
	systemPrompt string,
	history []*AkariContext,
	message string,
	functions []AkariFunction,
) ([]*string, []*genai.Part, error) {
	return g.geminiModel.SendChatMessage(ctx, systemPrompt, history, message, functions)
}
