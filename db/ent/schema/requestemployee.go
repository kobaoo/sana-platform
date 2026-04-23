package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type RequestEmployee struct {
	ent.Schema
}

func (RequestEmployee) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("request_id", uuid.UUID{}),
		field.UUID("employee_id", uuid.UUID{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (RequestEmployee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id"),
		index.Fields("employee_id"),
	}
}
