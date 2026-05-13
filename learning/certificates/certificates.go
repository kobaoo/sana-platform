package certificates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/certificate"
	"encore.app/db/ent/employee"
	entuser "encore.app/db/ent/user"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/objects"
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

// ════ SECRETS ════

var secrets struct {
	MailServer   string
	MailPort     string
	MailUsername string
	MailPassword string
	MailFrom     string
	AppURL       string
}

// ════ OBJECT STORAGE ════

var CertificatesBucket = objects.NewBucket("certificates", objects.BucketConfig{})

const maxUploadSize = 10 << 20 // 10 MiB

func objectKey(id uuid.UUID) string {
	return id.String() + ".pdf"
}

// ════ ENDPOINTS ════

// List returns certificates with optional filtering. EMP role is not allowed — use /my/certificates.
//
//encore:api auth method=GET path=/certificates
func List(ctx context.Context, params *ListParams) (*ListResponse, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Err()
	}
	if ad.Role == authhandler.RoleEMP {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("используйте /my/certificates для просмотра своих сертификатов").Err()
	}

	query := Client.Certificate.Query().Where(certificate.IsActiveEQ(true))

	if ad.Role == authhandler.RoleHR {
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

// MyCertificates returns certificates belonging to the current user's employee record.
//
//encore:api auth method=GET path=/my/certificates
func MyCertificates(ctx context.Context) (*ListResponse, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Err()
	}

	empID, err := resolveEmployeeIDForUser(ctx, ad.KeycloakUserID)
	if err != nil {
		return &ListResponse{Certificates: []Certificate{}, Total: 0}, nil
	}

	rows, err := Client.Certificate.Query().
		Where(certificate.EmployeeIDEQ(empID), certificate.IsActiveEQ(true)).
		Order(ent.Desc(certificate.FieldIssuedDate)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query certificates").Cause(err).Err()
	}

	certs := make([]Certificate, 0, len(rows))
	for _, r := range rows {
		certs = append(certs, *entToCert(r))
	}
	return &ListResponse{Certificates: certs, Total: len(certs)}, nil
}

// Create creates a new certificate. Requires SA or ADM role.
//
//encore:api auth method=POST path=/certificates
func Create(ctx context.Context, req *CreateRequest) (*GetCertResponse, error) {
	if err := requireSAorADM(); err != nil {
		return nil, err
	}

	if req.Title == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}
	if req.IssuedDate.IsZero() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("issued_date is required").Err()
	}
	if req.Type != "EXTERNAL" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("type must be EXTERNAL").Err()
	}

	cert, err := insertCert(ctx, req)
	if err != nil {
		return nil, err
	}
	return &GetCertResponse{Certificate: *cert}, nil
}

// Update updates certificate fields. Requires SA or ADM role.
//
//encore:api auth method=PUT path=/certificates/:id
func Update(ctx context.Context, id string, req *UpdateRequest) (*GetCertResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	if err := requireSAorADM(); err != nil {
		return nil, err
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

// MyHRContact returns the contact details of active HR users in the caller's DZO.
//
//encore:api auth method=GET path=/my/hr-contact
func MyHRContact(ctx context.Context) (*HRContactResponse, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Err()
	}
	if ad.DzoID == "" {
		return &HRContactResponse{Contacts: []HRContact{}}, nil
	}
	dzoUID, err := uuid.Parse(ad.DzoID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("invalid dzo_id in token").Err()
	}

	hrUsers, err := Client.User.Query().
		Where(entuser.RoleEQ("HR"), entuser.DzoIDEQ(dzoUID), entuser.IsActiveEQ(true)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query HR contacts").Cause(err).Err()
	}

	hrUserIDs := make([]uuid.UUID, 0, len(hrUsers))
	for _, u := range hrUsers {
		hrUserIDs = append(hrUserIDs, u.ID)
	}

	empByUserID := make(map[uuid.UUID]*ent.Employee)
	if len(hrUserIDs) > 0 {
		empRows, empErr := Client.Employee.Query().
			Where(employee.UserIDIn(hrUserIDs...), employee.IsDeletedEQ(false)).
			All(ctx)
		if empErr == nil {
			for _, e := range empRows {
				if e.UserID != nil {
					empByUserID[*e.UserID] = e
				}
			}
		}
	}

	contacts := make([]HRContact, 0, len(hrUsers))
	for _, u := range hrUsers {
		c := HRContact{Email: u.Email}
		if emp, found := empByUserID[u.ID]; found {
			c.Name = emp.FullName
			c.Phone = emp.InternalPhone
		}
		contacts = append(contacts, c)
	}

	return &HRContactResponse{Contacts: contacts}, nil
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

// ListExpiring returns certificates expiring within the given number of days for a DZO.
//
//encore:api auth method=GET path=/expiring-certificates
func ListExpiring(ctx context.Context, params *ExpiringParams) (*ListResponse, error) {
	if params.DzoID == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id is required").Err()
	}

	days := params.Days
	if days <= 0 {
		days = 180
	}

	if ad, ok := auth.Data().(*authhandler.AuthData); ok && ad.Role == authhandler.RoleHR {
		if ad.DzoID != params.DzoID {
			return nil, errs.B().Code(errs.PermissionDenied).Msg("you can only view your own DZO").Err()
		}
	}

	dzoUID, err := uuid.Parse(params.DzoID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}

	now := time.Now()
	threshold := now.AddDate(0, 0, days)

	empIDs, err := Client.Employee.Query().
		Where(employee.DzoIDEQ(dzoUID), employee.IsDeletedEQ(false)).
		IDs(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query employees").Cause(err).Err()
	}

	query := Client.Certificate.Query().
		Where(
			certificate.IsActiveEQ(true),
			certificate.ExpiryDateNotNil(),
			certificate.ExpiryDateGT(now),
			certificate.ExpiryDateLTE(threshold),
		)

	if len(empIDs) > 0 {
		query = query.Where(certificate.EmployeeIDIn(empIDs...))
	} else {
		return &ListResponse{Certificates: []Certificate{}, Total: 0}, nil
	}

	rows, err := query.Order(ent.Asc(certificate.FieldExpiryDate)).All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query expiring certificates").Cause(err).Err()
	}

	certs := make([]Certificate, 0, len(rows))
	for _, r := range rows {
		certs = append(certs, *entToCert(r))
	}
	return &ListResponse{Certificates: certs, Total: len(certs)}, nil
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
	now := time.Now()
	threshold := now.AddDate(0, 6, 0)
	rows, err := Client.Certificate.Query().
		Where(
			certificate.IsActiveEQ(true),
			certificate.ExpiryDateNotNil(),
			certificate.ExpiryDateGT(now),
			certificate.ExpiryDateLTE(threshold),
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

func queryEmployeeDzoMap(ctx context.Context, certs []Certificate) (map[string]string, error) {
	seen := make(map[uuid.UUID]struct{}, len(certs))
	for _, c := range certs {
		uid, err := uuid.Parse(c.EmployeeID)
		if err == nil {
			seen[uid] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return map[string]string{}, nil
	}

	ids := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}

	rows, err := Client.Employee.Query().
		Where(employee.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query employee DZO map").Cause(err).Err()
	}

	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.ID.String()] = r.DzoID.String()
	}
	return m, nil
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

func requireSAorADM() error {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok || (ad.Role != authhandler.RoleSA && ad.Role != authhandler.RoleADM) {
		return errs.B().Code(errs.PermissionDenied).Msg("Нет доступа").Err()
	}
	return nil
}

func checkHRCertScope(ctx context.Context, ad *authhandler.AuthData, empID uuid.UUID) error {
	if ad.DzoID == "" {
		return errs.B().Code(errs.PermissionDenied).Msg("HR user has no DZO assigned").Err()
	}
	dzoUID, err := uuid.Parse(ad.DzoID)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("invalid dzo_id in token").Err()
	}
	ok, err := Client.Employee.Query().
		Where(employee.IDEQ(empID), employee.DzoIDEQ(dzoUID), employee.IsDeletedEQ(false)).
		Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to validate employee DZO").Cause(err).Err()
	}
	if !ok {
		return errs.B().Code(errs.PermissionDenied).Msg("employee is outside your DZO").Err()
	}
	return nil
}

func resolveEmployeeIDForUser(ctx context.Context, keycloakUserID string) (uuid.UUID, error) {
	userRow, err := Client.User.Query().
		Where(entuser.KeycloakUserIDEQ(keycloakUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, errs.B().Code(errs.PermissionDenied).Msg("user not found").Err()
		}
		return uuid.UUID{}, errs.B().Code(errs.Internal).Err()
	}
	empRow, err := Client.Employee.Query().
		Where(employee.UserIDEQ(userRow.ID), employee.IsDeletedEQ(false)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, errs.B().Code(errs.PermissionDenied).Msg("no employee linked to your account").Err()
		}
		return uuid.UUID{}, errs.B().Code(errs.Internal).Err()
	}
	return empRow.ID, nil
}

// uploadCertificate stores a PDF in the Encore-managed bucket and returns the object key.
// Bucket-level size limit defends against callers that bypass the HTTP-level MaxBytesReader.
func uploadCertificate(ctx context.Context, id uuid.UUID, file io.Reader) (string, error) {
	key := objectKey(id)
	w := CertificatesBucket.Upload(ctx, key, objects.WithUploadAttrs(objects.UploadAttrs{
		ContentType: "application/pdf",
	}))

	limited := io.LimitReader(file, maxUploadSize+1)
	n, err := io.Copy(w, limited)
	if err != nil {
		w.Abort(err)
		return "", fmt.Errorf("upload write: %w", err)
	}
	if n > maxUploadSize {
		w.Abort(fmt.Errorf("file exceeds %d bytes", maxUploadSize))
		return "", fmt.Errorf("file exceeds bucket limit of %d bytes", maxUploadSize)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("upload close: %w", err)
	}
	return key, nil
}

// signedDownloadURL returns a 15-minute presigned URL for downloading a certificate PDF.
func signedDownloadURL(ctx context.Context, key string) (string, error) {
	signed, err := CertificatesBucket.SignedDownloadURL(ctx, key, objects.WithTTL(15*time.Minute))
	if err != nil {
		return "", fmt.Errorf("signed url: %w", err)
	}
	return signed.URL, nil
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok || (ad.Role != authhandler.RoleSA && ad.Role != authhandler.RoleADM) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

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

	certRow, err := Client.Certificate.Query().
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

	if ad.Role == authhandler.RoleHR {
		if scopeErr := checkHRCertScope(ctx, ad, certRow.EmployeeID); scopeErr != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "file too large: maximum size is 10 MiB", http.StatusRequestEntityTooLarge)
			return
		}
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
		http.Error(w, "разрешены только PDF-файлы", http.StatusUnsupportedMediaType)
		return
	}

	magic := make([]byte, 4)
	if n, _ := io.ReadFull(file, magic); n < 4 || string(magic) != "%PDF" {
		http.Error(w, "разрешены только PDF-файлы", http.StatusUnsupportedMediaType)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "failed to process file", http.StatusInternalServerError)
		return
	}

	key, err := uploadCertificate(ctx, uid, file)
	if err != nil {
		http.Error(w, "failed to upload file to storage", http.StatusInternalServerError)
		return
	}

	updated, err := Client.Certificate.UpdateOneID(uid).SetFileURL(key).Save(ctx)
	if err != nil {
		http.Error(w, "database update error", http.StatusInternalServerError)
		return
	}

	cert := entToCert(updated)
	fileKey := ""
	if cert.FileURL != nil {
		fileKey = *cert.FileURL
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id":%q,"title":%q,"file_url":%q,"is_active":%v}`,
		cert.ID, cert.Title, fileKey, cert.IsActive)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

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

	switch ad.Role {
	case authhandler.RoleSA, authhandler.RoleADM:
		// full access
	case authhandler.RoleHR:
		if scopeErr := checkHRCertScope(ctx, ad, row.EmployeeID); scopeErr != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	case authhandler.RoleEMP:
		empID, resolveErr := resolveEmployeeIDForUser(ctx, ad.KeycloakUserID)
		if resolveErr != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if row.EmployeeID != empID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	default:
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if row.FileURL == nil || *row.FileURL == "" {
		http.Error(w, "no file attached to this certificate", http.StatusNotFound)
		return
	}

	presignURL, err := signedDownloadURL(ctx, *row.FileURL)
	if err != nil {
		http.Error(w, "failed to generate download link", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, presignURL, http.StatusTemporaryRedirect)
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
