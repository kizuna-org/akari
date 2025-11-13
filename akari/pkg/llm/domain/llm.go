//go:generate mockgen -package=mock -source=llm.go -destination=mock/llm.go

package domain

import (
	"context"

	"google.golang.org/genai"
)

type GeminiRepository interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*Content,
		message string,
		functions []Function,
	) ([]*string, []*Part, error)
}

type (
	Content          = genai.Content
	Part             = genai.Part
	FunctionCall     = genai.FunctionCall
	FunctionResponse = genai.FunctionResponse
)

type Function struct {
	FunctionDeclaration *genai.FunctionDeclaration
	Function            func(ctx context.Context, request *FunctionCall) (map[string]any, error)
}
