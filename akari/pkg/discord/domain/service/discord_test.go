package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/repository/mock"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDiscordService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockDiscordRepository(ctrl)
	svc := service.NewDiscordService(mockRepo)

	assert.NotNil(t, svc)
}

func TestDiscordService_SendMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID string
		content   string
		mockSetup func(*mock.MockDiscordRepository)
		wantErr   bool
	}{
		{
			name:      "success",
			channelID: "123",
			content:   "Hello",
			mockSetup: func(m *mock.MockDiscordRepository) {
				m.EXPECT().SendMessage(gomock.Any(), "123", "Hello").
					Return(&entity.Message{
						ID:        "msg-1",
						ChannelID: "123",
						GuildID:   "",
						AuthorID:  "",
						Content:   "Hello",
						Timestamp: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "empty channel ID",
			channelID: "",
			content:   "Hello",
			mockSetup: nil,
			wantErr:   true,
		},
		{
			name:      "empty content",
			channelID: "123",
			content:   "",
			mockSetup: nil,
			wantErr:   true,
		},
		{
			name:      "repository error",
			channelID: "123",
			content:   "Hello",
			mockSetup: func(m *mock.MockDiscordRepository) {
				m.EXPECT().SendMessage(gomock.Any(), "123", "Hello").Return(nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockRepo := mock.NewMockDiscordRepository(ctrl)
			if testCase.mockSetup != nil {
				testCase.mockSetup(mockRepo)
			}

			svc := service.NewDiscordService(mockRepo)
			msg, err := svc.SendMessage(t.Context(), testCase.channelID, testCase.content)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, msg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, msg)
				assert.Equal(t, testCase.channelID, msg.ChannelID)
			}
		})
	}
}

func TestDiscordService_GetMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID string
		messageID string
		mockSetup func(*mock.MockDiscordRepository)
		wantErr   bool
	}{
		{
			name:      "success",
			channelID: "123",
			messageID: "msg-1",
			mockSetup: func(m *mock.MockDiscordRepository) {
				m.EXPECT().GetMessage(gomock.Any(), "123", "msg-1").
					Return(&entity.Message{
						ID:        "msg-1",
						ChannelID: "123",
						GuildID:   "",
						AuthorID:  "",
						Content:   "",
						Timestamp: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "empty channel ID",
			channelID: "",
			messageID: "msg-1",
			mockSetup: nil,
			wantErr:   true,
		},
		{
			name:      "empty message ID",
			channelID: "123",
			messageID: "",
			mockSetup: nil,
			wantErr:   true,
		},
		{
			name:      "repository error",
			channelID: "123",
			messageID: "msg-1",
			mockSetup: func(m *mock.MockDiscordRepository) {
				m.EXPECT().GetMessage(gomock.Any(), "123", "msg-1").Return(nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockRepo := mock.NewMockDiscordRepository(ctrl)
			if testCase.mockSetup != nil {
				testCase.mockSetup(mockRepo)
			}

			svc := service.NewDiscordService(mockRepo)
			msg, err := svc.GetMessage(t.Context(), testCase.channelID, testCase.messageID)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, msg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, msg)
				assert.Equal(t, testCase.messageID, msg.ID)
			}
		})
	}
}
