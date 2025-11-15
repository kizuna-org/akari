package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Character struct {
	ent.Schema
}

func (Character) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Unique().Comment("The name of the AI character"),
		field.Time("created_at").Immutable().Default(time.Now).Comment("The time when the character was created"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("The time when the character was last updated"),
	}
}

func (Character) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("config", CharacterConfig.Type).
			Unique().
			Required().
			Comment("The configuration associated with this character"),
		edge.To("system_prompts", SystemPrompt.Type).
			Required().
			Comment("The system prompts associated with this character"),
		edge.From("conversation_groups", ConversationGroup.Type).
			Ref("character").
			Comment("The conversation groups associated with this character"),
	}
}
