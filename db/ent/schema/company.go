package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Company struct {
	ent.Schema
}

// Annotations forces the table name to 'clients'.
func (Company) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "clients"},
	}
}

func (Company) Fields() []ent.Field {
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
		field.String("language").
			MaxLen(10).
			Optional().
			Nillable(),
		field.Int("user_limit").
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

func (Company) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type),
		edge.To("events", Event.Type),
	}
}
