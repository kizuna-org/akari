package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SystemPrompt struct {
	ent.Schema
}

func (SystemPrompt) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").NotEmpty().Comment("An identifier title for the system prompt"),
		field.Enum("purpose").Values("text_chat").Comment("The purpose of the system prompt"),
		field.String("prompt").NotEmpty().Comment("The system prompt"),
		field.Strings("previous_prompts").Comment("Previous versions of the system prompt"),
		field.Int("version").Default(1).Comment("Version number of the system prompt"),
		field.Time("created_at").Immutable().Default(time.Now).Comment("The time when the system prompt was created"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("The time when the system prompt was last updated"),
	}
}

func (SystemPrompt) Edges() []ent.Edge {
	return nil
}
