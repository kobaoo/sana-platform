package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Request struct {
	ent.Schema
}

func (Request) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("initiator_id", uuid.UUID{}),
		field.UUID("parent_request_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("entity_id", uuid.UUID{}),
		field.String("entity_type").
			MaxLen(50).
			NotEmpty().
			Default("TRAINING_EVENT"),
		field.String("request_type").
			MaxLen(30).
			NotEmpty().
			Default("MAIN"),
		field.String("kind").
			MaxLen(30).
			NotEmpty().
			Default("REGULAR"),
		field.UUID("assigned_hr_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("target_dzo_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.String("title").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("category").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("format").
			MaxLen(50).
			Optional().
			Nillable(),
		field.UUID("responsible_admin_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Time("training_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Time("deadline_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Float("cost_amount").
			Optional().
			Nillable(),
		field.String("cost_mode").
			MaxLen(30).
			Optional().
			Nillable(),
		field.Int("step").
			Default(0),
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
		field.Time("completed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.String("status").
			MaxLen(50).
			NotEmpty().
			Default("DRAFT"),
	}
}

func (Request) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("initiator", User.Type).
			Ref("requests").
			Field("initiator_id").
			Unique().
			Required(),
		edge.From("parent", Request.Type).
			Ref("children").
			Field("parent_request_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("children", Request.Type),
	}
}

func (Request) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("initiator_id"),
		index.Fields("entity_id"),
		index.Fields("kind"),
		index.Fields("status"),
		index.Fields("step"),
		index.Fields("parent_request_id"),
		index.Fields("request_type"),
		index.Fields("assigned_hr_id"),
		index.Fields("target_dzo_id"),
	}
}
