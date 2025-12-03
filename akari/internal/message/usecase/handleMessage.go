package usecase

//go:generate go tool mockgen -package=mock -source=handleMessage.go -destination=mock/handleMessage.go

import (
	"context"
	"fmt"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain"
)

type HandleMessageInteractor interface {
	Handle(ctx context.Context, message *domain.Message) error
}

type handleMessageInteractorImpl struct {
	messageRepo    domain.MessageRepository
	responseRepo   domain.ResponseRepository
	llmRepo        domain.LLMRepository
	discordRepo    domain.DiscordRepository
	validationRepo domain.ValidationRepository
}

func NewHandleMessageInteractor(
	messageRepo domain.MessageRepository,
	responseRepo domain.ResponseRepository,
	llmRepo domain.LLMRepository,
	discordRepo domain.DiscordRepository,
	validationRepo domain.ValidationRepository,
) HandleMessageInteractor {
	return &handleMessageInteractorImpl{
		messageRepo:    messageRepo,
		responseRepo:   responseRepo,
		llmRepo:        llmRepo,
		discordRepo:    discordRepo,
		validationRepo: validationRepo,
	}
}

func (i *handleMessageInteractorImpl) Handle(ctx context.Context, message *domain.Message) error {
	if !i.validationRepo.ShouldProcessMessage(message) {
		return nil
	}

	if err := i.messageRepo.SaveMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	responseContent, err := i.llmRepo.GenerateResponse(ctx, "You are a helpful Discord bot assistant.", message.Content)
	if err != nil {
		return fmt.Errorf("failed to generate response: %w", err)
	}

	if err := i.discordRepo.SendMessage(ctx, message.ChannelID, responseContent); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	response := &domain.Response{
		ID:        "",
		ChannelID: message.ChannelID,
		MessageID: message.ID,
		Content:   responseContent,
		CreatedAt: time.Now(),
	}

	if err := i.responseRepo.SaveResponse(ctx, response); err != nil {
		return fmt.Errorf("failed to save response: %w", err)
	}

	return nil
}
