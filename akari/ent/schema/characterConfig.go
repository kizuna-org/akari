package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CharacterConfig struct {
	ent.Schema
}

func (CharacterConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("name_regexp").Optional().Nillable().Comment("The regular expression to match the AI character's name"),
		field.String("default_system_prompt").NotEmpty().Comment("The default system prompt for this character"),
	}
}

func (CharacterConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("character", Character.Type).
			Ref("config").
			Unique().
			Comment("The character associated with this configuration"),
	}
}
