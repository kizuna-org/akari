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
