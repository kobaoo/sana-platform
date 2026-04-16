package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Request struct {
	ent.Schema
}

func (Request) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("initiator_id", uuid.UUID{}),
		field.UUID("entity_id", uuid.UUID{}),
		field.String("entity_type"),
		field.Int("step"),
		field.Time("created_at").Default(time.Now),
		field.String("status"),
	}
}

func (Request) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("initiator", User.Type).
			Ref("requests").
			Field("initiator_id").
			Unique().
			Required(),
	}
}