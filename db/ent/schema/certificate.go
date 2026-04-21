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
		field.UUID("employee_id", uuid.UUID{}),
		field.Enum("type").
			Values("EXTERNAL", "SCORM", "INTERNAL"),
		field.String("title").
			MaxLen(300).
			NotEmpty(),
		field.Time("issued_date").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Time("expiry_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Text("file_url").
			Optional().
			Nillable(),
		field.UUID("uploaded_by", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Enum("entity_type").
			Values("SCORM_COURSE", "TRAINING_EVENT"),
		field.UUID("entity_id", uuid.UUID{}),
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
		index.Fields("expiry_date"),
	}
}
