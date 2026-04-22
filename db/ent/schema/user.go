package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Unique().
			Immutable(),
		field.String("keycloak_user_id").
			MaxLen(255).
			NotEmpty().
			Unique().
			Comment("Keycloak subject (sub claim)"),
		field.String("email").
			MaxLen(255).
			NotEmpty(),
		field.String("role").
			MaxLen(20).
			NotEmpty().
			Default("EMP").
			Comment("Business role: SA, ADM, HR, EMP"),
		field.UUID("dzo_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("DZO organization reference (from Keycloak dzoId claim)"),
		field.Bool("is_active").
			Default(true),
		field.Bool("is_onboarded").
			Default(true).
			Comment("True once the user has completed onboarding (first login after SA registration). " +
				"Pending admins created via RegisterAdmin start as false and are auto-activated on first login."),
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
		field.UUID("client_id", uuid.UUID{}).
			Comment("Reference to clients.id").
			Optional(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("client", Company.Type).
			Ref("users").
			Field("client_id").
			Unique(),
		edge.To("hosted_events", Event.Type),
		edge.To("reviewed_participations", EventParticipant.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email"),
		index.Fields("dzo_id"),
	}
}
