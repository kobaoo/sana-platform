package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type ApplicationParticipant struct {
	ent.Schema
}

func (ApplicationParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("application_id", uuid.UUID{}),
		field.UUID("user_id", uuid.UUID{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (ApplicationParticipant) Edges() []ent.Edge {
	return nil
}

func (ApplicationParticipant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("application_id"),
		index.Fields("user_id"),
		index.Fields("application_id", "user_id").Unique(),
	}
}
