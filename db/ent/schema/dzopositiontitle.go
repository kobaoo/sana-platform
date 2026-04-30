// db/ent/schema/dzopositiontitle.go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type DzoPositionTitle struct {
	ent.Schema
}

func (DzoPositionTitle) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "dzo_position_titles"},
	}
}

func (DzoPositionTitle) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		field.UUID("dzo_id", uuid.UUID{}),

		field.UUID("client_id", uuid.UUID{}),

		field.UUID("general_position_id", uuid.UUID{}).
			Optional().
			Nillable(),

		field.String("local_title").
			MaxLen(255).
			NotEmpty(),

		field.Bool("is_active").
			Default(true),

		field.Bool("is_deleted").
			Default(false),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (DzoPositionTitle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("dzo", DzoOrganization.Type).
			Ref("position_titles").
			Field("dzo_id").
			Required().
			Unique(),

		edge.From("general_position", GeneralPosition.Type).
			Ref("dzo_position_titles").
			Field("general_position_id").
			Unique(),

		edge.To("employees", Employee.Type),
	}
}

func (DzoPositionTitle) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("dzo_id", "local_title").
			Unique().
			Annotations(
				entsql.IndexWhere("is_deleted = false"),
			),
	}
}
