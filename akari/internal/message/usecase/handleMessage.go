package usecase

//go:generate go tool mockgen -package=mock -source=handleMessage.go -destination=mock/handleMessage.go

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
)

type HandleMessageInteractor interface {
	discordService.HandleMessageInteractor
}

type HandleMessageConfig struct {
	CharacterRepo       domain.CharacterRepository
	DiscordRepo         domain.DiscordRepository
	DiscordUserRepo     domain.DiscordUserRepository
	DiscordMessageRepo  domain.DiscordMessageRepository
	DiscordChannelRepo  domain.DiscordChannelRepository
	DiscordGuildRepo    domain.DiscordGuildRepository
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
	discordUserRepo     domain.DiscordUserRepository
	discordMessageRepo  domain.DiscordMessageRepository
	discordChannelRepo  domain.DiscordChannelRepository
	discordGuildRepo    domain.DiscordGuildRepository
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
		discordUserRepo:     config.DiscordUserRepo,
		discordMessageRepo:  config.DiscordMessageRepo,
		discordChannelRepo:  config.DiscordChannelRepo,
		discordGuildRepo:    config.DiscordGuildRepo,
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

func (i *handleMessageInteractorImpl) Handle(
	ctx context.Context,
	user *discordEntity.User,
	message *discordEntity.Message,
	channel *discordEntity.Channel,
	guild *discordEntity.Guild,
) error {
	if message == nil {
		return errors.New("usecase: message is nil")
	}

	domainMessage := entity.ToMessage(message)

	if user != nil {
		if _, err := i.discordUserRepo.CreateIfNotExists(ctx, entity.ToUser(user)); err != nil {
			return fmt.Errorf("usecase: get or create user: %w", err)
		}
	}

	if guild != nil {
		if _, err := i.discordGuildRepo.CreateIfNotExists(ctx, entity.ToGuild(guild)); err != nil {
			return fmt.Errorf("usecase: create discord guild if not exists: %w", err)
		}
	}

	if channel != nil {
		if _, err := i.discordChannelRepo.CreateIfNotExists(ctx, entity.ToChannel(channel)); err != nil {
			return fmt.Errorf("usecase: create discord channel if not exists: %w", err)
		}
	}

	if err := i.discordMessageRepo.SaveMessage(ctx, domainMessage); err != nil {
		return fmt.Errorf("usecase: save message: %w", err)
	}

	if !i.validationRepo.ShouldProcessMessage(domainMessage, i.botUserID, i.botNamePatternRegex) {
		return nil
	}

	return i.generateAndSendResponse(ctx, domainMessage)
}

func (i *handleMessageInteractorImpl) generateAndSendResponse(
	ctx context.Context,
	message *entity.Message,
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
