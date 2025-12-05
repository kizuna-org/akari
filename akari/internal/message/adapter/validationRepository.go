package adapter

import (
	"regexp"
	"slices"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type validationRepository struct{}

func NewValidationRepository() domain.ValidationRepository {
	return &validationRepository{}
}

func (r *validationRepository) ShouldProcessMessage(
	message *entity.Message,
	botUserID string,
	botNamePatternRegex *regexp.Regexp,
) bool {
	if message == nil || message.Content == "" {
		return false
	}

	if message.IsBot {
		return false
	}

	return slices.Contains(message.Mentions, botUserID) || botNamePatternRegex.MatchString(message.Content)
}
