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

func (r *validationRepository) ShouldProcessMessage(
	message *domain.Message,
	botUserID string,
	botNamePatternRegex *regexp.Regexp,
) bool {
	if message == nil || message.Content == "" {
		return false
	}

	if message.IsBot {
		return false
	}

	if botNamePatternRegex == nil {
		return slices.Contains(message.Mentions, botUserID)
	}

	return slices.Contains(message.Mentions, botUserID) || botNamePatternRegex.MatchString(message.Content)
}
