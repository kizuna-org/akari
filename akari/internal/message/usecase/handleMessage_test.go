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
	defaultPromptIndex   int
	wantErr              bool
	wantErrMsg           string
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

func (tc testCase) setup(ctrl *gomock.Controller) (*entity.Message, usecase.HandleMessageConfig) {
	msg := &entity.Message{
		ID:        defaultMessageID,
		ChannelID: defaultChannelID,
		GuildID:   defaultGuildID,
		AuthorID:  defaultAuthorID,
		Content:   defaultContent,
		IsBot:     tc.isBot,
	}

	discordMessageRepo := mock.NewMockDiscordMessageRepository(ctrl)
	discordMessageRepo.EXPECT().SaveMessage(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).Return(nil).Times(1)

	discordChannelRepo := mock.NewMockDiscordChannelRepository(ctrl)
	discordChannelRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		Return("", nil).AnyTimes()

	discordGuildRepo := mock.NewMockDiscordGuildRepository(ctrl)
	discordGuildRepo.EXPECT().CreateIfNotExists(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		Return("", nil).Times(1)

	validationRepo := mock.NewMockValidationRepository(ctrl)
	validationRepo.EXPECT().ShouldProcessMessage(gomock.Not(gomock.Nil()), defaultBotID, gomock.Not(gomock.Nil())).
		Return(tc.shouldProcessMessage).Times(1)

	config := usecase.HandleMessageConfig{
		LLMRepo:             tc.setupLLMRepo(ctrl),
		DiscordRepo:         tc.setupDiscordRepo(ctrl),
		DiscordMessageRepo:  discordMessageRepo,
		DiscordChannelRepo:  discordChannelRepo,
		DiscordGuildRepo:    discordGuildRepo,
		ValidationRepo:      validationRepo,
		CharacterRepo:       tc.setupCharacterRepo(ctrl),
		SystemPromptRepo:    tc.setupSystemPromptRepo(ctrl),
		DefaultCharacterID:  defaultCharacterID,
		DefaultPromptIndex:  tc.defaultPromptIndex,
		BotNamePatternRegex: regexp.MustCompile(defaultBotPattern),
	}

	return msg, config
}

func TestHandle(t *testing.T) {
	t.Parallel()

	tests := []testCase{
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
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msg, config := testCase.setup(ctrl)

			interactor := usecase.NewHandleMessageInteractor(config)
			interactor.SetBotUserID(defaultBotID)

			err := interactor.Handle(
				t.Context(),
				msg,
				&entity.Channel{ID: defaultChannelID, GuildID: defaultGuildID},
				&entity.Guild{ID: defaultGuildID},
			)

			if (err != nil) != testCase.wantErr {
				t.Errorf("expected error: %v, got: %v", testCase.wantErr, err)
			}

			if testCase.wantErr && testCase.wantErrMsg != "" && err != nil {
				if !strings.Contains(err.Error(), testCase.wantErrMsg) {
					t.Errorf(
						"expected error to contain '%s', but got '%s'",
						testCase.wantErrMsg,
						err.Error(),
					)
				}
			}
		})
	}
}
