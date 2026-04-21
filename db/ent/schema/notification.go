package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Notification struct {
	ent.Schema
}

func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("user_id", uuid.UUID{}),
		field.Enum("type").
			Values(
				"CERT_EXPIRING",
				"CERT_EXPIRED",
				"REQUEST_CREATED",
				"REQUEST_STEP_UPDATED",
				"REQUEST_APPROVED",
				"REQUEST_CANCELLED",
			),
		field.Enum("entity_type").
			Values(
				"CERTIFICATE",
				"REQUEST",
			),
		field.UUID("entity_id", uuid.UUID{}),
		field.Enum("status").
			Values("PENDING", "SENT", "FAILED").
			Default("PENDING"),
		field.Time("sent_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (Notification) Edges() []ent.Edge {
	return nil
}

func (Notification) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "type", "entity_type", "entity_id").Unique(),
		index.Fields("user_id"),
		index.Fields("status"),
	}
}
