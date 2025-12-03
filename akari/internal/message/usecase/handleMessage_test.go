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
	conversationRepo domain.ConversationRepository,
	conversationGroupRepo domain.ConversationGroupRepository,
	discordUserRepo domain.DiscordUserRepository,
) usecase.HandleMessageInteractor {
	interactor := usecase.NewHandleMessageInteractor(
		usecase.HandleMessageConfig{
			MessageRepo:           msgRepo,
			ResponseRepo:          respRepo,
			LLMRepo:               llmRepo,
			DiscordRepo:           discordRepo,
			ValidationRepo:        validationRepo,
			CharacterRepo:         characterRepo,
			SystemPromptRepo:      systemPromptRepo,
			ConversationRepo:      conversationRepo,
			ConversationGroupRepo: conversationGroupRepo,
			DiscordUserRepo:       discordUserRepo,
			DefaultCharacterID:    1,
			DefaultPromptIndex:    0,
			BotNamePattern:        "(?i)bot",
		},
	)
	interactor.SetBotUserID("bot-001")

	return interactor
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		GuildID:   "guild-001",
		AuthorID:  "user-001",
		Content:   "Hello bot",
		Timestamp: time.Now(),
		Mentions:  []string{"bot-001"},
		IsBot:     false,
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
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
		llmRepo.EXPECT().GenerateResponse(
			gomock.Any(),
			"You are a helpful Discord bot assistant.",
			"Hello bot",
		).Return("Hi there!", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hi there!").Return(nil),
		respRepo.EXPECT().SaveResponse(gomock.Any(), gomock.Any()).Return(nil),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "",
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(false),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	assert.NoError(t, err)
}

func TestHandleMessageInteractor_Handle_BotNotMentioned(t *testing.T) {
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:       "msg-001",
		Content:  "Hello",
		Mentions: []string{"user-002"},
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(false),
		validationRepo.EXPECT().ContainsBotName(msg, "(?i)bot").Return(false),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	assert.NoError(t, err)
}

func TestHandleMessageInteractor_Handle_BotNameInContent(t *testing.T) {
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		GuildID:   "guild-001",
		AuthorID:  "user-001",
		Content:   "Hey bot, what time is it?",
		Timestamp: time.Now(),
		Mentions:  []string{},
		IsBot:     false,
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
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(false),
		validationRepo.EXPECT().ContainsBotName(msg, "(?i)bot").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
		llmRepo.EXPECT().GenerateResponse(
			gomock.Any(),
			"You are a helpful Discord bot assistant.",
			"Hey bot, what time is it?",
		).Return("It's 3 PM", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "It's 3 PM").Return(nil),
		respRepo.EXPECT().SaveResponse(gomock.Any(), gomock.Any()).Return(nil),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:      "msg-001",
		Content: "Hello",
	}

	msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(errors.New("db error"))

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:       "msg-001",
		Content:  "Hello",
		AuthorID: "user-001",
		IsBot:    false,
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(
			nil,
			errors.New("character not found"),
		),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:       "msg-001",
		Content:  "Hello",
		AuthorID: "user-001",
		IsBot:    false,
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
		characterRepo.EXPECT().GetCharacterByID(gomock.Any(), 1).Return(&domain.Character{
			ID:              1,
			Name:            "TestBot",
			SystemPromptIDs: []int{1},
		}, nil),
		systemPromptRepo.EXPECT().GetSystemPromptByID(gomock.Any(), 1).Return(
			nil,
			errors.New("prompt not found"),
		),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:       "msg-001",
		Content:  "Hello",
		AuthorID: "user-001",
		IsBot:    false,
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
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
			gomock.Any(),
			"You are a helpful Discord bot assistant.",
			"Hello",
		).Return("", errors.New("llm error")),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		Content:   "Hello",
		AuthorID:  "user-001",
		IsBot:     false,
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
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
			gomock.Any(),
			"You are a helpful Discord bot assistant.",
			"Hello",
		).Return("Hi", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hi").Return(
			errors.New("discord error"),
		),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
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
	conversationRepo := mock.NewMockConversationRepository(ctrl)
	conversationGroupRepo := mock.NewMockConversationGroupRepository(ctrl)
	discordUserRepo := mock.NewMockDiscordUserRepository(ctrl)

	msg := &domain.Message{
		ID:        "msg-001",
		ChannelID: "ch-001",
		Content:   "Hello",
		AuthorID:  "user-001",
		IsBot:     false,
	}

	gomock.InOrder(
		msgRepo.EXPECT().SaveMessage(gomock.Any(), msg).Return(nil),
		validationRepo.EXPECT().ShouldProcessMessage(msg).Return(true),
		validationRepo.EXPECT().IsBotMentioned(msg, "bot-001").Return(true),
		discordUserRepo.EXPECT().GetOrCreateDiscordUser(
			gomock.Any(),
			msg.AuthorID,
			"",
			msg.IsBot,
		).Return(1, nil),
		conversationGroupRepo.EXPECT().GetConversationGroupByCharacterID(gomock.Any(), 1).Return(&domain.ConversationGroup{
			ID:          1,
			CharacterID: 1,
		}, nil),
		conversationRepo.EXPECT().CreateConversation(
			gomock.Any(),
			msg.ID,
			1,
			gomock.Any(),
		).Return(nil),
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
			gomock.Any(),
			"You are a helpful Discord bot assistant.",
			"Hello",
		).Return("Hi", nil),
		discordRepo.EXPECT().SendMessage(gomock.Any(), "ch-001", "Hi").Return(nil),
		respRepo.EXPECT().SaveResponse(gomock.Any(), gomock.Any()).Return(
			errors.New("db error"),
		),
	)

	interactor := newTestInteractor(
		msgRepo,
		respRepo,
		llmRepo,
		discordRepo,
		validationRepo,
		characterRepo,
		systemPromptRepo,
		conversationRepo,
		conversationGroupRepo,
		discordUserRepo,
	)
	err := interactor.Handle(t.Context(), msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save response")
}
