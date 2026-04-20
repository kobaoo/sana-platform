package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Certificate struct {
	ent.Schema
}

func (Certificate) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.UUID("employee_id", uuid.UUID{}), // Изменено на UUID по ТЗ
		// field.UUID("dzo_id", uuid.UUID{}).Optional().Nillable(), // Закомментировал
		field.Enum("type").
			Values("EXTERNAL", "SCORM"), // Добавлено из Data Dictionary
		field.String("title").
			MaxLen(300). // VARCHAR(300) по ТЗ
			NotEmpty(),
		field.Time("issued_date"). // Переименовано
						SchemaType(map[string]string{
				dialect.Postgres: "date", // В ДД указан тип DATE
			}),
		field.Time("expiry_date"). // Переименовано
						Optional().
						Nillable().
						SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Text("file_url"). // Сделано Optional/Nillable
					Optional().
					Nillable(),
		field.UUID("uploaded_by", uuid.UUID{}).
			Optional().
			Nillable(), // Добавлено по ТЗ
		field.UUID("event_id", uuid.UUID{}).
			Optional().
			Nillable(), // Добавлено по ТЗ
		field.UUID("scorm_course_id", uuid.UUID{}).
			Optional().
			Nillable(), // Добавлено по ТЗ
		field.Enum("entity_type").
			Values("SCORM_COURSE", "TRAINING_EVENT"), // Добавлено
		field.UUID("entity_id", uuid.UUID{}), // Добавлено
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Certificate) Edges() []ent.Edge {
	return nil
}

func (Certificate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("employee_id"),
		index.Fields("is_active"),
	}
}
