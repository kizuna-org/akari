package service

import (
	"context"

	"github.com/kizuna-org/akari/pkg/llm/adapter/repository"
	"google.golang.org/genai"
)

type GeminiService interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*genai.Content,
		message string,
		functions []repository.AkariFunction,
	) ([]*string, []*genai.Part, error)
}

type GeminiServiceImpl struct {
	geminiRepository repository.GeminiRepository
}

func NewGeminiService(
	geminiRepository repository.GeminiRepository,
) GeminiService {
	return &GeminiServiceImpl{
		geminiRepository: geminiRepository,
	}
}

func (g *GeminiServiceImpl) SendChatMessage(
	ctx context.Context,
	systemPrompt string,
	history []*genai.Content,
	message string,
	functions []repository.AkariFunction,
) ([]*string, []*genai.Part, error) {
	return g.geminiRepository.SendChatMessage(ctx, systemPrompt, history, message, functions)
}
