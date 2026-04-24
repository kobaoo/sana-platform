package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type ContractSupplierHistory struct {
	ent.Schema
}

func (ContractSupplierHistory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("history_id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("contract_id", uuid.UUID{}),
		field.String("operation_type").
			MaxLen(50).
			NotEmpty(),
		field.Time("changed_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.UUID("changed_by", uuid.UUID{}).
			Optional().
			Nillable(),
		field.JSON("snapshot", map[string]interface{}{}).
			Optional(),
		field.JSON("diff", map[string]interface{}{}).
			Optional(),
	}
}

func (ContractSupplierHistory) Edges() []ent.Edge {
	return nil
}

func (ContractSupplierHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("contract_id").StorageKey("contract_supplier_history_contract_id"),
		index.Fields("changed_at").StorageKey("contract_supplier_history_changed_at"),
	}
}
