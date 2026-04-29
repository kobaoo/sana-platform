package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type ScormProgress struct {
	ent.Schema
}

func (ScormProgress) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		field.UUID("course_id", uuid.UUID{}),

		field.UUID("employee_id", uuid.UUID{}),

		field.Enum("status").
			Values("NOT_STARTED", "IN_PROGRESS", "COMPLETED").
			Default("NOT_STARTED"),

		field.Int("score").
			Optional().
			Nillable(),

		field.Time("completed_at").
			Optional().
			Nillable(),
		field.Time("start_date").
			Optional().
			Nillable(),
		field.Time("end_date").
			Optional().
			Nillable(),
		field.Text("suspend_data").
			Optional().
			Nillable(),
	}
}
func (ScormProgress) Edges() []ent.Edge {
	return []ent.Edge{

		edge.From("progress", ScormCourse.Type).
			Ref("course_progress").
			Unique().
			Field("course_id").
			Required(),

		// edge.From("employee", Employee.Type).
		// 	Ref("progresses").
		// 	Field("employee_id"),
	}
}
