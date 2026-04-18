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

type Course struct {
	ent.Schema
}

func (Course) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.String("title").
			MaxLen(255).
			NotEmpty(),
		field.Text("description").
			Optional().
			Nillable(),
		field.String("format").
			MaxLen(50).
			Optional().
			Nillable(),
		field.String("category").
			MaxLen(100).
			Optional().
			Nillable(),
		field.Bool("is_external").
			Default(true),
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (Course) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("modules", CourseModule.Type),
	}
}

func (Course) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("title"),
		index.Fields("is_active"),
	}
}
