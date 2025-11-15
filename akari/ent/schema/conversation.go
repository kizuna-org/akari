package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Conversation struct {
	ent.Schema
}

func (Conversation) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time when the bot response was created"),
	}
}

func (Conversation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("discord_message", DiscordMessage.Type).
			Unique().
			Immutable().
			Required().
			Comment("The Discord message that related to this conversation"),
		edge.From("conversation_group", ConversationGroup.Type).
			Ref("conversations").
			Unique().
			Required().
			Comment("The conversation group this conversation belongs to"),
	}
}

func (Conversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
	}
}
