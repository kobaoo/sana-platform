package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type EventParticipant struct {
	ent.Schema
}

func (EventParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("event_id", uuid.UUID{}),
		field.UUID("employee_id", uuid.UUID{}),
		field.UUID("reviewed_by", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Time("joined_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Enum("attendance_status").
			Values("PENDING", "ATTENDED", "MISSED").
			Default("PENDING"),
		field.Time("reviewed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (EventParticipant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("event", Event.Type).
			Field("event_id").
			Ref("participants").
			Unique().
			Required(),

		edge.From("employee", Employee.Type).
			Field("employee_id").
			Ref("event_participations").
			Unique().
			Required(),

		edge.From("reviewer", User.Type).
			Field("reviewed_by").
			Ref("reviewed_participations").
			Unique(),
	}
}
