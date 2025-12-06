package domain

//go:generate go tool mockgen -package=mock -source=repository.go -destination=mock/repository.go

import (
	"context"
	"regexp"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type Character struct {
	ID              int
	Name            string
	SystemPromptIDs []int
}

type SystemPrompt struct {
	ID     int
	Prompt string
}

type CharacterRepository interface {
	Get(ctx context.Context, characterID int) (*Character, error)
}

type DiscordRepository interface {
	SendMessage(ctx context.Context, channelID string, content string) error
}

type LLMRepository interface {
	GenerateResponse(ctx context.Context, systemPrompt string, userMessage string) (string, error)
}

type SystemPromptRepository interface {
	Get(ctx context.Context, id int) (*SystemPrompt, error)
}

type ValidationRepository interface {
	ShouldProcessMessage(message *entity.Message, botUserID string, botNameRegExp *regexp.Regexp) bool
}
