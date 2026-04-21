package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Notification определяет Ent-схему для таблицы уведомлений.
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
			Values("CERT_EXPIRING", "CERT_EXPIRED"),
		field.Enum("entity_type").
			Values("CERTIFICATE"),
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
		// Составной уникальный индекс — анти-дублирующий ключ.
		index.Fields("user_id", "type", "entity_type", "entity_id").Unique(),
		index.Fields("user_id"),
		index.Fields("status"),
	}
}
