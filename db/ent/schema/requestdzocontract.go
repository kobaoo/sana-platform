package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type RequestDzoContract struct {
	ent.Schema
}

func (RequestDzoContract) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("request_id", uuid.UUID{}),
		field.UUID("dzo_id", uuid.UUID{}),
		field.String("file_name").
			MaxLen(255).
			NotEmpty(),
		field.String("file_url").
			MaxLen(1024).
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (RequestDzoContract) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id"),
		index.Fields("dzo_id"),
	}
}
