package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Request struct {
	ent.Schema
}

func (Request) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("initiator_id", uuid.UUID{}),
		field.UUID("entity_id", uuid.UUID{}),
		field.String("entity_type").
			MaxLen(50).
			NotEmpty(),
		field.Int("step").
			Default(0),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.String("status").
			MaxLen(50).
			NotEmpty().
			Default("PENDING"),
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

func (Request) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("initiator_id"),
		index.Fields("entity_id"),
		index.Fields("status"),
		index.Fields("step"),
	}
}
