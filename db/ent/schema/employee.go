package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Employee struct {
	ent.Schema
}

func (Employee) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "employees"},
	}
}

func (Employee) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		field.UUID("client_id", uuid.UUID{}),

		field.UUID("dzo_id", uuid.UUID{}),

		field.UUID("dzo_position_id", uuid.UUID{}).
			Optional().
			Nillable(),

		field.String("full_name").
			MaxLen(300).
			NotEmpty(),

		field.String("short_name").
			MaxLen(100).
			Optional().
			Nillable(),

		field.String("department").
			MaxLen(200).
			Optional().
			Nillable(),

		field.String("direction").
			MaxLen(100).
			Optional().
			Nillable(),

		field.String("email").
			MaxLen(255).
			NotEmpty(),

		field.String("internal_phone").
			MaxLen(20).
			Optional().
			Nillable(),

		field.Time("birth_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				"postgres": "date",
			}),

		field.Bool("is_active").
			Default(true),

		field.UUID("user_id", uuid.UUID{}).
			Optional().
			Nillable(),

		field.Bool("is_deleted").
			Default(false),
	}
}
func (Employee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("dzo", DzoOrganization.Type).
			Ref("employees").
			Field("dzo_id").
			Required().
			Unique(),

		edge.From("dzo_position_title", DzoPositionTitle.Type).
			Ref("employees").
			Field("dzo_position_id").
			Unique(),

		edge.To("event_participations", EventParticipant.Type),
	}

}
