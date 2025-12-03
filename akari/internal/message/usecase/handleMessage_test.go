package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/message/domain"
	"github.com/kizuna-org/akari/internal/message/domain/mock"
	"github.com/kizuna-org/akari/internal/message/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestInteractor(
	msgRepo domain.MessageRepository,
	respRepo domain.ResponseRepository,
	llmRepo domain.LLMRepository,
	discordRepo domain.DiscordRepository,
	validationRepo domain.ValidationRepository,
	characterRepo domain.CharacterRepository,
	systemPromptRepo domain.SystemPromptRepository,
) usecase.HandleMessageInteractor {
	return usecase.NewHandleMessageInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo, 1, 0,
	)
}

func TestNewHandleMessageInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	assert.NotNil(t, interactor)
}

func TestHandleMessageInteractor_Handle_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		GuildID:   "guild-001",
		AuthorID:  "user-001",
		Content:   "Hello bot",
		Timestamp: time.Now(),
	}

	characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
		ID:              1,
		Name:            "TestBot",
		SystemPromptIDs: []int{1},
	}, nil)

	systemPromptRepo.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(&domain.SystemPrompt{
		ID:     1,
		Prompt: "You are a helpful Discord bot assistant.",
	}, nil)

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		llmRepo.EXPECT().GenerateResponse(
			gomock.Any(), "You are a helpful Discord bot assistant.", "Hello bot",
		).Return("Hi there!", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hi there!").Return(nil),
		respRepo.EXPECT().SaveResponse(gomock.Any(), gomock.Any()).Return(nil),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	assert.NoError(t, err)
}

func TestHandleMessageInteractor_Handle_ValidationFails(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(false),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	assert.NoError(t, err)
}

func TestHandleMessageInteractor_Handle_SaveMessageError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "Hello",
	}

	msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(errors.New("db error"))

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save message")
}

func TestHandleMessageInteractor_Handle_GetCharacterError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "Hello",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(nil, errors.New("character not found")),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get character")
}

func TestHandleMessageInteractor_Handle_GetSystemPromptError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "Hello",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
			ID:              1,
			Name:            "TestBot",
			SystemPromptIDs: []int{1},
		}, nil),
		systemPromptRepo.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(
			nil, errors.New("prompt not found"),
		),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get system prompt")
}

func TestHandleMessageInteractor_Handle_LLMError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "Hello",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
			ID:              1,
			Name:            "TestBot",
			SystemPromptIDs: []int{1},
		}, nil),
		systemPromptRepo.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(&domain.SystemPrompt{
			ID:     1,
			Prompt: "You are a helpful Discord bot assistant.",
		}, nil),
		llmRepo.EXPECT().GenerateResponse(
			gomock.Any(), "You are a helpful Discord bot assistant.", "Hello",
		).Return("", errors.New("llm error")),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to generate response")
}

func TestHandleMessageInteractor_Handle_SendMessageError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		Content:   "Hello",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
			ID:              1,
			Name:            "TestBot",
			SystemPromptIDs: []int{1},
		}, nil),
		systemPromptRepo.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(&domain.SystemPrompt{
			ID:     1,
			Prompt: "You are a helpful Discord bot assistant.",
		}, nil),
		llmRepo.EXPECT().GenerateResponse(
			gomock.Any(), "You are a helpful Discord bot assistant.", "Hello",
		).Return("Hi", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hi").Return(
			errors.New("discord error"),
		),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send message")
}

func TestHandleMessageInteractor_Handle_SaveResponseError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	msgRepo := mock.NewMockMessageRepository(ctrl)
	respRepo := mock.NewMockResponseRepository(ctrl)
	llmRepo := mock.NewMockLLMRepository(ctrl)
	discordRepo := mock.NewMockDiscordRepository(ctrl)
	validationRepo := mock.NewMockValidationRepository(ctrl)
	characterRepo := mock.NewMockCharacterRepository(ctrl)
	systemPromptRepo := mock.NewMockSystemPromptRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		Content:   "Hello",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
			ID:              1,
			Name:            "TestBot",
			SystemPromptIDs: []int{1},
		}, nil),
		systemPromptRepo.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(&domain.SystemPrompt{
			ID:     1,
			Prompt: "You are a helpful Discord bot assistant.",
		}, nil),
		llmRepo.EXPECT().GenerateResponse(
			gomock.Any(), "You are a helpful Discord bot assistant.", "Hello",
		).Return("Hi", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hi").Return(nil),
		respRepo.EXPECT().SaveResponse(gomock.Any(), gomock.Any()).Return(
			errors.New("db error"),
		),
	)

	interactor := newTestInteractor(
		msgRepo, respRepo, llmRepo, discordRepo, validationRepo, characterRepo, systemPromptRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save response")
}
