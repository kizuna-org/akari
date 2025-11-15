package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type DiscordMessage struct {
	ent.Schema
}

// https://discord.com/developers/docs/resources/message
func (DiscordMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable().Unique().Comment("id of the message"),
		field.String("content").Immutable().Comment("contents of the message"),
		field.Time("timestamp").Immutable().Default(time.Now).Comment("when this message was sent"),
		field.Strings("mentions").Optional().Immutable().Comment("users specifically mentioned in the message"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time when the record was created in the database"),
	}
}

func (DiscordMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("author", DiscordUser.Type).
			Required().
			Unique().
			Comment("the author of this message"),
		edge.To("channel", DiscordChannel.Type).
			Required().
			Unique().
			Comment("the channel this message was sent in"),
		edge.From("conversation_message", Conversation.Type).
			Ref("discord_message").
			Unique().
			Comment("the conversation this message relates to"),
	}
}
