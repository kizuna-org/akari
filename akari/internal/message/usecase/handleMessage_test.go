package usecase_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/mock"
	"github.com/kizuna-org/akari/internal/message/usecase"
	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordService "github.com/kizuna-org/akari/pkg/discord/domain/service"
	"go.uber.org/mock/gomock"
)

const (
	defaultCharacterID = 1
	defaultPromptID    = 1
	defaultBotID       = "bot1"
	defaultMessageID   = "msg1"
	defaultChannelID   = "ch1"
	defaultGuildID     = "guild1"
	defaultAuthorID    = "user1"
	defaultContent     = "hello"
	defaultBotPattern  = "^bot"
)

func setupBaseConfig(ctrl *gomock.Controller) usecase.HandleMessageConfig {
	return usecase.HandleMessageConfig{
		LLMRepo:             mock.NewMockLLMRepository(ctrl),
		DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
		DiscordUserRepo:     setupDiscordUserRepo(ctrl, nil),
		DiscordMessageRepo:  setupDiscordMessageRepo(ctrl, nil),
		DiscordChannelRepo:  setupDiscordChannelRepo(ctrl, nil),
		DiscordGuildRepo:    setupDiscordGuildRepo(ctrl, nil),
		ValidationRepo:      mock.NewMockValidationRepository(ctrl),
		CharacterRepo:       mock.NewMockCharacterRepository(ctrl),
		SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
		DefaultCharacterID:  defaultCharacterID,
		DefaultPromptIndex:  0,
		BotNamePatternRegex: regexp.MustCompile(defaultBotPattern),
	}
}

func TestNewHandleMessageInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := setupBaseConfig(ctrl)
	interactor := usecase.NewHandleMessageInteractor(config)

	if interactor == nil {
		t.Error("expected interactor to not be nil")
	}
}

func TestSetBotUserID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := setupBaseConfig(ctrl)
	interactor := usecase.NewHandleMessageInteractor(config)
	interactor.SetBotUserID(defaultBotID)
}

type testCase struct {
	name                 string
	shouldProcessMessage bool
	character            *domain.Character
	characterErr         error
	wantErr              bool
	wantErrMsg           string
	nilDiscordData       bool
}

func buildDiscordData() *discordService.DiscordData {
	data := &discordService.DiscordData{
		User: &databaseDomain.DiscordUser{
			ID:        defaultAuthorID,
			Username:  "TestUser",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Message: &databaseDomain.DiscordMessage{
			ID:        defaultMessageID,
			ChannelID: defaultChannelID,
			AuthorID:  defaultAuthorID,
			Content:   defaultContent,
			CreatedAt: time.Now(),
		},
		Mentions: []string{},
		Channel: &databaseDomain.DiscordChannel{
			ID:        defaultChannelID,
			Type:      databaseDomain.DiscordChannelTypeGuildText,
			Name:      "test-channel",
			GuildID:   defaultGuildID,
			CreatedAt: time.Now(),
		},
		Guild: &databaseDomain.DiscordGuild{
			ID:        defaultGuildID,
			Name:      "test-guild",
			CreatedAt: time.Now(),
		},
	}

	return data
}

func TestHandle(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:           "nil discord params",
			nilDiscordData: true,
			wantErr:        true,
			wantErrMsg:     "discord parameter is nil",
		},
		{
			name:                 "message should not be processed",
			shouldProcessMessage: false,
			wantErr:              false,
		},
		{
			name:                 "failed to get character",
			shouldProcessMessage: true,
			character:            nil,
			characterErr:         errors.New("character not found"),
			wantErr:              true,
			wantErrMsg:           "usecase: get character",
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if testCase.nilDiscordData {
				interactor := usecase.NewHandleMessageInteractor(setupBaseConfig(ctrl))
				interactor.SetBotUserID(defaultBotID)
				err := interactor.Handle(t.Context(), nil)
				checkError(t, err, testCase.wantErr, testCase.wantErrMsg)

				return
			}

			config := setupBaseConfig(ctrl)
			config.CharacterRepo = setupCharacterRepo(ctrl, testCase.character, testCase.characterErr)
			config.ValidationRepo = setupValidationRepo(ctrl, testCase.shouldProcessMessage)

			interactor := usecase.NewHandleMessageInteractor(config)
			interactor.SetBotUserID(defaultBotID)

			data := buildDiscordData()
			err := interactor.Handle(t.Context(), data)
			checkError(t, err, testCase.wantErr, testCase.wantErrMsg)
		})
	}
}

func setupCharacterRepo(
	ctrl *gomock.Controller,
	character *domain.Character,
	err error,
) domain.CharacterRepository {
	repo := mock.NewMockCharacterRepository(ctrl)
	repo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(character, err).
		AnyTimes()

	return repo
}

func setupDiscordUserRepo(ctrl *gomock.Controller, err error) domain.DiscordUserRepository {
	repo := mock.NewMockDiscordUserRepository(ctrl)
	repo.EXPECT().
		CreateIfNotExists(gomock.Any(), gomock.Any()).
		Return("", err).
		AnyTimes()

	return repo
}

func setupDiscordMessageRepo(ctrl *gomock.Controller, err error) domain.DiscordMessageRepository {
	repo := mock.NewMockDiscordMessageRepository(ctrl)
	repo.EXPECT().
		SaveMessage(gomock.Any(), gomock.Any()).
		Return(err).
		AnyTimes()

	return repo
}

func setupDiscordChannelRepo(ctrl *gomock.Controller, err error) domain.DiscordChannelRepository {
	repo := mock.NewMockDiscordChannelRepository(ctrl)
	repo.EXPECT().
		CreateIfNotExists(gomock.Any(), gomock.Any()).
		Return("", err).
		AnyTimes()

	return repo
}

func setupDiscordGuildRepo(ctrl *gomock.Controller, err error) domain.DiscordGuildRepository {
	repo := mock.NewMockDiscordGuildRepository(ctrl)
	repo.EXPECT().
		CreateIfNotExists(gomock.Any(), gomock.Any()).
		Return("", err).
		AnyTimes()

	return repo
}

func setupValidationRepo(ctrl *gomock.Controller, shouldProcess bool) domain.ValidationRepository {
	repo := mock.NewMockValidationRepository(ctrl)
	repo.EXPECT().
		ShouldProcessMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(shouldProcess).
		AnyTimes()

	return repo
}

func checkError(t *testing.T, err error, wantErr bool, wantErrMsg string) {
	t.Helper()

	if (err != nil) != wantErr {
		t.Errorf("expected error: %v, got: %v", wantErr, err)
	}

	if wantErr && wantErrMsg != "" && err != nil {
		if !strings.Contains(err.Error(), wantErrMsg) {
			t.Errorf("expected error to contain '%s', but got '%s'", wantErrMsg, err.Error())
		}
	}
}
