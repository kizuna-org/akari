package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type DiscordChannel struct {
	ent.Schema
}

// https://discord.com/developers/docs/resources/channel
func (DiscordChannel) Fields() []ent.Field {
	discordChannelType := []string{
		"GUILD_TEXT",
		"DM",
		"GUILD_VOICE",
		"GROUP_DM",
		"GUILD_CATEGORY",
		"GUILD_ANNOUNCEMENT",
		"ANNOUNCEMENT_THREAD",
		"PUBLIC_THREAD",
		"PRIVATE_THREAD",
		"GUILD_STAGE_VOICE",
		"GUILD_DIRECTORY",
		"GUILD_FORUM",
		"GUILD_MEDIA",
	}

	return []ent.Field{
		field.String("id").NotEmpty().Immutable().Unique().Comment("the id of this channel"),
		field.Enum("type").Values(discordChannelType...).Comment("the type of channel"),
		field.String("name").NotEmpty().Comment("the name of the channel"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time when the record was created in the database"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("The time when the channel name was last updated"),
	}
}

func (DiscordChannel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("messages", DiscordMessage.Type).
			Ref("channel").
			Comment("the messages sent in this channel"),
		edge.To("guild", DiscordGuild.Type).
			Required().
			Unique().
			Comment("the guild this channel belongs to"),
	}
}
