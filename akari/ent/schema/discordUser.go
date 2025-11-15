package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type DiscordUser struct {
	ent.Schema
}

// https://discord.com/developers/docs/resources/user
func (DiscordUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable().Unique().Comment("the user's id"),
		field.String("username").NotEmpty().Comment("the user's username, not unique across the platform"),
		field.Bool("bot").Immutable().Default(false).Comment("whether the user belongs to an OAuth2 application"),
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

func (DiscordUser) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("akari_user", AkariUser.Type).
			Ref("discord_user").
			Unique().
			Required().
			Comment("The Akari user linked to this Discord user"),
		edge.From("messages", DiscordMessage.Type).
			Ref("author").
			Comment("messages sent by this user"),
	}
}
