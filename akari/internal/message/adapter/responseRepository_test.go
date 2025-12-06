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

func TestResponseRepository_SaveResponse(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		resp      *domain.Response
		setupMock func(*databaseMock.MockDiscordMessageRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			resp: &domain.Response{
				ID:        "resp-001",
				ChannelID: "ch-001",
				MessageID: "msg-001",
				Content:   "Response",
				CreatedAt: now,
			},
			setupMock: func(mockDB *databaseMock.MockDiscordMessageRepository) {
				expectedDbMsg := databaseDomain.DiscordMessage{
					ID:        "resp-001",
					ChannelID: "ch-001",
					AuthorID:  "",
					Content:   "Response",
					Timestamp: now,
					CreatedAt: now,
				}
				mockDB.EXPECT().CreateDiscordMessage(gomock.Any(), expectedDbMsg).Return(&expectedDbMsg, nil)
			},
			wantErr: false,
		},
		{
			name: "database error",
			resp: &domain.Response{
				ID:        "resp-002",
				ChannelID: "ch-001",
				MessageID: "msg-002",
				Content:   "Response",
				CreatedAt: now,
			},
			setupMock: func(mockDB *databaseMock.MockDiscordMessageRepository) {
				mockDB.EXPECT().CreateDiscordMessage(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:      "nil response",
			resp:      nil,
			setupMock: func(mockDB *databaseMock.MockDiscordMessageRepository) {},
			wantErr:   true,
			errMsg:    "response is nil",
		},
	}

	for _, testCase := range tests {
		testCaseVal := testCase
		t.Run(testCaseVal.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDiscordMsgRepo := databaseMock.NewMockDiscordMessageRepository(ctrl)
			testCaseVal.setupMock(mockDiscordMsgRepo)

			repo := adapter.NewResponseRepository(mockDiscordMsgRepo)
			err := repo.SaveResponse(t.Context(), testCaseVal.resp)

			if testCaseVal.wantErr {
				if testCaseVal.errMsg != "" {
					require.ErrorContains(t, err, testCaseVal.errMsg)
				} else {
					require.Error(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
