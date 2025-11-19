package domain

//go:generate go tool mockgen -package=mock -source=conversationGroup.go -destination=mock/conversationGroup.go

import (
	"context"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type ConversationGroupRepository interface {
	CreateConversationGroup(ctx context.Context, characterID int) (*ConversationGroup, error)
	GetConversationGroupByID(ctx context.Context, id int) (*ConversationGroup, error)
	GetConversationGroupByCharacterID(ctx context.Context, characterID int) (*ConversationGroup, error)
	ListConversationGroups(ctx context.Context) ([]*ConversationGroup, error)
	DeleteConversationGroup(ctx context.Context, id int) error
}

type ConversationGroup struct {
	ID        int
	Character *Character
	CreatedAt time.Time
}

func FromEntConversationGroup(entConversationGroup *ent.ConversationGroup) *ConversationGroup {
	if entConversationGroup == nil {
		return nil
	}

	var character *Character
	if entConversationGroup.Edges.Character != nil {
		character = FromEntCharacter(entConversationGroup.Edges.Character)
	}

	return &ConversationGroup{
		ID:        entConversationGroup.ID,
		Character: character,
		CreatedAt: entConversationGroup.CreatedAt,
	}
}
