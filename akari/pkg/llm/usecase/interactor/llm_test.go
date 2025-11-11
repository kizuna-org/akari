package interactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizuna-org/akari/pkg/llm/domain"
	"github.com/kizuna-org/akari/pkg/llm/domain/mock"
	"github.com/kizuna-org/akari/pkg/llm/usecase/interactor"
	"go.uber.org/mock/gomock"
	"google.golang.org/genai"
)

const testSystemPrompt = "You are a helpful assistant"

func ptr(s string) *string {
	return &s
}

func TestNewLLMInteractor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	interactor := interactor.NewLLMInteractor(mockRepo)

	if interactor == nil {
		t.Fatal("Expected non-nil interactor")
	}
}

func TestLLMInteractorImpl_SendChatMessage_SuccessWithHistory(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	llmInteractor := interactor.NewLLMInteractor(mockRepo)

	ctx := t.Context()
	history := []*domain.Content{
		{Role: "user", Parts: []*genai.Part{{Text: "Hello"}}},
		{Role: "model", Parts: []*genai.Part{{Text: "Hi there!"}}},
	}
	message := "How are you?"
	functions := []domain.Function{}

	expectedMessages := []*string{ptr("I'm doing well, thank you!")}
	expectedParts := []*domain.Part{{Text: "I'm doing well, thank you!"}}

	mockRepo.EXPECT().
		SendChatMessage(ctx, testSystemPrompt, history, message, functions).
		Return(expectedMessages, expectedParts, nil).
		Times(1)

	messages, parts, err := llmInteractor.SendChatMessage(ctx, testSystemPrompt, history, message, functions)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(messages) != len(expectedMessages) {
		t.Fatalf("Expected %d messages, got %d", len(expectedMessages), len(messages))
	}

	if len(parts) != len(expectedParts) {
		t.Fatalf("Expected %d parts, got %d", len(expectedParts), len(parts))
	}

	if messages[0] == nil || *messages[0] != *expectedMessages[0] {
		t.Errorf("Expected message %v, got %v", *expectedMessages[0], *messages[0])
	}
}

func TestLLMInteractorImpl_SendChatMessage_SuccessWithFunctions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	llmInteractor := interactor.NewLLMInteractor(mockRepo)

	ctx := t.Context()
	history := []*domain.Content{}
	message := "What's the weather?"
	functions := []domain.Function{
		{
			FunctionDeclaration: &genai.FunctionDeclaration{
				Name:                 "get_weather",
				Description:          "Get current weather",
				Parameters:           nil,
				ParametersJsonSchema: nil,
				Response:             nil,
				ResponseJsonSchema:   nil,
				Behavior:             "",
			},
			Function: func(ctx context.Context, request *genai.FunctionCall) (map[string]any, error) {
				return map[string]any{"temperature": 72}, nil
			},
		},
	}

	expectedMessages := []*string{ptr("The weather is 72 degrees.")}
	expectedParts := []*domain.Part{{Text: "The weather is 72 degrees."}}

	mockRepo.EXPECT().
		SendChatMessage(ctx, testSystemPrompt, history, message, functions).
		Return(expectedMessages, expectedParts, nil).
		Times(1)

	messages, parts, err := llmInteractor.SendChatMessage(ctx, testSystemPrompt, history, message, functions)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if len(parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(parts))
	}
}

func TestLLMInteractorImpl_SendChatMessage_Error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	llmInteractor := interactor.NewLLMInteractor(mockRepo)

	ctx := t.Context()
	history := []*domain.Content{}
	message := "Hello"
	functions := []domain.Function{}
	expectedError := errors.New("failed to send message")

	mockRepo.EXPECT().
		SendChatMessage(ctx, testSystemPrompt, history, message, functions).
		Return(nil, nil, expectedError).
		Times(1)

	messages, parts, err := llmInteractor.SendChatMessage(ctx, testSystemPrompt, history, message, functions)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != expectedError.Error() {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}

	if messages != nil || parts != nil {
		t.Error("Expected nil messages and parts on error")
	}
}

func TestLLMInteractorImpl_SendChatMessage_EmptyHistory(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	llmInteractor := interactor.NewLLMInteractor(mockRepo)

	ctx := t.Context()
	history := []*domain.Content{}
	message := "First message"
	functions := []domain.Function{}

	expectedMessages := []*string{ptr("Hello! How can I help you?")}
	expectedParts := []*domain.Part{{Text: "Hello! How can I help you?"}}

	mockRepo.EXPECT().
		SendChatMessage(ctx, testSystemPrompt, history, message, functions).
		Return(expectedMessages, expectedParts, nil).
		Times(1)

	messages, parts, err := llmInteractor.SendChatMessage(ctx, testSystemPrompt, history, message, functions)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if len(parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(parts))
	}
}

func TestLLMInteractorImpl_SendChatMessage_MultipleResponses(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	llmInteractor := interactor.NewLLMInteractor(mockRepo)

	ctx := t.Context()
	history := []*domain.Content{}
	message := "Tell me a story"
	functions := []domain.Function{}

	expectedMessages := []*string{
		ptr("Once upon a time..."),
		ptr("There was a brave knight..."),
		ptr("And they lived happily ever after."),
	}
	expectedParts := []*domain.Part{
		{Text: "Once upon a time..."},
		{Text: "There was a brave knight..."},
		{Text: "And they lived happily ever after."},
	}

	mockRepo.EXPECT().
		SendChatMessage(ctx, testSystemPrompt, history, message, functions).
		Return(expectedMessages, expectedParts, nil).
		Times(1)

	messages, parts, err := llmInteractor.SendChatMessage(ctx, testSystemPrompt, history, message, functions)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	if len(parts) != 3 {
		t.Fatalf("Expected 3 parts, got %d", len(parts))
	}

	for i, msg := range messages {
		if msg == nil || *msg != *expectedMessages[i] {
			t.Errorf("Message %d: expected %v, got %v", i, *expectedMessages[i], *msg)
		}
	}
}

func TestLLMInteractorImpl_SendChatMessage_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGeminiRepository(ctrl)
	llmInteractor := interactor.NewLLMInteractor(mockRepo)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	history := []*domain.Content{}
	message := "Hello"
	functions := []domain.Function{}
	expectedError := context.Canceled

	mockRepo.EXPECT().
		SendChatMessage(ctx, testSystemPrompt, history, message, functions).
		Return(nil, nil, expectedError).
		Times(1)

	messages, parts, err := llmInteractor.SendChatMessage(ctx, testSystemPrompt, history, message, functions)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if messages != nil || parts != nil {
		t.Error("Expected nil messages and parts on error")
	}
}
