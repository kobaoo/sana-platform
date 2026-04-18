package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Certificate struct {
	ent.Schema
}

func (Certificate) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.Int64("employee_id"),
		field.Int64("dzo_id").Optional(), // Добавлено для фильтрации по организациям
		field.String("title").
			MaxLen(255).
			NotEmpty(),
		field.String("file_url").
			NotEmpty(),
		field.Time("issue_date").
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Time("expiration_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
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

func (Certificate) Edges() []ent.Edge {
	return nil
}

func (Certificate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("employee_id"),
		index.Fields("dzo_id"),
		index.Fields("is_active"),
	}
}
