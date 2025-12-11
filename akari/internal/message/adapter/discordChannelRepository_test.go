package adapter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/internal/message/adapter"
	"github.com/kizuna-org/akari/internal/message/domain/entity"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordChannelMock "github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDiscordChannelRepository(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockInteractor := discordChannelMock.NewMockDiscordChannelInteractor(ctrl)

	repo := adapter.NewDiscordChannelRepository(mockInteractor)

	require.NotNil(t, repo)
}

func TestDiscordChannelRepository_CreateIfNotExists(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name      string
		channel   *entity.DiscordChannel
		setupMock func(*discordChannelMock.MockDiscordChannelInteractor, context.Context)
		want      string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "channel already exists",
			channel: &entity.DiscordChannel{
				ID:        "ch-001",
				Type:      0,
				Name:      "general",
				GuildID:   "g-001",
				CreatedAt: now,
			},
			setupMock: func(m *discordChannelMock.MockDiscordChannelInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordChannelByID(ctx, "ch-001").
					Return(&databaseDomain.DiscordChannel{
						ID:      "ch-001",
						Type:    "0",
						Name:    "general",
						GuildID: "g-001",
					}, nil)
			},
			want:    "ch-001",
			wantErr: false,
		},
		{
			name: "create new channel",
			channel: &entity.DiscordChannel{
				ID:        "ch-002",
				Type:      0,
				Name:      "random",
				GuildID:   "g-001",
				CreatedAt: now,
			},
			setupMock: func(m *discordChannelMock.MockDiscordChannelInteractor, ctx context.Context) {
				notFoundErr := &ent.NotFoundError{}
				m.EXPECT().
					GetDiscordChannelByID(ctx, "ch-002").
					Return(nil, fmt.Errorf("failed to get discord channel by id: %w", notFoundErr))
				m.EXPECT().
					CreateDiscordChannel(ctx, gomock.Any()).
					Return(&databaseDomain.DiscordChannel{
						ID:      "ch-002",
						Type:    "0",
						Name:    "random",
						GuildID: "g-001",
					}, nil)
			},
			want:    "ch-002",
			wantErr: false,
		},
		{
			name:      "nil channel",
			channel:   nil,
			setupMock: func(m *discordChannelMock.MockDiscordChannelInteractor, ctx context.Context) {},
			want:      "",
			wantErr:   true,
			errMsg:    "channel is required",
		},
		{
			name: "create channel error",
			channel: &entity.DiscordChannel{
				ID:        "ch-003",
				Type:      1,
				Name:      "dm",
				GuildID:   "g-001",
				CreatedAt: now,
			},
			setupMock: func(m *discordChannelMock.MockDiscordChannelInteractor, ctx context.Context) {
				notFoundErr := &ent.NotFoundError{}
				m.EXPECT().
					GetDiscordChannelByID(ctx, "ch-003").
					Return(nil, fmt.Errorf("failed to get discord channel by id: %w", notFoundErr))
				m.EXPECT().
					CreateDiscordChannel(ctx, gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			want:    "",
			wantErr: true,
			errMsg:  "database error",
		},
		{
			name: "database error on get",
			channel: &entity.DiscordChannel{
				ID:        "ch-005",
				Type:      0,
				Name:      "error",
				GuildID:   "g-001",
				CreatedAt: now,
			},
			setupMock: func(m *discordChannelMock.MockDiscordChannelInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordChannelByID(ctx, "ch-005").
					Return(nil, errors.New("connection refused"))
			},
			want:    "",
			wantErr: true,
			errMsg:  "connection refused",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			ctx := t.Context()
			mockInteractor := discordChannelMock.NewMockDiscordChannelInteractor(ctrl)
			testCase.setupMock(mockInteractor, ctx)

			repo := adapter.NewDiscordChannelRepository(mockInteractor)
			got, err := repo.CreateIfNotExists(ctx, testCase.channel)

			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errMsg)
				require.Equal(t, testCase.want, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.want, got)
			}
		})
	}
}
