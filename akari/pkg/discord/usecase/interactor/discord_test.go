package interactor_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	servicemock "github.com/kizuna-org/akari/pkg/discord/domain/service/mock"
	"github.com/kizuna-org/akari/pkg/discord/usecase/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDiscordInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockService := servicemock.NewMockDiscordService(ctrl)
	inter := interactor.NewDiscordInteractor(mockService)

	assert.NotNil(t, inter)
}

func TestDiscordInteractor_SendMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID string
		content   string
		mockSetup func(*servicemock.MockDiscordService)
		wantErr   bool
	}{
		{
			name:      "success",
			channelID: "123",
			content:   "Hello",
			mockSetup: func(m *servicemock.MockDiscordService) {
				m.EXPECT().SendMessage(gomock.Any(), "123", "Hello").
					Return(&entity.Message{
						ID:        "msg-1",
						ChannelID: "123",
						Content:   "Hello",
						Timestamp: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "service error",
			channelID: "123",
			content:   "Hello",
			mockSetup: func(m *servicemock.MockDiscordService) {
				m.EXPECT().SendMessage(gomock.Any(), "123", "Hello").Return(nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockService := servicemock.NewMockDiscordService(ctrl)
			testCase.mockSetup(mockService)

			inter := interactor.NewDiscordInteractor(mockService)
			msg, err := inter.SendMessage(t.Context(), testCase.channelID, testCase.content)

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

func TestDiscordInteractor_GetMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		channelID string
		messageID string
		mockSetup func(*servicemock.MockDiscordService)
		wantErr   bool
	}{
		{
			name:      "success",
			channelID: "123",
			messageID: "msg-1",
			mockSetup: func(m *servicemock.MockDiscordService) {
				m.EXPECT().GetMessage(gomock.Any(), "123", "msg-1").
					Return(&entity.Message{
						ID:        "msg-1",
						ChannelID: "123",
						Timestamp: time.Now(),
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "service error",
			channelID: "123",
			messageID: "msg-1",
			mockSetup: func(m *servicemock.MockDiscordService) {
				m.EXPECT().GetMessage(gomock.Any(), "123", "msg-1").Return(nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockService := servicemock.NewMockDiscordService(ctrl)
			testCase.mockSetup(mockService)

			inter := interactor.NewDiscordInteractor(mockService)
			msg, err := inter.GetMessage(t.Context(), testCase.channelID, testCase.messageID)

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
