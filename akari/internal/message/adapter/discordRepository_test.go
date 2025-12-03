package adapter_test

import (
	"errors"
	"testing"

	"github.com/kizuna-org/akari/internal/message/adapter"
	discordMock "github.com/kizuna-org/akari/pkg/discord/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDiscordRepository_SendMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID string
		content   string
		setupMock func(*discordMock.MockDiscordInteractor)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "success",
			channelID: "ch-001",
			content:   "Hello",
			setupMock: func(m *discordMock.MockDiscordInteractor) {
				m.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hello").Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name:      "discord error",
			channelID: "ch-001",
			content:   "Hello",
			setupMock: func(m *discordMock.MockDiscordInteractor) {
				m.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hello").Return(nil, errors.New("discord error"))
			},
			wantErr: true,
			errMsg:  "failed to send discord message",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDiscordInteractor := discordMock.NewMockDiscordInteractor(ctrl)
			testCase.setupMock(mockDiscordInteractor)

			repo := adapter.NewDiscordRepository(mockDiscordInteractor)
			err := repo.SendMessage(t.Context(), testCase.channelID, testCase.content)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
