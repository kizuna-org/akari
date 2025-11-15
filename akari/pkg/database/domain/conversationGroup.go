package domain

//go:generate go tool mockgen -package=mock -source=conversationGroup.go -destination=mock/conversationGroup.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/ent"
)

type ConversationGroupRepository interface {
	CreateConversationGroup(ctx context.Context, characterID int) (*ConversationGroup, error)
	GetConversationGroupByID(ctx context.Context, id int) (*ConversationGroup, error)
	GetConversationGroupByCharacterID(ctx context.Context, characterID int) (*ConversationGroup, error)
	ListConversationGroups(ctx context.Context) ([]*ConversationGroup, error)
	DeleteConversationGroup(ctx context.Context, id int) error
}

type ConversationGroup = ent.ConversationGroup
