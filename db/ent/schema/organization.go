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

type Organization struct {
	ent.Schema
}

func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.String("name").
			MaxLen(255).
			NotEmpty(),
		field.String("code").
			MaxLen(100).
			NotEmpty().
			Unique(),
		field.UUID("parent_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.String("type").
			MaxLen(50).
			NotEmpty().
			Default("subsidiary"),
		field.Bool("is_active").
			Default(true),
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
	}
}

func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parent", Organization.Type).
			Ref("children").
			Field("parent_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("children", Organization.Type),
	}
}

func (Organization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("parent_id"),
		index.Fields("code"),
	}
}
