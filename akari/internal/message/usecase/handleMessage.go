package usecase

//go:generate go tool mockgen -package=mock -source=handleMessage.go -destination=mock/handleMessage.go

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
)

type HandleMessageInteractor interface {
	discordService.HandleMessageInteractor
}

type HandleMessageConfig struct {
	CharacterRepo      domain.CharacterRepository
	DiscordRepo        domain.DiscordRepository
	DiscordUserRepo    domain.DiscordUserRepository
	DiscordMessageRepo domain.DiscordMessageRepository
	DiscordChannelRepo domain.DiscordChannelRepository
	DiscordGuildRepo   domain.DiscordGuildRepository
	LLMRepo            domain.LLMRepository
	SystemPromptRepo   domain.SystemPromptRepository
	ValidationRepo     domain.ValidationRepository
	DefaultCharacterID int
	DefaultPromptIndex int
}

type handleMessageInteractorImpl struct {
	characterRepo      domain.CharacterRepository
	discordRepo        domain.DiscordRepository
	discordUserRepo    domain.DiscordUserRepository
	discordMessageRepo domain.DiscordMessageRepository
	discordChannelRepo domain.DiscordChannelRepository
	discordGuildRepo   domain.DiscordGuildRepository
	llmRepo            domain.LLMRepository
	systemPromptRepo   domain.SystemPromptRepository
	validationRepo     domain.ValidationRepository
	defaultCharacterID int
	defaultPromptIndex int
	botUserID          string
}

func NewHandleMessageInteractor(config HandleMessageConfig) discordService.HandleMessageInteractor {
	return &handleMessageInteractorImpl{
		characterRepo:      config.CharacterRepo,
		discordRepo:        config.DiscordRepo,
		discordUserRepo:    config.DiscordUserRepo,
		discordMessageRepo: config.DiscordMessageRepo,
		discordChannelRepo: config.DiscordChannelRepo,
		discordGuildRepo:   config.DiscordGuildRepo,
		llmRepo:            config.LLMRepo,
		systemPromptRepo:   config.SystemPromptRepo,
		validationRepo:     config.ValidationRepo,
		defaultCharacterID: config.DefaultCharacterID,
		defaultPromptIndex: config.DefaultPromptIndex,
		botUserID:          "",
	}
}

func (i *handleMessageInteractorImpl) SetBotUserID(botUserID string) {
	i.botUserID = botUserID
}

func (i *handleMessageInteractorImpl) Handle(
	ctx context.Context,
	discordParams *discordService.DiscordData,
) error {
	if discordParams == nil {
		return errors.New("usecase: discord parameter is nil")
	}

	if err := i.saveDiscordEntities(ctx, discordParams); err != nil {
		return err
	}

	domainUser := entity.ToDiscordUser(discordParams.User)

	domainMessage := entity.ToDiscordMessage(discordParams.Message)

	character, err := i.characterRepo.Get(ctx, i.defaultCharacterID)
	if err != nil {
		return fmt.Errorf("usecase: failed to get character for validation: %w", err)
	}

	var botNameRegex *regexp.Regexp

	if character.NameRegExp != nil && *character.NameRegExp != "" {
		var regexErr error

		botNameRegex, regexErr = regexp.Compile(*character.NameRegExp)
		if regexErr != nil {
			return fmt.Errorf("usecase: invalid bot name regex pattern: %w", regexErr)
		}
	}

	if !i.validationRepo.ShouldProcessMessage(
		domainUser,
		domainMessage,
		discordParams.Mentions,
		i.botUserID,
		botNameRegex,
	) {
		return nil
	}

	return i.generateAndSendResponse(ctx, domainMessage)
}

func (i *handleMessageInteractorImpl) saveDiscordEntities(
	ctx context.Context,
	params *discordService.DiscordData,
) error {
	if params.User != nil {
		if _, err := i.discordUserRepo.CreateIfNotExists(ctx, entity.ToDiscordUser(params.User)); err != nil {
			return fmt.Errorf("usecase: get or create user: %w", err)
		}
	}

	if params.Guild != nil {
		if _, err := i.discordGuildRepo.CreateIfNotExists(ctx, entity.ToDiscordGuild(params.Guild)); err != nil {
			return fmt.Errorf("usecase: create discord guild if not exists: %w", err)
		}
	}

	if params.Channel != nil {
		if _, err := i.discordChannelRepo.CreateIfNotExists(ctx, entity.ToDiscordChannel(params.Channel)); err != nil {
			return fmt.Errorf("usecase: create discord channel if not exists: %w", err)
		}
	}

	if params.Message != nil {
		if err := i.discordMessageRepo.SaveMessage(ctx, entity.ToDiscordMessage(params.Message)); err != nil {
			return fmt.Errorf("usecase: save message: %w", err)
		}
	}

	return nil
}

func (i *handleMessageInteractorImpl) generateAndSendResponse(
	ctx context.Context,
	message *entity.DiscordMessage,
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
