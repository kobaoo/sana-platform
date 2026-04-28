package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type RequestTargetDzo struct {
	ent.Schema
}

func (RequestTargetDzo) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("request_id", uuid.UUID{}),
		field.UUID("dzo_id", uuid.UUID{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (RequestTargetDzo) Edges() []ent.Edge {
	return nil
}

func (RequestTargetDzo) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id"),
		index.Fields("dzo_id"),
		index.Fields("request_id", "dzo_id").Unique(),
	}
}
