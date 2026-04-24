package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Event struct {
	ent.Schema
}

func (Event) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("client_id", uuid.UUID{}),
		field.UUID("host_id", uuid.UUID{}),
		field.String("title").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.String("zoom_link").
			NotEmpty(),
		field.Time("event_date").
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Int("max_participants").
			Positive(),
		field.String("materials_url").
			Optional().
			Nillable(),
		field.Enum("status").
			Values("ACTIVE", "COMPLETED", "CANCELLED").
			Default("ACTIVE"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Event) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("clients", Company.Type).
			Field("client_id").
			Ref("events").
			Unique().
			Required(),

		edge.From("host", User.Type).
			Field("host_id").
			Ref("hosted_events").
			Unique().
			Required(),

		edge.To("participants", EventParticipant.Type),
	}
}
