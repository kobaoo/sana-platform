// db/ent/schema/generalposition.go
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

type GeneralPosition struct {
	ent.Schema
}

func (GeneralPosition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "general_positions"},
	}
}

func (GeneralPosition) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		field.String("name").
			MaxLen(255).
			NotEmpty(),

		field.Text("description").
			Optional().
			Nillable(),

		field.Bool("is_deleted").
			Default(false),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (GeneralPosition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("dzo_position_titles", DzoPositionTitle.Type),
	}
}
