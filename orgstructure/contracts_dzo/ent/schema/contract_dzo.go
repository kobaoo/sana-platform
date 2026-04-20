package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type ContractDZO struct {
	ent.Schema
}

func (ContractDZO) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("dzo_id", uuid.UUID{}),
		field.String("contract_number").
			MaxLen(100).
			NotEmpty(),
		field.String("category").
			MaxLen(100).
			NotEmpty(),
		field.Time("signed_date").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("expiry_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Float("amount_with_vat").
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.String("amendment_number").
			MaxLen(100).
			Optional().
			Nillable(),
		field.Time("amendment_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Float("amendment_amount").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.Float("total_amount").
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.Float("spent_amount").
			Default(0).
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.Float("remaining_amount").
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
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

func (ContractDZO) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("dzo_id"),
		index.Fields("is_active"),
		index.Fields("remaining_amount"),
	}
}
