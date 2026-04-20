package certificates

import (
	"context"
	"time"

	"encore.app/db/ent"
	"encore.app/db/ent/certificate"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
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

// ════ ENDPOINTS ════

// Create creates a new certificate.
//
//encore:api auth method=POST path=/certificates
func Create(ctx context.Context, req *CreateRequest) (*GetCertResponse, error) {
	if req.Title == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}
	cert, err := insertCert(ctx, req)
	if err != nil {
		return nil, err
	}
	return &GetCertResponse{Certificate: *cert}, nil
}

// GetExpiring returns certificates expiring within the next 6 months.
//
//encore:api auth method=GET path=/certificates/expiring
func GetExpiring(ctx context.Context) (*ListResponse, error) {
	rows, err := queryExpiringCerts(ctx)
	if err != nil {
		return nil, err
	}
	return &ListResponse{Certificates: rows, Total: len(rows)}, nil
}

// GetByID returns a single certificate by ID.
//
//encore:api auth method=GET path=/certificates/i/:id
func GetByID(ctx context.Context, id string) (*GetCertResponse, error) {
	cert, err := queryCertByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &GetCertResponse{Certificate: *cert}, nil
}

// Delete soft-deletes a certificate.
//
//encore:api auth method=DELETE path=/certificates/:id
func Delete(ctx context.Context, id string) (*DeleteResponse, error) {
	if err := softDeleteCert(ctx, id); err != nil {
		return nil, err
	}
	return &DeleteResponse{Message: "certificate deleted successfully"}, nil
}

// ════ INTERNAL ════

func insertCert(ctx context.Context, req *CreateRequest) (*Certificate, error) {
	row, err := Client.Certificate.Create().
		SetEmployeeID(req.EmployeeID).
		SetType(certificate.Type(req.Type)).
		SetTitle(req.Title).
		SetNillableFileURL(req.FileURL).
		SetIssuedDate(req.IssuedDate).
		SetNillableExpiryDate(req.ExpiryDate).
		SetNillableUploadedBy(req.UploadedBy).
		SetEntityType(certificate.EntityType(req.EntityType)).
		SetEntityID(req.EntityID).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create certificate").Cause(err).Err()
	}
	return entToCert(row), nil
}

func queryCertByID(ctx context.Context, id string) (*Certificate, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row, err := Client.Certificate.Query().
		Where(certificate.IDEQ(uid), certificate.IsActiveEQ(true)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("certificate not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Err()
	}
	return entToCert(row), nil
}

func queryExpiringCerts(ctx context.Context) ([]Certificate, error) {
	threshold := time.Now().AddDate(0, 6, 0)
	rows, err := Client.Certificate.Query().
		Where(
			certificate.ExpiryDateLTE(threshold),
			certificate.IsActiveEQ(true),
		).All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}

	certs := make([]Certificate, 0, len(rows))
	for _, r := range rows {
		certs = append(certs, *entToCert(r))
	}
	return certs, nil
}

func softDeleteCert(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	exists, err := Client.Certificate.Query().
		Where(certificate.IDEQ(uid), certificate.IsActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete certificate").Cause(err).Err()
	}
	if !exists {
		return errs.B().Code(errs.NotFound).Msg("certificate not found").Err()
	}

	return Client.Certificate.UpdateOneID(uid).SetIsActive(false).Exec(ctx)
}

// ════ HELPERS ════

func entToCert(e *ent.Certificate) *Certificate {
	var uploadedBy *string
	if e.UploadedBy != nil {
		s := e.UploadedBy.String()
		uploadedBy = &s
	}

	return &Certificate{
		ID:         e.ID.String(),
		EmployeeID: e.EmployeeID.String(),
		Type:       string(e.Type),
		Title:      e.Title,
		FileURL:    e.FileURL,
		IssuedDate: e.IssuedDate,
		ExpiryDate: e.ExpiryDate,
		UploadedBy: uploadedBy,
		EntityType: string(e.EntityType),
		EntityID:   e.EntityID.String(),
		IsActive:   e.IsActive,
	}
}
