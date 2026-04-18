package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Application struct {
	ent.Schema
}

func (Application) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.String("kind").
			MaxLen(20).
			NotEmpty().
			Default("regular"),
		field.String("status").
			MaxLen(20).
			NotEmpty().
			Default("draft"),
		field.UUID("dzo_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("created_by_user_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("course_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.String("requested_course_name").
			MaxLen(255).
			NotEmpty(),
		field.String("expense_category").
			MaxLen(100).
			Optional().
			Nillable(),
		field.Text("comment").
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

func (Application) Edges() []ent.Edge {
	return nil
}

func (Application) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("kind"),
		index.Fields("status"),
		index.Fields("dzo_id"),
		index.Fields("created_by_user_id"),
		index.Fields("course_id"),
	}
}
