package adapter_test

import (
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidationRepository_ShouldProcessMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  *domain.Message
		want bool
	}{
		{
			name: "valid message",
			msg: &domain.Message{
				Content: "Hello",
			},
			want: true,
		},
		{
			name: "empty content",
			msg: &domain.Message{
				Content: "",
			},
			want: false,
		},
		{
			name: "nil message",
			msg:  nil,
			want: false,
		},
		{
			name: "bot message",
			msg: &domain.Message{
				Content: "Hello",
				IsBot:   true,
			},
			want: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := adapter.NewValidationRepository()
			result := repo.ShouldProcessMessage(testCase.msg)

			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestValidationRepository_IsBotMentioned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       *domain.Message
		botUserID string
		want      bool
	}{
		{
			name: "bot mentioned",
			msg: &domain.Message{
				Mentions: []string{"bot-123", "user-456"},
			},
			botUserID: "bot-123",
			want:      true,
		},
		{
			name: "bot not mentioned",
			msg: &domain.Message{
				Mentions: []string{"user-456", "user-789"},
			},
			botUserID: "bot-123",
			want:      false,
		},
		{
			name: "empty mentions",
			msg: &domain.Message{
				Mentions: []string{},
			},
			botUserID: "bot-123",
			want:      false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := adapter.NewValidationRepository()
			result := repo.IsBotMentioned(testCase.msg, testCase.botUserID)

			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestValidationRepository_ContainsBotName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		msg            *domain.Message
		botNamePattern string
		want           bool
	}{
		{
			name: "contains bot name",
			msg: &domain.Message{
				Content: "Hey akari, how are you?",
			},
			botNamePattern: "akari",
			want:           true,
		},
		{
			name: "does not contain bot name",
			msg: &domain.Message{
				Content: "Hey there, how are you?",
			},
			botNamePattern: "akari",
			want:           false,
		},
		{
			name: "case insensitive match",
			msg: &domain.Message{
				Content: "Hey AKARI, how are you?",
			},
			botNamePattern: "(?i)akari",
			want:           true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := adapter.NewValidationRepository()
			result := repo.ContainsBotName(testCase.msg, testCase.botNamePattern)

			assert.Equal(t, testCase.want, result)
		})
	}
}
