package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type DiscordGuild struct {
	ent.Schema
}

// https://discord.com/developers/docs/resources/guild
func (DiscordGuild) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable().Unique().Comment("guild id"),
		field.String("name").NotEmpty().Comment("guild name"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time when the record was created in the database"),
	}
}

func (DiscordGuild) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("channels", DiscordChannel.Type).
			Ref("guild").
			Comment("the channels belonging to this guild"),
	}
}
