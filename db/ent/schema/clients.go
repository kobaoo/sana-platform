package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Clients struct {
	ent.Schema
}

func (Clients) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.String("name").
			MaxLen(255).
			NotEmpty(),
		field.String("domain").
			MaxLen(100).
			Optional().
			Nillable(),
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (Clients) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("events", Event.Type),
	}
}
