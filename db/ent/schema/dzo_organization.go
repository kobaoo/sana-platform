package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type DzoOrganization struct {
	ent.Schema
}

func (DzoOrganization) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "dzo_organizations"},
	}
}

func (DzoOrganization) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		field.UUID("client_id", uuid.UUID{}),

		field.String("name").
			MaxLen(300).
			NotEmpty(),

		field.String("short_name").
			MaxLen(100).
			Optional().
			Nillable(),

		field.String("bin").
			MaxLen(12).
			Optional().
			Nillable(),

		field.Bool("is_active").
			Default(true),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
func (DzoOrganization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("employees", Employee.Type),
	}
}
