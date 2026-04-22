package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type TrainingEvent struct {
	ent.Schema
}

func (TrainingEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("title"),
		field.Time("start_date"),
		field.Time("end_date"),
		field.String("location_type"),
		field.String("location_city").Optional().Nillable(),

		field.UUID("category_id", uuid.UUID{}),
		field.String("direction").Optional().Nillable(),
		field.UUID("dzo_id", uuid.UUID{}),

		field.UUID("dzo_contract_id", uuid.UUID{}).Optional().Nillable(),
		field.Int("participants_count"),

		field.Float("cost_per_person_vat").Optional().Nillable(),
		field.Float("cost_group_vat").Optional().Nillable(),
		field.Float("kyu_hourly_rate").Optional().Nillable(),

		field.UUID("supplier_id", uuid.UUID{}).Optional().Nillable(),
		field.UUID("supplier_contract_id", uuid.UUID{}).Optional().Nillable(),

		field.Float("supplier_cost_vat").Optional().Nillable(),
		field.Float("supplier_cost_currency").Optional().Nillable(),
		field.String("supplier_currency").Optional().Nillable(),
		field.Float("local_content_pct").Optional().Nillable(),
	}
}

// разкомментить после создания schemas Category, DZO Organization, DZO Contract, Supplier, Sup contract, TrainingParticipants
// func (TrainingEvent) Edges() []ent.Edge {
// 	return []ent.Edge{
// 		// ==========================================
// 		// CATEGORY (categories.id)
// 		// ==========================================
// 		edge.From("category", Category.Type).
// 			Ref("training_events").
// 			Field("category_id").
// 			Unique().
// 			Required(),

// 		// ==========================================
// 		// DZO ORGANIZATION (dzo_organizations.id)
// 		// ==========================================
// 		edge.From("dzo", DzoOrganization.Type).
// 			Ref("training_events").
// 			Field("dzo_id").
// 			Unique().
// 			Required(),

// 		// ==========================================
// 		// DZO CONTRACT (contracts_dzo.id)
// 		// ==========================================
// 		edge.From("dzo_contract", ContractDzo.Type).
// 			Ref("training_events").
// 			Field("dzo_contract_id").
// 			Unique(),

// 		// ==========================================
// 		// SUPPLIER (suppliers.id)
// 		// ==========================================
// 		edge.From("supplier", Supplier.Type).
// 			Ref("training_events").
// 			Field("supplier_id").
// 			Unique(),

// 		// ==========================================
// 		// SUPPLIER CONTRACT (contracts_suppliers.id)
// 		// ==========================================
// 		edge.From("supplier_contract", ContractSupplier.Type).
// 			Ref("training_events").
// 			Field("supplier_contract_id").
// 			Unique(),

// 		// ==========================================
// 		// PARTICIPANTS (1 event -> many participants)
// 		// ==========================================
// 		edge.To("participants", TrainingParticipant.Type),
// 	}
// }