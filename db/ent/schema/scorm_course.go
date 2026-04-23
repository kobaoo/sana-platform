package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type ScormCourse struct {
	ent.Schema
}

func (ScormCourse) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		field.UUID("client_id", uuid.UUID{}),

		field.String("title").
			NotEmpty(),
		
		field.JSON("category_ids", []uuid.UUID{}),

		field.Text("description").
			Optional().
			Nillable(),

		field.String("lecturer").
			MaxLen(255).
			Optional().
			Nillable(),

		field.Text("scorm_url").
			NotEmpty(),

		field.Bool("is_active").
			Default(true),
	}
}

func (ScormCourse) Edges() []ent.Edge {
	return []ent.Edge{
		// Clients
		// edge.From("client", Client.Type).
		// 	Ref("courses").
		// 	Field("client_id").
		// 	Required().
		// 	Unique(),

		// Category
		// edge.From("category", Category.Type).
		// 	Ref("courses").
		// 	Field("category_id").
		// 	Required().
		// 	Unique(),

		//
		edge.To("course_progress", ScormProgress.Type),
	}
}
