package adapter

import (
	"regexp"
	"slices"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
)

type validationRepository struct{}

func NewValidationRepository() domain.ValidationRepository {
	return &validationRepository{}
}

func (r *validationRepository) ShouldProcessMessage(
	user *entity.DiscordUser,
	message *entity.DiscordMessage,
	mentions []string,
	botUserID string,
	botNamePatternRegex *regexp.Regexp,
) bool {
	if message == nil || message.Content == "" {
		return false
	}

	if user == nil || user.Bot {
		return false
	}

	mentioned := slices.Contains(mentions, botUserID)

	if botNamePatternRegex == nil {
		return mentioned
	}

	return mentioned || botNamePatternRegex.MatchString(message.Content)
}
