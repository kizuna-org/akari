package domain

//go:generate go tool mockgen -package=mock -source=conversation.go -destination=mock/conversation.go

import (
	"context"
	"errors"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type ConversationRepository interface {
	CreateConversation(ctx context.Context, params Conversation) (*Conversation, error)
	GetConversationByID(ctx context.Context, id int) (*Conversation, error)
	ListConversations(ctx context.Context) ([]*Conversation, error)
	DeleteConversation(ctx context.Context, id int) error
}

type Conversation struct {
	ID                  int
	UserID              int
	DiscordMessageID    string
	ConversationGroupID int
	CreatedAt           time.Time
}

func FromEntConversation(entConversation *ent.Conversation) (*Conversation, error) {
	if entConversation == nil {
		return nil, errors.New("conversation is nil")
	}

	if entConversation.Edges.User == nil {
		return nil, errors.New("conversation.User edge is nil")
	}

	userID := entConversation.Edges.User.ID

	if entConversation.Edges.DiscordMessage == nil {
		return nil, errors.New("conversation.DiscordMessage edge is nil")
	}

	discordMessageID := entConversation.Edges.DiscordMessage.ID

	if entConversation.Edges.ConversationGroup == nil {
		return nil, errors.New("conversation.ConversationGroup edge is nil")
	}

	conversationGroupID := entConversation.Edges.ConversationGroup.ID

	return &Conversation{
		ID:                  entConversation.ID,
		UserID:              userID,
		DiscordMessageID:    discordMessageID,
		ConversationGroupID: conversationGroupID,
		CreatedAt:           entConversation.CreatedAt,
	}, nil
}
