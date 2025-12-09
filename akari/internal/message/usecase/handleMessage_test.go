package usecase_test

import (
	"errors"
	"regexp"
	"testing"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/mock"
	"github.com/kizuna-org/akari/internal/message/usecase"
	"github.com/kizuna-org/akari/pkg/discord/domain/entity"
	"go.uber.org/mock/gomock"
)

func TestNewHandleMessageInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := usecase.HandleMessageConfig{
		LLMRepo:             mock.NewMockLLMRepository(ctrl),
		DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
		ValidationRepo:      mock.NewMockValidationRepository(ctrl),
		CharacterRepo:       mock.NewMockCharacterRepository(ctrl),
		SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
		DefaultCharacterID:  1,
		DefaultPromptIndex:  0,
		BotNamePatternRegex: regexp.MustCompile(`^bot$`),
	}

	interactor := usecase.NewHandleMessageInteractor(config)

	if interactor == nil {
		t.Error("expected interactor to not be nil")
	}
}

func TestSetBotUserID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := usecase.HandleMessageConfig{
		LLMRepo:             mock.NewMockLLMRepository(ctrl),
		DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
		ValidationRepo:      mock.NewMockValidationRepository(ctrl),
		CharacterRepo:       mock.NewMockCharacterRepository(ctrl),
		SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
		DefaultCharacterID:  1,
		DefaultPromptIndex:  0,
		BotNamePatternRegex: regexp.MustCompile(`^bot$`),
	}

	interactor := usecase.NewHandleMessageInteractor(config)
	interactor.SetBotUserID("bot123")
}

type testSetup struct {
	msg    *entity.Message
	botID  string
	config usecase.HandleMessageConfig
	err    bool
	errMsg string
}

func setupSuccessfulMessage(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg123",
		ChannelID: "ch123",
		GuildID:   "guild123",
		AuthorID:  "user123",
		Content:   "hello bot",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot123", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.Character{
			ID:              1,
			Name:            "TestCharacter",
			SystemPromptIDs: []int{1, 2},
		},
		nil,
	).Times(1)

	promptRepo := mock.NewMockSystemPromptRepository(ctrl)
	promptRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.SystemPrompt{
			ID:     1,
			Prompt: "You are a helpful bot",
		},
		nil,
	).Times(1)

	llmRepo := mock.NewMockLLMRepository(ctrl)
	llmRepo.EXPECT().GenerateResponse(
		gomock.Any(),
		"You are a helpful bot",
		"hello bot",
	).Return("Hello! How can I help?", nil).Times(1)

	discordRepo := mock.NewMockDiscordRepository(ctrl)
	discordRepo.EXPECT().SendMessage(
		gomock.Any(),
		"ch123",
		"Hello! How can I help?",
	).Return(nil).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot123",
		config: usecase.HandleMessageConfig{
			LLMRepo:             llmRepo,
			DiscordRepo:         discordRepo,
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    promptRepo,
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err: false,
	}
}

func setupMessageNotProcessed(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg456",
		ChannelID: "ch456",
		GuildID:   "guild456",
		AuthorID:  "bot456",
		Content:   "I am a bot",
		IsBot:     true,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot456", gomock.Any()).Return(false).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot456",
		config: usecase.HandleMessageConfig{
			LLMRepo:             mock.NewMockLLMRepository(ctrl),
			DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
			ValidationRepo:      valRepo,
			CharacterRepo:       mock.NewMockCharacterRepository(ctrl),
			SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err: false,
	}
}

func setupFailedCharacterGet(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg999",
		ChannelID: "ch999",
		GuildID:   "guild999",
		AuthorID:  "user999",
		Content:   "hello",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot999", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		nil,
		errors.New("character not found"),
	).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot999",
		config: usecase.HandleMessageConfig{
			LLMRepo:             mock.NewMockLLMRepository(ctrl),
			DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err:    true,
		errMsg: "usecase: get character",
	}
}

func setupEmptyPromptIDs(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg111",
		ChannelID: "ch111",
		GuildID:   "guild111",
		AuthorID:  "user111",
		Content:   "hello",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot111", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.Character{
			ID:              1,
			Name:            "TestCharacter",
			SystemPromptIDs: []int{},
		},
		nil,
	).Times(1)

	llmRepo := mock.NewMockLLMRepository(ctrl)
	llmRepo.EXPECT().GenerateResponse(
		gomock.Any(),
		"",
		"hello",
	).Return("Hello!", nil).Times(1)

	discordRepo := mock.NewMockDiscordRepository(ctrl)
	discordRepo.EXPECT().SendMessage(
		gomock.Any(),
		"ch111",
		"Hello!",
	).Return(nil).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot111",
		config: usecase.HandleMessageConfig{
			LLMRepo:             llmRepo,
			DiscordRepo:         discordRepo,
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err: false,
	}
}

func setupPromptIndexOutOfRange(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg222",
		ChannelID: "ch222",
		GuildID:   "guild222",
		AuthorID:  "user222",
		Content:   "hello",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot222", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.Character{
			ID:              1,
			Name:            "TestCharacter",
			SystemPromptIDs: []int{1},
		},
		nil,
	).Times(1)

	llmRepo := mock.NewMockLLMRepository(ctrl)
	llmRepo.EXPECT().GenerateResponse(
		gomock.Any(),
		"",
		"hello",
	).Return("Hello!", nil).Times(1)

	discordRepo := mock.NewMockDiscordRepository(ctrl)
	discordRepo.EXPECT().SendMessage(
		gomock.Any(),
		"ch222",
		"Hello!",
	).Return(nil).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot222",
		config: usecase.HandleMessageConfig{
			LLMRepo:             llmRepo,
			DiscordRepo:         discordRepo,
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    mock.NewMockSystemPromptRepository(ctrl),
			DefaultCharacterID:  1,
			DefaultPromptIndex:  5,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err: false,
	}
}

func setupFailedGenerateResponse(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg333",
		ChannelID: "ch333",
		GuildID:   "guild333",
		AuthorID:  "user333",
		Content:   "hello",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot333", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.Character{
			ID:              1,
			Name:            "TestCharacter",
			SystemPromptIDs: []int{1},
		},
		nil,
	).Times(1)

	promptRepo := mock.NewMockSystemPromptRepository(ctrl)
	promptRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.SystemPrompt{
			ID:     1,
			Prompt: "You are a helpful bot",
		},
		nil,
	).Times(1)

	llmRepo := mock.NewMockLLMRepository(ctrl)
	llmRepo.EXPECT().GenerateResponse(
		gomock.Any(),
		"You are a helpful bot",
		"hello",
	).Return("", errors.New("llm error")).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot333",
		config: usecase.HandleMessageConfig{
			LLMRepo:             llmRepo,
			DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    promptRepo,
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err:    true,
		errMsg: "usecase: generate response",
	}
}

func setupFailedSendMessage(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg444",
		ChannelID: "ch444",
		GuildID:   "guild444",
		AuthorID:  "user444",
		Content:   "hello",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot444", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.Character{
			ID:              1,
			Name:            "TestCharacter",
			SystemPromptIDs: []int{1},
		},
		nil,
	).Times(1)

	promptRepo := mock.NewMockSystemPromptRepository(ctrl)
	promptRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.SystemPrompt{
			ID:     1,
			Prompt: "You are a helpful bot",
		},
		nil,
	).Times(1)

	llmRepo := mock.NewMockLLMRepository(ctrl)
	llmRepo.EXPECT().GenerateResponse(
		gomock.Any(),
		"You are a helpful bot",
		"hello",
	).Return("Hello!", nil).Times(1)

	discordRepo := mock.NewMockDiscordRepository(ctrl)
	discordRepo.EXPECT().SendMessage(
		gomock.Any(),
		"ch444",
		"Hello!",
	).Return(errors.New("discord error")).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot444",
		config: usecase.HandleMessageConfig{
			LLMRepo:             llmRepo,
			DiscordRepo:         discordRepo,
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    promptRepo,
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err:    true,
		errMsg: "usecase: send message",
	}
}

func setupFailedGetSystemPrompt(ctrl *gomock.Controller) testSetup {
	msg := &entity.Message{
		ID:        "msg666",
		ChannelID: "ch666",
		GuildID:   "guild666",
		AuthorID:  "user666",
		Content:   "hello",
		IsBot:     false,
	}

	valRepo := mock.NewMockValidationRepository(ctrl)
	valRepo.EXPECT().ShouldProcessMessage(msg, "bot666", gomock.Any()).Return(true).Times(1)

	charRepo := mock.NewMockCharacterRepository(ctrl)
	charRepo.EXPECT().Get(gomock.Any(), 1).Return(
		&domain.Character{
			ID:              1,
			Name:            "TestCharacter",
			SystemPromptIDs: []int{1},
		},
		nil,
	).Times(1)

	promptRepo := mock.NewMockSystemPromptRepository(ctrl)
	promptRepo.EXPECT().Get(gomock.Any(), 1).Return(
		nil,
		errors.New("prompt not found"),
	).Times(1)

	return testSetup{
		msg:   msg,
		botID: "bot666",
		config: usecase.HandleMessageConfig{
			LLMRepo:             mock.NewMockLLMRepository(ctrl),
			DiscordRepo:         mock.NewMockDiscordRepository(ctrl),
			ValidationRepo:      valRepo,
			CharacterRepo:       charRepo,
			SystemPromptRepo:    promptRepo,
			DefaultCharacterID:  1,
			DefaultPromptIndex:  0,
			BotNamePatternRegex: regexp.MustCompile(`^bot`),
		},
		err:    true,
		errMsg: "usecase: get system prompt",
	}
}

func TestHandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*gomock.Controller) testSetup
	}{
		{name: "successful message handling with response", setup: setupSuccessfulMessage},
		{name: "message should not be processed", setup: setupMessageNotProcessed},
		{name: "failed to get character", setup: setupFailedCharacterGet},
		{name: "empty system prompt IDs", setup: setupEmptyPromptIDs},
		{name: "prompt index out of range", setup: setupPromptIndexOutOfRange},
		{name: "failed to generate response", setup: setupFailedGenerateResponse},
		{name: "failed to send discord message", setup: setupFailedSendMessage},
		{name: "failed to get system prompt", setup: setupFailedGetSystemPrompt},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testSetup := testCase.setup(ctrl)

			interactor := usecase.NewHandleMessageInteractor(testSetup.config)
			interactor.SetBotUserID(testSetup.botID)

			err := interactor.Handle(t.Context(), testSetup.msg)

			if (err != nil) != testSetup.err {
				t.Errorf("expected error: %v, got: %v", testSetup.err, err)
			}

			if testSetup.err && testSetup.errMsg != "" && err != nil {
				if !stringContains(err.Error(), testSetup.errMsg) {
					t.Errorf("expected error to contain '%s', but got '%s'", testSetup.errMsg, err.Error())
				}
			}
		})
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true

		for j := range len(substr) {
			if s[i+j] != substr[j] {
				match = false

				break
			}
		}

		if match {
			return true
		}
	}

	return false
}
