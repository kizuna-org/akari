package domain

//go:generate go tool mockgen -package=mock -source=conversation.go -destination=mock/conversation.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type ConversationRepository interface {
	CreateConversation(ctx context.Context, messageID string, conversationGroupID *int) (*Conversation, error)
	GetConversationByID(ctx context.Context, id int) (*Conversation, error)
	ListConversations(ctx context.Context) ([]*Conversation, error)
	DeleteConversation(ctx context.Context, id int) error
}

type Conversation struct {
	ID                int
	User              *AkariUser
	DiscordMessage    *DiscordMessage
	ConversationGroup *ConversationGroup
	CreatedAt         time.Time
}

func FromEntConversation(entConversation *ent.Conversation) *Conversation {
	if entConversation == nil {
		return nil
	}

	var user *AkariUser
	if entConversation.Edges.User != nil {
		user = FromEntAkariUser(entConversation.Edges.User)
	}

	var discordMessage *DiscordMessage
	if entConversation.Edges.DiscordMessage != nil {
		discordMessage = FromEntDiscordMessage(entConversation.Edges.DiscordMessage)
	}

	var conversationGroup *ConversationGroup
	if entConversation.Edges.ConversationGroup != nil {
		conversationGroup = FromEntConversationGroup(entConversation.Edges.ConversationGroup)
	}

	return &Conversation{
		ID:                entConversation.ID,
		User:              user,
		DiscordMessage:    discordMessage,
		ConversationGroup: conversationGroup,
		CreatedAt:         entConversation.CreatedAt,
	}
}
