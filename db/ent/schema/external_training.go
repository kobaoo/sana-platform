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

type ExternalTrainingEvent struct {
	ent.Schema
}

func (ExternalTrainingEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "external_training_events"},
	}
}

func (ExternalTrainingEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),

		field.String("name").
			NotEmpty().
			Comment("Название обучения"),

		field.String("format").
			Optional().
			Nillable().
			Comment("Формат: онлайн, офлайн"),

		field.Int("capacity").
			Optional().
			Nillable().
			Comment("Количество мест"),

		field.Float("supplier_cost_vat").
			Optional().
			Nillable().
			Comment("Внутренний бюджет — сумма к выплате поставщику"),

		field.Time("start_date").
			Comment("Дата и время проведения"),

		field.Bool("is_active").
			Default(true),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.UUID("category_id", uuid.UUID{}).
			Optional().
			Nillable(),

		field.Bool("is_deleted").
			Default(false),
		field.UUID("supplier_id", uuid.UUID{}),

		field.UUID("contract_id", uuid.UUID{}),

		field.UUID("responsible_user_id", uuid.UUID{}).
			Optional().
			Nillable(),
	}
}

func (ExternalTrainingEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("category", Category.Type).
			Ref("external_training_events").
			Field("category_id").
			Unique(),

		edge.From("supplier", Supplier.Type).
			Ref("external_training_events").
			Field("supplier_id").
			Unique().
			Required(),

		edge.From("contract", ContractSupplier.Type).
			Ref("external_training_events").
			Field("contract_id").
			Unique().
			Required(),

		edge.From("responsible_user", User.Type).
			Ref("responsible_external_training_events").
			Field("responsible_user_id").
			Unique(),
	}
}