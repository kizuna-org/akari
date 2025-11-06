package schema

import "entgo.io/ent"

type SystemPrompt struct {
	ent.Schema
}

func (SystemPrompt) Fields() []ent.Field {
	return nil
}

func (SystemPrompt) Edges() []ent.Edge {
	return nil
}
