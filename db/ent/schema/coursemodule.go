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

type CourseModule struct {
	ent.Schema
}

func (CourseModule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("course_id", uuid.UUID{}),
		field.String("title").
			MaxLen(255).
			NotEmpty(),
		field.Text("description").
			Optional().
			Nillable(),
		field.Int("sort_order").
			Default(0),
		field.Int("duration_minutes").
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
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (CourseModule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("course", Course.Type).
			Ref("modules").
			Field("course_id").
			Unique().
			Required(),
	}
}

func (CourseModule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("course_id"),
		index.Fields("course_id", "sort_order"),
	}
}
