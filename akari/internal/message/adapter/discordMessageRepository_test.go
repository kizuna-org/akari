package adapter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordMessageMock "github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDiscordMessageRepository(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockInteractor := discordMessageMock.NewMockDiscordMessageInteractor(ctrl)

	repo := adapter.NewDiscordMessageRepository(mockInteractor)

	require.NotNil(t, repo)
}

func TestDiscordMessageRepository_SaveMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name      string
		message   *entity.DiscordMessage
		setupMock func(*discordMessageMock.MockDiscordMessageInteractor)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			message: &entity.DiscordMessage{
				ID:        "msg-001",
				ChannelID: "ch-001",
				AuthorID:  "usr-001",
				Content:   "Hello",
				Timestamp: now,
			},
			setupMock: func(m *discordMessageMock.MockDiscordMessageInteractor) {
				m.EXPECT().
					CreateDiscordMessage(gomock.Any(), gomock.Any()).
					Return(&databaseDomain.DiscordMessage{
						ID:        "msg-001",
						ChannelID: "ch-001",
						AuthorID:  "usr-001",
						Content:   "Hello",
						Timestamp: now,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "interactor error",
			message: &entity.DiscordMessage{
				ID:        "msg-001",
				ChannelID: "ch-001",
				AuthorID:  "usr-001",
				Content:   "Hello",
				Timestamp: now,
			},
			setupMock: func(m *discordMessageMock.MockDiscordMessageInteractor) {
				m.EXPECT().
					CreateDiscordMessage(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "database error",
		},
		{
			name:      "nil message",
			message:   nil,
			setupMock: func(m *discordMessageMock.MockDiscordMessageInteractor) {},
			wantErr:   true,
			errMsg:    "message is nil",
		},
		{
			name: "empty message content",
			message: &entity.DiscordMessage{
				ID:        "msg-001",
				ChannelID: "ch-001",
				AuthorID:  "usr-001",
				Content:   "",
				Timestamp: now,
			},
			setupMock: func(m *discordMessageMock.MockDiscordMessageInteractor) {
				m.EXPECT().
					CreateDiscordMessage(gomock.Any(), gomock.Any()).
					Return(&databaseDomain.DiscordMessage{
						ID:        "msg-001",
						ChannelID: "ch-001",
						AuthorID:  "usr-001",
						Content:   "",
						Timestamp: now,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "message with mentions",
			message: &entity.DiscordMessage{
				ID:        "msg-002",
				ChannelID: "ch-002",
				AuthorID:  "usr-002",
				Content:   "Hello @user1 @user2",
				Timestamp: now,
				Mentions:  []string{"usr-002", "usr-003"},
			},
			setupMock: func(m *discordMessageMock.MockDiscordMessageInteractor) {
				m.EXPECT().
					CreateDiscordMessage(gomock.Any(), gomock.Any()).
					Return(&databaseDomain.DiscordMessage{
						ID:        "msg-002",
						ChannelID: "ch-002",
						AuthorID:  "usr-002",
						Content:   "Hello @user1 @user2",
						Timestamp: now,
					}, nil)
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockInteractor := discordMessageMock.NewMockDiscordMessageInteractor(ctrl)
			testCase.setupMock(mockInteractor)

			repo := adapter.NewDiscordMessageRepository(mockInteractor)
			err := repo.SaveMessage(t.Context(), testCase.message)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
