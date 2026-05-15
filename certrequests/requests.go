package certrequests

import (
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"encore.app/db/ent"
)

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

func toResponse(r *ent.Request) *RequestResponse {
	return &RequestResponse{
		ID:          r.ID,
		InitiatorID: r.InitiatorID,
		EntityID:    r.EntityID,
		EntityType:  r.EntityType,
		Step:        r.Step,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
}
