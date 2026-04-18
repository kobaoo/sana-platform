package certificates

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"encore.app/db/ent"
	"encore.app/db/ent/certificate"
)

// ════ DATABASE ════
var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

// ════ MODELS ════
type Certificate struct {
	ID             string     `json:"id"`
	EmployeeID     int64      `json:"employee_id"`
	DzoID          int64      `json:"dzo_id"`
	Title          string     `json:"title"`
	FileURL        string     `json:"file_url"`
	IssueDate      time.Time  `json:"issue_date"`
	ExpirationDate *time.Time `json:"expiration_date"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	EmployeeID     int64      `json:"employee_id"`
	DzoID          int64      `json:"dzo_id"`
	Title          string     `json:"title"`
	FileURL        string     `json:"file_url"`
	IssueDate      time.Time  `json:"issue_date"`
	ExpirationDate *time.Time `json:"expiration_date"`
}

type ExpiringParams struct {
	Days  int   `query:"days"`   // по умолчанию будет 180
	DzoID int64 `query:"dzo_id"` // опциональный фильтр
}

type ListResponse struct {
	Certificates []Certificate `json:"certificates"`
	Total        int           `json:"total"`
}

// ════ ENDPOINTS ════

// Create — POST /certificates
//
//encore:api method=POST path=/certificates
func Create(ctx context.Context, req *CreateRequest) (*Certificate, error) {
	row, err := Client.Certificate.Create().
		SetEmployeeID(req.EmployeeID).
		SetDzoID(req.DzoID).
		SetTitle(req.Title).
		SetFileURL(req.FileURL).
		SetIssueDate(req.IssueDate).
		SetNillableExpirationDate(req.ExpirationDate).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create certificate").Cause(err).Err()
	}
	return entToCert(row), nil
}

// GetByID — GET /certificates/:id
//
//encore:api method=GET path=/certificates/:id
func GetByID(ctx context.Context, id string) (*Certificate, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid uuid").Err()
	}
	row, err := Client.Certificate.Query().Where(certificate.IDEQ(uid)).Only(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.NotFound).Msg("certificate not found").Err()
	}
	return entToCert(row), nil
}

// ListByEmployee — GET /employees/:employeeID/certificates
//
//encore:api method=GET path=/employees/:employeeID/certificates
func ListByEmployee(ctx context.Context, employeeID int64) (*ListResponse, error) {
	rows, err := Client.Certificate.Query().
		Where(certificate.EmployeeIDEQ(employeeID), certificate.IsActive(true)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}
	return &ListResponse{Certificates: entToCerts(rows), Total: len(rows)}, nil
}

// GetExpiring — GET /certificates/expiring
//
//encore:api method=GET path=/certificates-list/expiring
func GetExpiring(ctx context.Context, params *ExpiringParams) (*ListResponse, error) {
	days := params.Days
	if days == 0 {
		days = 180
	}
	threshold := time.Now().AddDate(0, 0, days)

	query := Client.Certificate.Query().
		Where(
			certificate.ExpirationDateLTE(threshold),
			certificate.IsActive(true),
		)

	if params.DzoID != 0 {
		query.Where(certificate.DzoIDEQ(params.DzoID))
	}

	rows, err := query.All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}
	return &ListResponse{Certificates: entToCerts(rows), Total: len(rows)}, nil
}

// Update — PUT /certificates/:id
//
//encore:api method=PUT path=/certificates/:id
func Update(ctx context.Context, id string, req *CreateRequest) (*Certificate, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid uuid").Err()
	}
	row, err := Client.Certificate.UpdateOneID(uid).
		SetEmployeeID(req.EmployeeID).
		SetDzoID(req.DzoID).
		SetTitle(req.Title).
		SetFileURL(req.FileURL).
		SetIssueDate(req.IssueDate).
		SetNillableExpirationDate(req.ExpirationDate).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}
	return entToCert(row), nil
}

// Delete — DELETE /certificates/:id (Мягкое удаление)
//
//encore:api method=DELETE path=/certificates/:id
func Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid uuid").Err()
	}
	return Client.Certificate.UpdateOneID(uid).SetIsActive(false).Exec(ctx)
}

// ════ HELPERS ════
func entToCert(e *ent.Certificate) *Certificate {
	return &Certificate{
		ID:             e.ID.String(),
		EmployeeID:     e.EmployeeID,
		DzoID:          e.DzoID,
		Title:          e.Title,
		FileURL:        e.FileURL,
		IssueDate:      e.IssueDate,
		ExpirationDate: e.ExpirationDate,
		IsActive:       e.IsActive,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func entToCerts(rows []*ent.Certificate) []Certificate {
	certs := make([]Certificate, 0, len(rows))
	for _, r := range rows {
		certs = append(certs, *entToCert(r))
	}
	return certs
}
