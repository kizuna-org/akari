package usecase

//go:generate go tool mockgen -package=mock -source=handleMessage.go -destination=mock/handleMessage.go

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain"
	dbdomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type HandleMessageInteractor interface {
	Handle(ctx context.Context, message *domain.Message) error
	SetBotUserID(botUserID string)
}

type HandleMessageConfig struct {
	MessageRepo           domain.MessageRepository
	ResponseRepo          domain.ResponseRepository
	LLMRepo               domain.LLMRepository
	DiscordRepo           domain.DiscordRepository
	ValidationRepo        domain.ValidationRepository
	CharacterRepo         domain.CharacterRepository
	SystemPromptRepo      domain.SystemPromptRepository
	ConversationRepo      domain.ConversationRepository
	ConversationGroupRepo domain.ConversationGroupRepository
	DiscordUserRepo       domain.DiscordUserRepository
	DefaultCharacterID    int
	DefaultPromptIndex    int
	BotNamePatternRegex   *regexp.Regexp
}

type handleMessageInteractorImpl struct {
	messageRepo           domain.MessageRepository
	responseRepo          domain.ResponseRepository
	llmRepo               domain.LLMRepository
	discordRepo           domain.DiscordRepository
	validationRepo        domain.ValidationRepository
	characterRepo         domain.CharacterRepository
	systemPromptRepo      domain.SystemPromptRepository
	conversationRepo      domain.ConversationRepository
	conversationGroupRepo domain.ConversationGroupRepository
	discordUserRepo       domain.DiscordUserRepository
	defaultCharacterID    int
	defaultPromptIndex    int
	botUserID             string
	botNamePatternRegex   *regexp.Regexp
}

func NewHandleMessageInteractor(config HandleMessageConfig) HandleMessageInteractor {
	return &handleMessageInteractorImpl{
		messageRepo:           config.MessageRepo,
		responseRepo:          config.ResponseRepo,
		llmRepo:               config.LLMRepo,
		discordRepo:           config.DiscordRepo,
		validationRepo:        config.ValidationRepo,
		characterRepo:         config.CharacterRepo,
		systemPromptRepo:      config.SystemPromptRepo,
		conversationRepo:      config.ConversationRepo,
		conversationGroupRepo: config.ConversationGroupRepo,
		discordUserRepo:       config.DiscordUserRepo,
		defaultCharacterID:    config.DefaultCharacterID,
		defaultPromptIndex:    config.DefaultPromptIndex,
		botUserID:             "",
		botNamePatternRegex:   config.BotNamePatternRegex,
	}
}

func (i *handleMessageInteractorImpl) SetBotUserID(botUserID string) {
	i.botUserID = botUserID
}

func (i *handleMessageInteractorImpl) Handle(ctx context.Context, message *domain.Message) error {
	if err := i.messageRepo.SaveMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	if !i.validationRepo.ShouldProcessMessage(message) {
		return nil
	}

	if !i.isMentioned(message) {
		return nil
	}

	userID, err := i.discordUserRepo.GetOrCreateDiscordUser(
		ctx,
		message.AuthorID,
		"",
		message.IsBot,
	)
	if err != nil {
		return fmt.Errorf("failed to get or create discord user: %w", err)
	}

	conversationGroup, err := i.conversationGroupRepo.GetConversationGroupByCharacterID(ctx, i.defaultCharacterID)
	if err != nil && !errors.Is(err, dbdomain.ErrNotFound) {
		return fmt.Errorf("usecase: failed to get conversation group: %w", err)
	}

	if conversationGroup == nil {
		conversationGroup, err = i.conversationGroupRepo.CreateConversationGroup(ctx, i.defaultCharacterID)
		if err != nil {
			return fmt.Errorf("usecase: failed to create conversation group: %w", err)
		}
	}

	if err := i.conversationRepo.CreateConversation(
		ctx,
		message.ID,
		userID,
		&conversationGroup.ID,
	); err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return i.generateAndSendResponse(ctx, message)
}

func (i *handleMessageInteractorImpl) generateAndSendResponse(ctx context.Context, message *domain.Message) error {
	character, err := i.characterRepo.GetCharacterByID(ctx, i.defaultCharacterID)
	if err != nil {
		return fmt.Errorf("failed to get character: %w", err)
	}

	systemPromptContent, err := i.getSystemPromptContent(ctx, character)
	if err != nil {
		return err
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

func (i *handleMessageInteractorImpl) getSystemPromptContent(
	ctx context.Context,
	character *domain.Character,
) (string, error) {
	if len(character.SystemPromptIDs) <= i.defaultPromptIndex {
		return "", nil
	}

	systemPrompt, err := i.systemPromptRepo.GetSystemPromptByID(
		ctx,
		character.SystemPromptIDs[i.defaultPromptIndex],
	)
	if err != nil {
		return "", fmt.Errorf("failed to get system prompt: %w", err)
	}

	return systemPrompt.Prompt, nil
}

func (i *handleMessageInteractorImpl) isMentioned(message *domain.Message) bool {
	return i.validationRepo.IsBotMentioned(message, i.botUserID) ||
		i.validationRepo.ContainsBotName(message, i.botNamePatternRegex)
}
