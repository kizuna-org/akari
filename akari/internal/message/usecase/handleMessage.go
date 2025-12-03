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
	messageRepo        domain.MessageRepository
	responseRepo       domain.ResponseRepository
	llmRepo            domain.LLMRepository
	discordRepo        domain.DiscordRepository
	validationRepo     domain.ValidationRepository
	characterRepo      domain.CharacterRepository
	systemPromptRepo   domain.SystemPromptRepository
	defaultCharacterID int
	defaultPromptIndex int
}

func NewHandleMessageInteractor(
	messageRepo domain.MessageRepository,
	responseRepo domain.ResponseRepository,
	llmRepo domain.LLMRepository,
	discordRepo domain.DiscordRepository,
	validationRepo domain.ValidationRepository,
	characterRepo domain.CharacterRepository,
	systemPromptRepo domain.SystemPromptRepository,
	defaultCharacterID int,
	defaultPromptIndex int,
) HandleMessageInteractor {
	return &handleMessageInteractorImpl{
		messageRepo:        messageRepo,
		responseRepo:       responseRepo,
		llmRepo:            llmRepo,
		discordRepo:        discordRepo,
		validationRepo:     validationRepo,
		characterRepo:      characterRepo,
		systemPromptRepo:   systemPromptRepo,
		defaultCharacterID: defaultCharacterID,
		defaultPromptIndex: defaultPromptIndex,
	}
}

func (i *handleMessageInteractorImpl) Handle(ctx context.Context, message *domain.Message) error {
	if !i.validationRepo.ShouldProcessMessage(message) {
		return nil
	}

	if err := i.messageRepo.SaveMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	character, err := i.characterRepo.GetCharacterByID(ctx, i.defaultCharacterID)
	if err != nil {
		return fmt.Errorf("failed to get character: %w", err)
	}

	systemPromptContent := ""

	if len(character.SystemPromptIDs) > i.defaultPromptIndex {
		systemPrompt, err := i.systemPromptRepo.GetSystemPromptByID(ctx, character.SystemPromptIDs[i.defaultPromptIndex])
		if err != nil {
			return fmt.Errorf("failed to get system prompt: %w", err)
		}

		systemPromptContent = systemPrompt.Prompt
	}

	responseContent, err := i.llmRepo.GenerateResponse(ctx, systemPromptContent, message.Content)
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
