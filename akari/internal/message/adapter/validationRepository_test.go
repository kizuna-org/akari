package adapter_test

import (
	"regexp"
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestValidationRepository_ShouldProcessMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		user      *entity.DiscordUser
		msg       *entity.DiscordMessage
		mentions  []string
		botUserID string
		botName   string
		want      bool
	}{
		{
			name:      "valid message with bot mentioned",
			msg:       &entity.DiscordMessage{Content: "Hello"},
			user:      &entity.DiscordUser{Bot: false},
			mentions:  []string{"bot-123"},
			botUserID: "bot-123",
			botName:   "akari",
			want:      true,
		},
		{
			name:      "valid message with bot name",
			msg:       &entity.DiscordMessage{Content: "Hey akari, how are you?"},
			user:      &entity.DiscordUser{Bot: false},
			mentions:  []string{},
			botUserID: "bot-123",
			botName:   "akari",
			want:      true,
		},
		{
			name:      "empty content",
			msg:       &entity.DiscordMessage{Content: ""},
			user:      &entity.DiscordUser{Bot: false},
			mentions:  []string{},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "nil message",
			msg:       nil,
			user:      nil,
			mentions:  []string{},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "bot message",
			msg:       &entity.DiscordMessage{Content: "Hello"},
			user:      &entity.DiscordUser{Bot: true},
			mentions:  []string{},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "no mention and no bot name",
			msg:       &entity.DiscordMessage{Content: "Hey there"},
			user:      &entity.DiscordUser{Bot: false},
			mentions:  []string{},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "case insensitive bot name match",
			msg:       &entity.DiscordMessage{Content: "Hey AKARI, how are you?"},
			user:      &entity.DiscordUser{Bot: false},
			mentions:  []string{},
			botUserID: "bot-123",
			botName:   "(?i)akari",
			want:      true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := adapter.NewValidationRepository()
			botNameRegex := regexp.MustCompile(testCase.botName)
			result := repo.ShouldProcessMessage(testCase.user, testCase.msg, testCase.mentions, testCase.botUserID, botNameRegex)

			assert.Equal(t, testCase.want, result)
		})
	}
}
