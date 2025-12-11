package usecase_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/mock"
	"github.com/kizuna-org/akari/internal/message/usecase"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"github.com/kizuna-org/akari/pkg/discord/domain/service"
	"go.uber.org/mock/gomock"
)

const (
	defaultCharacterID   = 1
	defaultPromptID      = 1
	defaultBotID         = "bot1"
	defaultMessageID     = "msg1"
	defaultChannelID     = "ch1"
	defaultGuildID       = "guild1"
	defaultAuthorID      = "user1"
	defaultContent       = "hello"
	defaultSystemPrompt  = "system prompt"
	defaultLLMResponse   = "response"
	defaultCharacterName = "TestCharacter"
	defaultBotPattern    = "^bot"
)

func newCharacter(promptIDs []int) *domain.Character {
	return &domain.Character{ID: defaultCharacterID, Name: defaultCharacterName, SystemPromptIDs: promptIDs}
}

func newSystemPrompt(prompt string) *domain.SystemPrompt {
	return &domain.SystemPrompt{ID: defaultPromptID, Prompt: prompt}
}

func setupBaseConfig(ctrl *gomock.Controller) usecase.HandleMessageConfig {
	return usecase.HandleMessageConfig{
		LLMRepo:             mock.NewMockLLMRepository(ctrl),
		DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
		DiscordMessageRepo:  mock.NewMockDiscordMessageRepository(ctrl),
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
	isBot                bool
	shouldProcessMessage bool
	character            *domain.Character
	characterErr         error
	systemPrompt         *domain.SystemPrompt
	systemPromptErr      error
	llmResponse          string
	llmErr               error
	discordErr           error
	userCreateErr        error
	guildCreateErr       error
	channelCreateErr     error
	messageCreateErr     error
	defaultPromptIndex   int
	wantErr              bool
	wantErrMsg           string
	nilDiscordData       bool
	includeUser          bool
	includeGuild         bool
	includeChannel       bool
	includeMessage       bool
}

func (tc testCase) setupCharacterRepo(ctrl *gomock.Controller) *mock.MockCharacterRepository {
	if tc.shouldProcessMessage {
		repo := mock.NewMockCharacterRepository(ctrl)
		repo.EXPECT().Get(gomock.Not(gomock.Nil()), defaultCharacterID).Return(tc.character, tc.characterErr).Times(1)

		return repo
	}

	return mock.NewMockCharacterRepository(ctrl)
}

func (tc testCase) setupSystemPromptRepo(ctrl *gomock.Controller) *mock.MockSystemPromptRepository {
	shouldFetch := tc.character != nil && len(tc.character.SystemPromptIDs) > 0 &&
		tc.characterErr == nil && tc.defaultPromptIndex < len(tc.character.SystemPromptIDs)
	if shouldFetch {
		repo := mock.NewMockSystemPromptRepository(ctrl)
		repo.EXPECT().Get(gomock.Not(gomock.Nil()), defaultPromptID).Return(tc.systemPrompt, tc.systemPromptErr).Times(1)

		return repo
	}

	return mock.NewMockSystemPromptRepository(ctrl)
}

func (tc testCase) setupLLMRepo(ctrl *gomock.Controller) *mock.MockLLMRepository {
	shouldCall := tc.character != nil && tc.characterErr == nil && tc.systemPromptErr == nil
	if !shouldCall {
		return mock.NewMockLLMRepository(ctrl)
	}

	var systemPromptText string
	if len(tc.character.SystemPromptIDs) > 0 && tc.systemPrompt != nil &&
		tc.defaultPromptIndex < len(tc.character.SystemPromptIDs) {
		systemPromptText = tc.systemPrompt.Prompt
	}

	repo := mock.NewMockLLMRepository(ctrl)
	repo.EXPECT().GenerateResponse(gomock.Not(gomock.Nil()), systemPromptText, defaultContent).
		Return(tc.llmResponse, tc.llmErr).Times(1)

	return repo
}

func (tc testCase) setupDiscordRepo(ctrl *gomock.Controller) *mock.MockDiscordRepository {
	shouldSend := tc.character != nil && tc.characterErr == nil &&
		tc.systemPromptErr == nil && tc.llmErr == nil
	if shouldSend {
		repo := mock.NewMockDiscordRepository(ctrl)
		repo.EXPECT().SendMessage(gomock.Not(gomock.Nil()), defaultChannelID, tc.llmResponse).Return(tc.discordErr).Times(1)

		return repo
	}

	return mock.NewMockDiscordRepository(ctrl)
}

func (tc testCase) expectUserRepo(ctrl *gomock.Controller) *mock.MockDiscordUserRepository {
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)
	if tc.includeUser && tc.userCreateErr != nil {
		discordUserRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return("", tc.userCreateErr).Times(1)
	} else if tc.includeUser {
		discordUserRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return("", nil).Times(1)
	}

	return discordUserRepo
}

func (tc testCase) expectGuildRepo(ctrl *gomock.Controller) *mock.MockDiscordGuildRepository {
	discordGuildRepo := mock.NewMockDiscordGuildRepository(ctrl)
	if tc.includeGuild && tc.guildCreateErr != nil {
		discordGuildRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return("", tc.guildCreateErr).Times(1)
	} else if tc.includeGuild && tc.userCreateErr == nil {
		discordGuildRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return("", nil).Times(1)
	}

	return discordGuildRepo
}

func (tc testCase) expectChannelRepo(ctrl *gomock.Controller) *mock.MockDiscordChannelRepository {
	discordChannelRepo := mock.NewMockDiscordChannelRepository(ctrl)
	if tc.includeChannel && tc.channelCreateErr != nil {
		discordChannelRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return("", tc.channelCreateErr).Times(1)
	} else if tc.includeChannel && tc.userCreateErr == nil && tc.guildCreateErr == nil {
		discordChannelRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return("", nil).Times(1)
	}

	return discordChannelRepo
}

func (tc testCase) expectMessageRepo(ctrl *gomock.Controller) *mock.MockDiscordMessageRepository {
	discordMessageRepo := mock.NewMockDiscordMessageRepository(ctrl)
	if tc.includeMessage && tc.messageCreateErr != nil {
		discordMessageRepo.EXPECT().SaveMessage(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return(tc.messageCreateErr).Times(1)
	} else if tc.includeMessage && tc.userCreateErr == nil && tc.guildCreateErr == nil && tc.channelCreateErr == nil {
		discordMessageRepo.EXPECT().SaveMessage(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
			Return(nil).Times(1)
	}

	return discordMessageRepo
}

func (tc testCase) expectValidationRepo(ctrl *gomock.Controller) *mock.MockValidationRepository {
	validationRepo := mock.NewMockValidationRepository(ctrl)
	if tc.userCreateErr == nil && tc.guildCreateErr == nil && tc.channelCreateErr == nil && tc.messageCreateErr == nil {
		validationRepo.EXPECT().ShouldProcessMessage(gomock.Not(gomock.Nil()), defaultBotID, gomock.Not(gomock.Nil())).
			Return(tc.shouldProcessMessage).Times(1)
	}

	return validationRepo
}

func (tc testCase) setup(ctrl *gomock.Controller) (*entity.Message, usecase.HandleMessageConfig) {
	msg := &entity.Message{
		ID:        defaultMessageID,
		ChannelID: defaultChannelID,
		GuildID:   defaultGuildID,
		AuthorID:  defaultAuthorID,
		Content:   defaultContent,
		IsBot:     tc.isBot,
	}

	config := usecase.HandleMessageConfig{
		LLMRepo:             tc.setupLLMRepo(ctrl),
		DiscordRepo:         tc.setupDiscordRepo(ctrl),
		DiscordUserRepo:     tc.expectUserRepo(ctrl),
		DiscordMessageRepo:  tc.expectMessageRepo(ctrl),
		DiscordChannelRepo:  tc.expectChannelRepo(ctrl),
		DiscordGuildRepo:    tc.expectGuildRepo(ctrl),
		ValidationRepo:      tc.expectValidationRepo(ctrl),
		CharacterRepo:       tc.setupCharacterRepo(ctrl),
		SystemPromptRepo:    tc.setupSystemPromptRepo(ctrl),
		DefaultCharacterID:  defaultCharacterID,
		DefaultPromptIndex:  tc.defaultPromptIndex,
		BotNamePatternRegex: regexp.MustCompile(defaultBotPattern),
	}

	return msg, config
}

func buildDiscordData(testCase testCase) *service.DiscordData {
	data := &service.DiscordData{
		Message: &entity.Message{
			ID:        defaultMessageID,
			ChannelID: defaultChannelID,
			GuildID:   defaultGuildID,
			AuthorID:  defaultAuthorID,
			Content:   defaultContent,
			IsBot:     testCase.isBot,
		},
	}

	if testCase.includeUser {
		data.User = &entity.User{ID: defaultAuthorID}
	}

	if testCase.includeGuild {
		data.Guild = &entity.Guild{ID: defaultGuildID}
	}

	if testCase.includeChannel {
		data.Channel = &entity.Channel{ID: defaultChannelID, GuildID: defaultGuildID}
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
			name:                 "successful message handling with response",
			isBot:                false,
			shouldProcessMessage: true,
			character:            newCharacter([]int{1}),
			characterErr:         nil,
			systemPrompt:         newSystemPrompt(defaultSystemPrompt),
			systemPromptErr:      nil,
			llmResponse:          defaultLLMResponse,
			llmErr:               nil,
			discordErr:           nil,
			defaultPromptIndex:   0,
			wantErr:              false,
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "message should not be processed",
			isBot:                true,
			shouldProcessMessage: false,
			character:            nil,
			characterErr:         nil,
			systemPrompt:         nil,
			systemPromptErr:      nil,
			llmResponse:          "",
			llmErr:               nil,
			discordErr:           nil,
			defaultPromptIndex:   0,
			wantErr:              false,
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "failed to create user",
			isBot:                false,
			shouldProcessMessage: false,
			userCreateErr:        errors.New("user creation failed"),
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "get or create user",
			includeUser:          true,
		},
		{
			name:                 "failed to create guild",
			isBot:                false,
			shouldProcessMessage: false,
			guildCreateErr:       errors.New("guild creation failed"),
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "create discord guild if not exists",
			includeGuild:         true,
		},
		{
			name:                 "failed to create channel",
			isBot:                false,
			shouldProcessMessage: false,
			channelCreateErr:     errors.New("channel creation failed"),
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "create discord channel if not exists",
			includeChannel:       true,
		},
		{
			name:                 "failed to save message",
			isBot:                false,
			shouldProcessMessage: false,
			messageCreateErr:     errors.New("save message failed"),
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "save message",
			includeMessage:       true,
		},
		{
			name:                 "failed to get character",
			isBot:                false,
			shouldProcessMessage: true,
			character:            nil,
			characterErr:         errors.New("character not found"),
			systemPrompt:         nil,
			systemPromptErr:      nil,
			llmResponse:          "",
			llmErr:               nil,
			discordErr:           nil,
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "usecase: get character",
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "empty system prompt IDs",
			isBot:                false,
			shouldProcessMessage: true,
			character:            newCharacter([]int{}),
			characterErr:         nil,
			systemPrompt:         nil,
			systemPromptErr:      nil,
			llmResponse:          defaultLLMResponse,
			llmErr:               nil,
			discordErr:           nil,
			defaultPromptIndex:   0,
			wantErr:              false,
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "prompt index out of range",
			isBot:                false,
			shouldProcessMessage: true,
			character:            newCharacter([]int{1}),
			characterErr:         nil,
			systemPrompt:         nil,
			systemPromptErr:      nil,
			llmResponse:          defaultLLMResponse,
			llmErr:               nil,
			discordErr:           nil,
			defaultPromptIndex:   5,
			wantErr:              false,
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "failed to generate response",
			isBot:                false,
			shouldProcessMessage: true,
			character:            newCharacter([]int{1}),
			characterErr:         nil,
			systemPrompt:         newSystemPrompt(defaultSystemPrompt),
			systemPromptErr:      nil,
			llmResponse:          "",
			llmErr:               errors.New("llm error"),
			discordErr:           nil,
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "usecase: generate response",
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "failed to send discord message",
			isBot:                false,
			shouldProcessMessage: true,
			character:            newCharacter([]int{1}),
			characterErr:         nil,
			systemPrompt:         newSystemPrompt(defaultSystemPrompt),
			systemPromptErr:      nil,
			llmResponse:          defaultLLMResponse,
			llmErr:               nil,
			discordErr:           errors.New("discord error"),
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "usecase: send message",
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
		},
		{
			name:                 "failed to get system prompt",
			isBot:                false,
			shouldProcessMessage: true,
			character:            newCharacter([]int{1}),
			characterErr:         nil,
			systemPrompt:         nil,
			systemPromptErr:      errors.New("prompt not found"),
			llmResponse:          "",
			llmErr:               nil,
			discordErr:           nil,
			defaultPromptIndex:   0,
			wantErr:              true,
			wantErrMsg:           "usecase: get system prompt",
			includeUser:          true,
			includeGuild:         true,
			includeChannel:       true,
			includeMessage:       true,
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

			_, config := testCase.setup(ctrl)
			interactor := usecase.NewHandleMessageInteractor(config)
			interactor.SetBotUserID(defaultBotID)

			data := buildDiscordData(testCase)
			err := interactor.Handle(t.Context(), data)
			checkError(t, err, testCase.wantErr, testCase.wantErrMsg)
		})
	}
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
