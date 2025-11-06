package gemini

import (
	"context"
	"log/slog"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/llm/domain"
	"github.com/samber/lo"
	"google.golang.org/genai"
)

const (
	temperature = 1.0
)

type repositoryImpl struct {
	client *genai.Client
	logger *slog.Logger
	model  string
}

func NewRepository(cfg config.ConfigRepository, logger *slog.Logger) (domain.GeminiRepository, error) {
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

	return &repositoryImpl{
		client: client,
		logger: logger.With("component", "gemini_repository"),
		model:  config.LLM.ModelName,
	}, nil
}

//nolint:cyclop,funlen
func (r *repositoryImpl) SendChatMessage(
	ctx context.Context,
	systemPrompt string,
	history []*domain.Content,
	message string,
	functions []domain.Function,
) ([]*string, []*domain.Part, error) {
	chat, err := r.client.Chats.Create(ctx, r.model, r.createConfig(systemPrompt, functions), history)
	if err != nil {
		return nil, nil, err
	}

	//nolint:exhaustruct
	res, err := chat.SendMessage(ctx, genai.Part{Text: message})
	if err != nil {
		return nil, nil, err
	}

	messages := make([]*string, 0)
	parts := make([]*domain.Part, 0)

	for {
		if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
			r.logger.Info("Model response is empty")

			break
		}

		messages = append(messages, lo.ToPtr(res.Candidates[0].Content.Parts[0].Text))
		parts = append(parts, res.Candidates[0].Content.Parts[0])

		r.logger.Info("res", "res", lo.Map(res.Candidates, func(candidate *genai.Candidate, _ int) []genai.Part {
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
						r.logger.Error("Failed to execute function", "error", err)
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
				r.logger.Error("Failed to send function response", "error", err)
			}
		}

		if len(functionResponses) == 0 {
			break
		}
	}

	return messages, parts, nil
}

func (r *repositoryImpl) createConfig(systemPrompt string, functions []domain.Function) *genai.GenerateContentConfig {
	//nolint:exhaustruct
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
