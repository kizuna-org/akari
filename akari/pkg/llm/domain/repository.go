package domain

import (
	"context"
)

// GeminiRepository はGemini APIとのやり取りを抽象化するリポジトリインターフェース.
type GeminiRepository interface {
	SendChatMessage(
		ctx context.Context,
		systemPrompt string,
		history []*Content,
		message string,
		functions []Function,
	) ([]*string, []*Part, error)
}
