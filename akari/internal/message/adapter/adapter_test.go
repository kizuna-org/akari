package adapter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	databaseMock "github.com/kizuna-org/akari/pkg/database/domain/mock"
	discordMock "github.com/kizuna-org/akari/pkg/discord/usecase/interactor/mock"
	llmMock "github.com/kizuna-org/akari/pkg/llm/usecase/interactor/mock"
	"github.com/stretchr/testify/assert"
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
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDiscordMsgRepo := databaseMock.NewMockDiscordMessageRepository(ctrl)
			testCase.setupMock(mockDiscordMsgRepo)

			repo := adapter.NewResponseRepository(mockDiscordMsgRepo)
			err := repo.SaveResponse(t.Context(), testCase.resp)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

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

func TestLLMRepository_GenerateResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		systemPrompt string
		userMessage  string
		setupMock    func(*llmMock.MockLLMInteractor)
		want         string
		wantErr      bool
	}{
		{
			name:         "success",
			systemPrompt: "system prompt",
			userMessage:  "Hello user",
			setupMock: func(m *llmMock.MockLLMInteractor) {
				response := "Hello response"
				responses := []*string{&response}
				m.EXPECT().SendChatMessage(gomock.Any(), "system prompt", nil, "Hello user", nil).
					Return(responses, nil, nil)
			},
			want:    "Hello response",
			wantErr: false,
		},
		{
			name:         "no response",
			systemPrompt: "system prompt",
			userMessage:  "Hello user",
			setupMock: func(m *llmMock.MockLLMInteractor) {
				responses := []*string{}
				m.EXPECT().SendChatMessage(gomock.Any(), "system prompt", nil, "Hello user", nil).
					Return(responses, nil, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			name:         "llm error",
			systemPrompt: "system prompt",
			userMessage:  "Hello user",
			setupMock: func(m *llmMock.MockLLMInteractor) {
				m.EXPECT().SendChatMessage(gomock.Any(), "system prompt", nil, "Hello user", nil).
					Return(nil, nil, errors.New("llm error"))
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockLLMInteractor := llmMock.NewMockLLMInteractor(ctrl)
			testCase.setupMock(mockLLMInteractor)

			repo := adapter.NewLLMRepository(mockLLMInteractor)
			result, err := repo.GenerateResponse(t.Context(), testCase.systemPrompt, testCase.userMessage)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.want, result)
			}
		})
	}
}

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
