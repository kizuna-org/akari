package domain

//go:generate go tool mockgen -package=mock -source=repository.go -destination=mock/repository.go

import "context"

type Character struct {
	ID              int
	Name            string
	SystemPromptIDs []int
}

type SystemPrompt struct {
	ID     int
	Prompt string
}

type ConversationGroup struct {
	ID          int
	CharacterID int
}

type MessageRepository interface {
	SaveMessage(ctx context.Context, message *Message) error
}

type ResponseRepository interface {
	SaveResponse(ctx context.Context, response *Response) error
}

type LLMRepository interface {
	GenerateResponse(ctx context.Context, systemPrompt string, userMessage string) (string, error)
}

type DiscordRepository interface {
	SendMessage(ctx context.Context, channelID string, content string) error
}

type ValidationRepository interface {
	ShouldProcessMessage(message *Message) bool
	IsBotMentioned(message *Message, botUserID string) bool
	ContainsBotName(message *Message, botNamePattern string) bool
}

type CharacterRepository interface {
	GetCharacterByID(ctx context.Context, characterID int) (*Character, error)
}

type SystemPromptRepository interface {
	GetSystemPromptByID(ctx context.Context, id int) (*SystemPrompt, error)
}

type ConversationRepository interface {
	CreateConversation(
		ctx context.Context,
		messageID string,
		userID int,
		conversationGroupID *int,
	) error
}

type ConversationGroupRepository interface {
	GetConversationGroupByCharacterID(ctx context.Context, characterID int) (*ConversationGroup, error)
	CreateConversationGroup(ctx context.Context, characterID int) (*ConversationGroup, error)
}

type DiscordUserRepository interface {
	GetDiscordUserByID(ctx context.Context, discordUserID string) (int, error)
	GetOrCreateDiscordUser(
		ctx context.Context,
		discordUserID string,
		username string,
		isBot bool,
	) (int, error)
}

type AkariUserRepository interface {
	GetOrCreateAkariUserByDiscordUserID(ctx context.Context, discordUserID string) (int, error)
}
