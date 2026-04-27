package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Category struct {
	ent.Schema
}

func (Category) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),

		field.String("name").
			NotEmpty(),

		field.String("description").
			Optional().
			Nillable(),
	}
}

func (Category) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("external_training_events", ExternalTrainingEvent.Type),
	}
}