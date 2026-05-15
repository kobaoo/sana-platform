package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type TrainingParticipant struct {
	ent.Schema
}

func (TrainingParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("event_id", uuid.UUID{}),
		field.UUID("employee_id", uuid.UUID{}),
		field.String("status"),
		field.UUID("certificate_id", uuid.UUID{}).Optional().Nillable(),
	}
}

// разкомментить после создания schemas Employee, Event
// func (TrainingParticipant) Edges() []ent.Edge {
// 	return []ent.Edge{
// // 		// ==========================================
// // 		// EVENT (event.id)
// // 		// ==========================================
// 		edge.From("event", TrainingEvent.Type).
// 			Ref("participants").
// 			Field("event_id").
// 			Unique().
// 			Required(),
// // 		// ==========================================
// // 		// EMPLOYEE (employee.id)
// // 		// ==========================================
// 		edge.From("employee", Employee.Type).
// 			Ref("training_participations").
// 			Field("employee_id").
// 			Unique().
// 			Required(),
// 	}
// }