package entity_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestMessage_Structure(t *testing.T) {
	t.Parallel()

	timestamp := time.Now()

	msg := &entity.Message{
		ID:        "msg-123",
		ChannelID: "channel-456",
		GuildID:   "guild-789",
		AuthorID:  "user-001",
		Content:   "Test message",
		Timestamp: timestamp,
	}

	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, "channel-456", msg.ChannelID)
	assert.Equal(t, "guild-789", msg.GuildID)
	assert.Equal(t, "user-001", msg.AuthorID)
	assert.Equal(t, "Test message", msg.Content)
	assert.Equal(t, timestamp, msg.Timestamp)
}

func TestMessage_EmptyValues(t *testing.T) {
	t.Parallel()

	msg := &entity.Message{}

	assert.Empty(t, msg.ID)
	assert.Empty(t, msg.ChannelID)
	assert.Empty(t, msg.GuildID)
	assert.Empty(t, msg.AuthorID)
	assert.Empty(t, msg.Content)
	assert.True(t, msg.Timestamp.IsZero())
}

func TestMessage_Timestamps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		timestamp time.Time
	}{
		{
			name:      "current time",
			timestamp: time.Now(),
		},
		{
			name:      "past time",
			timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "zero time",
			timestamp: time.Time{},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			msg := &entity.Message{
				ID:        "msg-001",
				ChannelID: "channel-001",
				Timestamp: testCase.timestamp,
			}

			assert.Equal(t, testCase.timestamp, msg.Timestamp)
		})
	}
}
