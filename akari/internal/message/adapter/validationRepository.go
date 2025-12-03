package adapter

import (
	"regexp"
	"slices"

	"github.com/kizuna-org/akari/internal/message/domain"
)

type validationRepository struct{}

func NewValidationRepository() domain.ValidationRepository {
	return &validationRepository{}
}

func (r *validationRepository) ShouldProcessMessage(message *domain.Message) bool {
	if message == nil || message.Content == "" {
		return false
	}

	if message.IsBot {
		return false
	}

	return true
}

func (r *validationRepository) IsBotMentioned(message *domain.Message, botUserID string) bool {
	return slices.Contains(message.Mentions, botUserID)
}

func (r *validationRepository) ContainsBotName(message *domain.Message, botNamePattern string) bool {
	return regexp.MustCompile(botNamePattern).MatchString(message.Content)
}
