package domain

//go:generate go tool mockgen -package=mock -source=conversation.go -destination=mock/conversation.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type ConversationRepository interface {
	CreateConversation(ctx context.Context, messageID string, conversationGroupID *int) (*Conversation, error)
	GetConversationByID(ctx context.Context, id int) (*Conversation, error)
	ListConversations(ctx context.Context) ([]*Conversation, error)
	DeleteConversation(ctx context.Context, id int) error
}

type Conversation = ent.Conversation
