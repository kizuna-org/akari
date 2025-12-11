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
	discordGuildMock "github.com/kizuna-org/akari/pkg/database/usecase/interactor/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDiscordGuildRepository(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockInteractor := discordGuildMock.NewMockDiscordGuildInteractor(ctrl)

	repo := adapter.NewDiscordGuildRepository(mockInteractor)

	require.NotNil(t, repo)
}

func TestDiscordGuildRepository_CreateIfNotExists(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name      string
		guild     *entity.DiscordGuild
		setupMock func(*discordGuildMock.MockDiscordGuildInteractor, context.Context)
		want      string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "guild already exists",
			guild: &entity.DiscordGuild{
				ID:        "g-001",
				Name:      "test guild",
				CreatedAt: now,
			},
			setupMock: func(m *discordGuildMock.MockDiscordGuildInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordGuildByID(ctx, "g-001").
					Return(&databaseDomain.DiscordGuild{
						ID:   "g-001",
						Name: "test guild",
					}, nil)
			},
			want:    "g-001",
			wantErr: false,
		},
		{
			name: "create new guild",
			guild: &entity.DiscordGuild{
				ID:        "g-002",
				Name:      "new guild",
				CreatedAt: now,
			},
			setupMock: func(m *discordGuildMock.MockDiscordGuildInteractor, ctx context.Context) {
				notFoundErr := &ent.NotFoundError{}
				m.EXPECT().
					GetDiscordGuildByID(ctx, "g-002").
					Return(nil, fmt.Errorf("failed to get discord guild by id: %w", notFoundErr))
				m.EXPECT().
					CreateDiscordGuild(ctx, gomock.Any()).
					Return(&databaseDomain.DiscordGuild{
						ID:   "g-002",
						Name: "new guild",
					}, nil)
			},
			want:    "g-002",
			wantErr: false,
		},
		{
			name:      "nil guild",
			guild:     nil,
			setupMock: func(m *discordGuildMock.MockDiscordGuildInteractor, ctx context.Context) {},
			want:      "",
			wantErr:   true,
			errMsg:    "guild is required",
		},
		{
			name: "create guild error",
			guild: &entity.DiscordGuild{
				ID:        "g-003",
				Name:      "error guild",
				CreatedAt: now,
			},
			setupMock: func(m *discordGuildMock.MockDiscordGuildInteractor, ctx context.Context) {
				notFoundErr := &ent.NotFoundError{}
				m.EXPECT().
					GetDiscordGuildByID(ctx, "g-003").
					Return(nil, fmt.Errorf("failed to get discord guild by id: %w", notFoundErr))
				m.EXPECT().
					CreateDiscordGuild(ctx, gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			want:    "",
			wantErr: true,
			errMsg:  "database error",
		},
		{
			name: "database error on get",
			guild: &entity.DiscordGuild{
				ID:        "g-004",
				Name:      "error guild",
				CreatedAt: now,
			},
			setupMock: func(m *discordGuildMock.MockDiscordGuildInteractor, ctx context.Context) {
				m.EXPECT().
					GetDiscordGuildByID(ctx, "g-004").
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
			mockInteractor := discordGuildMock.NewMockDiscordGuildInteractor(ctrl)
			testCase.setupMock(mockInteractor, ctx)

			repo := adapter.NewDiscordGuildRepository(mockInteractor)
			got, err := repo.CreateIfNotExists(ctx, testCase.guild)

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
