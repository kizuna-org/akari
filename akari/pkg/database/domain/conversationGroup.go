package domain

//go:generate go tool mockgen -package=mock -source=conversationGroup.go -destination=mock/conversationGroup.go

import (
	"context"
	"errors"
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
	ID          int
	CharacterID int
	CreatedAt   time.Time
}

func FromEntConversationGroup(entConversationGroup *ent.ConversationGroup) (*ConversationGroup, error) {
	if entConversationGroup == nil {
		return nil, errors.New("conversationGroup is nil")
	}

	if entConversationGroup.Edges.Character == nil {
		return nil, errors.New("conversationGroup.Character edge is nil")
	}

	characterID := entConversationGroup.Edges.Character.ID

	return &ConversationGroup{
		ID:          entConversationGroup.ID,
		CharacterID: characterID,
		CreatedAt:   entConversationGroup.CreatedAt,
	}, nil
}
