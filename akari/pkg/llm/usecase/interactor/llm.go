package interactor

import (
	"context"

	"github.com/kizuna-org/akari/pkg/llm/adapter/repository"
	"github.com/kizuna-org/akari/pkg/llm/domain/service"
	"google.golang.org/genai"
)

type LLMInteractor interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*genai.Content,
		message string,
		functions []repository.AkariFunction,
	) ([]*string, []*genai.Part, error)
}

type LLMInteractorImpl struct {
	geminiService service.GeminiService
}

func NewLLMInteractor(
	geminiService service.GeminiService,
) LLMInteractor {
	return &LLMInteractorImpl{
		geminiService: geminiService,
	}
}

func (l *LLMInteractorImpl) SendChatMessage(
	ctx context.Context,
	systemPrompt string,
	history []*genai.Content,
	message string,
	functions []repository.AkariFunction,
) ([]*string, []*genai.Part, error) {
	return l.geminiService.SendChatMessage(ctx, systemPrompt, history, message, functions)
}
