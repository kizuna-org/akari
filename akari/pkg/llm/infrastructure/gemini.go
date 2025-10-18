package infrastructure

import (
	"context"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/samber/lo"
	"google.golang.org/genai"
)

const (
	temperature = 1.0
)

type Function struct {
	FunctionDeclaration *genai.FunctionDeclaration
	Function            func(ctx context.Context, request *genai.FunctionCall) (map[string]any, error)
}

type GeminiModel interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*genai.Content,
		message string,
		functions []Function,
	) ([]*string, []*genai.Part, error)
}

type GeminiModelImpl struct {
	client *genai.Client
	logger *slog.Logger
	model  string
}

func NewGeminiModel(cfg config.ConfigRepository, logger *slog.Logger) (GeminiModel, error) {
	ctx := context.Background()
	config := cfg.GetConfig()

	//nolint:exhaustruct
	client, err := genai.NewClient(
		ctx,
		&genai.ClientConfig{
			Project:  config.LLM.ProjectID,
			Location: config.LLM.Location,
			Backend:  genai.BackendVertexAI,
		},
	)
	if err != nil {
		return nil, err
	}

	return &GeminiModelImpl{
		client: client,
		logger: logger.With("component", "gemini_model"),
		model:  config.LLM.ModelName,
	}, nil
}

//nolint:cyclop,funlen
func (m *GeminiModelImpl) SendChatMessage(
	ctx context.Context,
	systemPrompt string,
	history []*genai.Content,
	message string,
	functions []Function,
) ([]*string, []*genai.Part, error) {
	chat, err := m.client.Chats.Create(ctx, m.model, m.createConfig(systemPrompt, functions), history)
	if err != nil {
		return nil, nil, err
	}

	//nolint:exhaustruct
	res, err := chat.SendMessage(ctx, genai.Part{Text: message})
	if err != nil {
		return nil, nil, err
	}

	messages := make([]*string, 0)
	parts := make([]*genai.Part, 0)

	for {
		if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
			m.logger.Info("Model response is empty")

			break
		}

		messages = append(messages, lo.ToPtr(res.Candidates[0].Content.Parts[0].Text))
		parts = append(parts, res.Candidates[0].Content.Parts[0])

		m.logger.Info("res", "res", lo.Map(res.Candidates, func(candidate *genai.Candidate, _ int) []genai.Part {
			return lo.Map(candidate.Content.Parts, func(part *genai.Part, _ int) genai.Part {
				return *part
			})
		}))

		var functionResponses []genai.Part

		for _, part := range res.Candidates[0].Content.Parts {
			if part.FunctionCall == nil {
				continue
			}

			var result map[string]any

			for _, function := range functions {
				if function.FunctionDeclaration.Name == part.FunctionCall.Name {
					result, err = function.Function(ctx, part.FunctionCall)

					if err != nil {
						m.logger.Error("Failed to execute function", "error", err)
					}

					break
				}
			}

			//nolint:exhaustruct
			functionResponses = append(functionResponses, genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     part.FunctionCall.Name,
					Response: result,
				},
			})

			res, err = chat.SendMessage(ctx, functionResponses...)
			if err != nil {
				m.logger.Error("Failed to send function response", "error", err)
			}
		}

		if len(functionResponses) == 0 {
			break
		}
	}

	return messages, parts, nil
}

func (m *GeminiModelImpl) createConfig(systemPrompt string, functions []Function) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](temperature),
		SystemInstruction: &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{
					Text: systemPrompt,
				},
			},
		},
	}

	if len(functions) > 0 {
		functionDeclarations := make([]*genai.FunctionDeclaration, len(functions))
		for i, function := range functions {
			functionDeclarations[i] = function.FunctionDeclaration
		}

		config.Tools = []*genai.Tool{
			{
				FunctionDeclarations: functionDeclarations,
			},
		}
	}

	return config
}
