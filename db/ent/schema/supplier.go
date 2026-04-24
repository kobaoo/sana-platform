package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Supplier struct {
	ent.Schema
}

func (Supplier) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("client_id", uuid.UUID{}),
		field.Enum("type").
			Values(
				"LEGAL",
				"INDIVIDUAL",
			),
		field.String("name").
			MaxLen(300).
			NotEmpty(),
		field.String("bin_or_iin").
			MaxLen(12).
			Optional().
			Nillable().
			Unique(),
		field.Float("local_content_pct").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(5,2)",
			}),
		field.Bool("is_active").
			Default(true),
	}
}

func (Supplier) Edges() []ent.Edge {
	return nil
}

func (Supplier) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("client_id"),
		index.Fields("type"),
		index.Fields("is_active"),
	}
}
