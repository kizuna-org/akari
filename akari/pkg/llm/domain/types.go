package domain

import (
	"context"

	"google.golang.org/genai"
)

// 型エイリアス: 外部ライブラリの型をドメインでラップ.
type (
	Content          = genai.Content
	Part             = genai.Part
	FunctionCall     = genai.FunctionCall
	FunctionResponse = genai.FunctionResponse
)

// Function はLLMで使用可能な関数を表すドメインモデル.
type Function struct {
	FunctionDeclaration *genai.FunctionDeclaration
	Function            func(ctx context.Context, request *FunctionCall) (map[string]any, error)
}
