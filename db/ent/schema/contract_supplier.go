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

type ContractSupplier struct {
	ent.Schema
}

func (ContractSupplier) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("supplier_id", uuid.UUID{}),
		field.String("contract_number").
			MaxLen(100).
			NotEmpty(),
		field.Int("vat_flag").
			Default(0).
			Min(0).
			Max(100),
		field.Time("signed_date").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("end_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.Float("amount_currency").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.String("currency").
			MaxLen(10).
			Optional().
			Nillable(),
		field.Float("balance_at_year_end").
			Optional().
			Nillable().
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
		field.Float("total_with_amendment").
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.Float("remaining_amount").
			SchemaType(map[string]string{
				dialect.Postgres: "decimal(14,2)",
			}),
		field.String("file_key").
			MaxLen(500).
			Optional().
			Nillable(),
		field.String("file_name").
			MaxLen(255).
			Optional().
			Nillable(),
		field.Int64("file_size").
			Optional().
			Nillable(),
		field.String("file_mime_type").
			MaxLen(100).
			Optional().
			Nillable(),
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}).
			Annotations(entsql.Default("NOW()")),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}).
			Annotations(entsql.Default("NOW()")),
	}
}

func (ContractSupplier) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("external_training_events", ExternalTrainingEvent.Type),
	}
}

func (ContractSupplier) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("supplier_id").StorageKey("contract_supplier_supplier_id"),
		index.Fields("contract_number").StorageKey("contract_supplier_contract_number"),
	}
}
