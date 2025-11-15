package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type AkariUser struct {
	ent.Schema
}

func (AkariUser) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time when the record was created in the database"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("The time when the user was last updated"),
	}
}

func (AkariUser) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("discord_user", DiscordUser.Type).
			Unique().
			Comment("The Discord user linked to this Akari user"),
		edge.From("conversations", Conversation.Type).
			Ref("user").
			Comment("conversations associated with this Akari user"),
	}
}
