package usecase

//go:generate go tool mockgen -package=mock -source=handleMessage.go -destination=mock/handleMessage.go

import (
	"context"
	"fmt"
	"regexp"

	"github.com/kizuna-org/akari/internal/message/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
)

type HandleMessageInteractor interface {
	discordService.HandleMessageInteractor
}

type HandleMessageConfig struct {
	CharacterRepo       domain.CharacterRepository
	DiscordRepo         domain.DiscordRepository
	LLMRepo             domain.LLMRepository
	SystemPromptRepo    domain.SystemPromptRepository
	ValidationRepo      domain.ValidationRepository
	DefaultCharacterID  int
	DefaultPromptIndex  int
	BotNamePatternRegex *regexp.Regexp
}

type handleMessageInteractorImpl struct {
	characterRepo       domain.CharacterRepository
	discordRepo         domain.DiscordRepository
	llmRepo             domain.LLMRepository
	systemPromptRepo    domain.SystemPromptRepository
	validationRepo      domain.ValidationRepository
	defaultCharacterID  int
	defaultPromptIndex  int
	botUserID           string
	botNamePatternRegex *regexp.Regexp
}

func NewHandleMessageInteractor(config HandleMessageConfig) discordService.HandleMessageInteractor {
	return &handleMessageInteractorImpl{
		characterRepo:       config.CharacterRepo,
		discordRepo:         config.DiscordRepo,
		llmRepo:             config.LLMRepo,
		systemPromptRepo:    config.SystemPromptRepo,
		validationRepo:      config.ValidationRepo,
		defaultCharacterID:  config.DefaultCharacterID,
		defaultPromptIndex:  config.DefaultPromptIndex,
		botUserID:           "",
		botNamePatternRegex: config.BotNamePatternRegex,
	}
}

func (i *handleMessageInteractorImpl) SetBotUserID(botUserID string) {
	i.botUserID = botUserID
}

func (i *handleMessageInteractorImpl) Handle(ctx context.Context, message *discordEntity.Message) error {
	domainMessage := domain.ToMessage(message)

	if !i.validationRepo.ShouldProcessMessage(domainMessage, i.botUserID, i.botNamePatternRegex) {
		return nil
	}

	return i.generateAndSendResponse(ctx, domainMessage)
}

func (i *handleMessageInteractorImpl) generateAndSendResponse(
	ctx context.Context,
	message *domain.Message,
) error {
	character, err := i.characterRepo.Get(ctx, i.defaultCharacterID)
	if err != nil {
		return fmt.Errorf("usecase: get character: %w", err)
	}

	systemPromptContent, err := i.getSystemPromptContent(ctx, character)
	if err != nil {
		return fmt.Errorf("usecase: get system prompt content: %w", err)
	}

	responseContent, err := i.llmRepo.GenerateResponse(ctx, systemPromptContent, message.Content)
	if err != nil {
		return fmt.Errorf("usecase: generate response: %w", err)
	}

	if err := i.discordRepo.SendMessage(ctx, message.ChannelID, responseContent); err != nil {
		return fmt.Errorf("usecase: send message: %w", err)
	}

	return nil
}

func (i *handleMessageInteractorImpl) getSystemPromptContent(
	ctx context.Context,
	character *domain.Character,
) (string, error) {
	if len(character.SystemPromptIDs) <= i.defaultPromptIndex {
		return "", nil
	}

	systemPrompt, err := i.systemPromptRepo.Get(
		ctx,
		character.SystemPromptIDs[i.defaultPromptIndex],
	)
	if err != nil {
		return "", fmt.Errorf("usecase: get system prompt: %w", err)
	}

	return systemPrompt.Prompt, nil
}
