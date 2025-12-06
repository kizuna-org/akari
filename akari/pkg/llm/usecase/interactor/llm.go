package interactor

//go:generate go tool mockgen -package=mock -source=llm.go -destination=mock/llm.go

import (
	"context"

	"github.com/kizuna-org/akari/pkg/llm/domain"
)

type LLMInteractor interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*domain.Content,
		message string,
		functions []domain.Function,
	) ([]*string, []*domain.Part, error)
}

type LLMInteractorImpl struct {
	geminiRepository domain.GeminiRepository
}

func NewLLMInteractor(
	geminiRepository domain.GeminiRepository,
) LLMInteractor {
	return &LLMInteractorImpl{
		geminiRepository: geminiRepository,
	}
}

func (l *LLMInteractorImpl) SendChatMessage(
	ctx context.Context,
	systemPrompt string,
	history []*domain.Content,
	message string,
	functions []domain.Function,
) ([]*string, []*domain.Part, error) {
	return l.geminiRepository.SendChatMessage(ctx, systemPrompt, history, message, functions)
}
