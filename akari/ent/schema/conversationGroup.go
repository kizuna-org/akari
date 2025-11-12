package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ConversationGroup struct {
	ent.Schema
}

func (ConversationGroup) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time when the conversation group was created"),
	}
}

func (ConversationGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("conversations", Conversation.Type).
			Immutable().
			Comment("Conversations in this conversation group"),
	}
}

func (ConversationGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
	}
}
