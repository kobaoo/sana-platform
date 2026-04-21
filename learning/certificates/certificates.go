package certificates

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/certificate"
	"encore.app/db/ent/employee"
	"encore.dev/beta/auth"
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

// List returns certificates with optional filtering.
//
//encore:api auth method=GET path=/certificates
func List(ctx context.Context, params *ListParams) (*ListResponse, error) {
	query := Client.Certificate.Query().Where(certificate.IsActiveEQ(true))

	if ad, ok := auth.Data().(*authhandler.AuthData); ok && ad.Role == authhandler.RoleHR {
		if ad.DzoID == "" {
			return nil, errs.B().Code(errs.PermissionDenied).Msg("HR user has no DZO assigned").Err()
		}
		dzoUID, err := uuid.Parse(ad.DzoID)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("invalid dzo_id in token").Err()
		}
		empIDs, err := Client.Employee.Query().
			Where(employee.DzoIDEQ(dzoUID), employee.IsDeletedEQ(false)).
			IDs(ctx)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scope certificates by DZO").Cause(err).Err()
		}
		query = query.Where(certificate.EmployeeIDIn(empIDs...))
	}

	if params.EmployeeID != "" {
		uid, err := uuid.Parse(params.EmployeeID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid employee_id format").Err()
		}
		query = query.Where(certificate.EmployeeIDEQ(uid))
	}
	if params.EntityType != "" {
		query = query.Where(certificate.EntityTypeEQ(certificate.EntityType(params.EntityType)))
	}

	rows, err := query.All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query certificates").Cause(err).Err()
	}

	certs := make([]Certificate, 0, len(rows))
	for _, r := range rows {
		certs = append(certs, *entToCert(r))
	}
	return &ListResponse{Certificates: certs, Total: len(certs)}, nil
}

// Create creates a new certificate.
//
//encore:api auth method=POST path=/certificates
func Create(ctx context.Context, req *CreateRequest) (*GetCertResponse, error) {
	if req.Title == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}
	if req.IssuedDate.IsZero() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("issued_date is required").Err()
	}
	validTypes := map[string]bool{"EXTERNAL": true, "SCORM": true, "INTERNAL": true}
	if !validTypes[req.Type] {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("type must be EXTERNAL, SCORM, or INTERNAL").Err()
	}

	if ad, ok := auth.Data().(*authhandler.AuthData); ok && ad.Role == authhandler.RoleHR {
		if ad.DzoID == "" {
			return nil, errs.B().Code(errs.PermissionDenied).Msg("HR user has no DZO assigned").Err()
		}
		dzoUID, err := uuid.Parse(ad.DzoID)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("invalid dzo_id in token").Err()
		}
		exists, err := Client.Employee.Query().
			Where(employee.IDEQ(req.EmployeeID), employee.IsDeletedEQ(false), employee.DzoIDEQ(dzoUID)).
			Exist(ctx)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to validate employee DZO").Cause(err).Err()
		}
		if !exists {
			return nil, errs.B().Code(errs.PermissionDenied).Msg("employee is outside your DZO").Err()
		}
	}

	cert, err := insertCert(ctx, req)
	if err != nil {
		return nil, err
	}
	return &GetCertResponse{Certificate: *cert}, nil
}

// Update updates certificate fields.
//
//encore:api auth method=PUT path=/certificates/:id
func Update(ctx context.Context, id string, req *UpdateRequest) (*GetCertResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	exists, err := Client.Certificate.Query().
		Where(certificate.IDEQ(uid), certificate.IsActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to check certificate").Cause(err).Err()
	}
	if !exists {
		return nil, errs.B().Code(errs.NotFound).Msg("certificate not found").Err()
	}

	row, err := Client.Certificate.UpdateOneID(uid).
		SetTitle(req.Title).
		SetType(certificate.Type(req.Type)).
		SetIssuedDate(req.IssuedDate).
		SetNillableExpiryDate(req.ExpiryDate).
		SetEntityType(certificate.EntityType(req.EntityType)).
		SetEntityID(req.EntityID).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update certificate").Cause(err).Err()
	}
	return &GetCertResponse{Certificate: *entToCert(row)}, nil
}

// GetByID returns a single certificate by ID.
//
//encore:api auth method=GET path=/certificates/:id
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

// UploadFile uploads a PDF file for a certificate.
//
//encore:api auth raw method=POST path=/certificates/:id/upload
func UploadFile(w http.ResponseWriter, r *http.Request) {
	handleUpload(w, r)
}

// DownloadFile downloads the PDF file for a certificate.
//
//encore:api auth raw method=GET path=/certificates/:id/download
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	handleDownload(w, r)
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

func handleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Encore raw endpoints may not set PathValue; find the UUID segment in the path.
	var idStr string
	for _, seg := range strings.Split(r.URL.Path, "/") {
		if _, parseErr := uuid.Parse(seg); parseErr == nil {
			idStr = seg
			break
		}
	}
	if idStr == "" {
		http.Error(w, "path parameter 'id' is required", http.StatusBadRequest)
		return
	}

	uid, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	exists, err := Client.Certificate.Query().
		Where(certificate.IDEQ(uid), certificate.IsActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "certificate not found", http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		http.Error(w, "only PDF files are allowed", http.StatusUnsupportedMediaType)
		return
	}

	// Verify PDF magic bytes (%PDF)
	magic := make([]byte, 4)
	if n, _ := io.ReadFull(file, magic); n < 4 || string(magic) != "%PDF" {
		http.Error(w, "only PDF files are allowed", http.StatusUnsupportedMediaType)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "failed to process file", http.StatusInternalServerError)
		return
	}

	// TODO: заменить на S3/object storage перед проде
	dir := "/tmp/certificates"
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, "failed to create directory", http.StatusInternalServerError)
		return
	}
	filePath := filepath.Join(dir, uid.String()+".pdf")

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	updated, err := Client.Certificate.UpdateOneID(uid).SetFileURL(filePath).Save(ctx)
	if err != nil {
		http.Error(w, "database update error", http.StatusInternalServerError)
		return
	}

	cert := entToCert(updated)
	fileURL := ""
	if cert.FileURL != nil {
		fileURL = *cert.FileURL
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id":%q,"title":%q,"file_url":%q,"is_active":%v}`,
		cert.ID, cert.Title, fileURL, cert.IsActive)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var idStr string
	for _, seg := range strings.Split(r.URL.Path, "/") {
		if _, parseErr := uuid.Parse(seg); parseErr == nil {
			idStr = seg
			break
		}
	}
	if idStr == "" {
		http.Error(w, "path parameter 'id' is required", http.StatusBadRequest)
		return
	}

	uid, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	row, err := Client.Certificate.Query().
		Where(certificate.IDEQ(uid), certificate.IsActiveEQ(true)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "certificate not found", http.StatusNotFound)
			return
		}
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if row.FileURL == nil || *row.FileURL == "" {
		http.Error(w, "no file attached to this certificate", http.StatusNotFound)
		return
	}

	f, err := os.Open(*row.FileURL)
	if err != nil {
		http.Error(w, "file not found on server", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, row.Title))
	if _, err := io.Copy(w, f); err != nil {
		// headers already sent; nothing useful to do
		return
	}
}

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
