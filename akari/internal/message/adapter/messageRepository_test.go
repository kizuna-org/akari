package adapter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	databaseMock "github.com/kizuna-org/akari/pkg/database/domain/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMessageRepository_SaveMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		msg       *domain.Message
		setupMock func(*databaseMock.MockDiscordMessageRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			msg: &domain.Message{
				ID:        "msg-001",
				ChannelID: "ch-001",
				GuildID:   "guild-001",
				AuthorID:  "user-001",
				Content:   "Hello",
				Timestamp: now,
			},
			setupMock: func(mockDB *databaseMock.MockDiscordMessageRepository) {
				expectedDbMsg := databaseDomain.DiscordMessage{
					ID:        "msg-001",
					ChannelID: "ch-001",
					AuthorID:  "user-001",
					Content:   "Hello",
					Timestamp: now,
					CreatedAt: now,
				}
				mockDB.EXPECT().CreateDiscordMessage(gomock.Any(), expectedDbMsg).Return(&expectedDbMsg, nil)
			},
			wantErr: false,
		},
		{
			name: "database error",
			msg: &domain.Message{
				ID:        "msg-001",
				ChannelID: "ch-001",
				AuthorID:  "user-001",
				Content:   "Hello",
				Timestamp: now,
			},
			setupMock: func(mockDB *databaseMock.MockDiscordMessageRepository) {
				mockDB.EXPECT().CreateDiscordMessage(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "failed to save discord message",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDiscordMsgRepo := databaseMock.NewMockDiscordMessageRepository(ctrl)
			testCase.setupMock(mockDiscordMsgRepo)

			repo := adapter.NewMessageRepository(mockDiscordMsgRepo)
			err := repo.SaveMessage(t.Context(), testCase.msg)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
