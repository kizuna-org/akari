package adapter_test

import (
	"regexp"
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestValidationRepository_ShouldProcessMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       *entity.Message
		botUserID string
		botName   string
		want      bool
	}{
		{
			name:      "valid message with bot mentioned",
			msg:       &entity.Message{Content: "Hello", Mentions: []string{"bot-123"}},
			botUserID: "bot-123",
			botName:   "akari",
			want:      true,
		},
		{
			name:      "valid message with bot name",
			msg:       &entity.Message{Content: "Hey akari, how are you?"},
			botUserID: "bot-123",
			botName:   "akari",
			want:      true,
		},
		{
			name:      "empty content",
			msg:       &entity.Message{Content: ""},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "nil message",
			msg:       nil,
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "bot message",
			msg:       &entity.Message{Content: "Hello", IsBot: true},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "no mention and no bot name",
			msg:       &entity.Message{Content: "Hey there"},
			botUserID: "bot-123",
			botName:   "akari",
			want:      false,
		},
		{
			name:      "case insensitive bot name match",
			msg:       &entity.Message{Content: "Hey AKARI, how are you?"},
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
			result := repo.ShouldProcessMessage(testCase.msg, testCase.botUserID, botNameRegex)

			assert.Equal(t, testCase.want, result)
		})
	}
}
