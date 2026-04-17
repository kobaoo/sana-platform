package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Employee struct {
	ent.Schema
}

func (Employee) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Unique().
			Immutable(),
	}
}

func (Employee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("event_participations", EventParticipant.Type),
	}
}
