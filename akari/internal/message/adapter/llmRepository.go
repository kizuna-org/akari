package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/kizuna-org/akari/internal/message/domain"
	llmInteractor "github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
)

type llmRepository struct {
	llmInteractor llmInteractor.LLMInteractor
}

func NewLLMRepository(llmInteractor llmInteractor.LLMInteractor) domain.LLMRepository {
	return &llmRepository{
		llmInteractor: llmInteractor,
	}
}

func (r *llmRepository) GenerateResponse(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	responses, _, err := r.llmInteractor.SendChatMessage(
		ctx,
		systemPrompt,
		nil,
		userMessage,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("adapter: failed to generate response from LLM: %w", err)
	}

	if len(responses) > 0 && responses[0] != nil {
		return *responses[0], nil
	}

	return "", errors.New("adapter: no response from LLM")
}
