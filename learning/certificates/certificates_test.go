package certificates

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/et"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
)

func ctx() context.Context {
	return context.Background()
}

func withADMAuth(t *testing.T) {
	t.Helper()
	et.OverrideAuthInfo(auth.UID("test-adm"), &authhandler.AuthData{
		KeycloakUserID: "test-adm",
		Role:           authhandler.RoleADM,
	})
}

func makeCert(t *testing.T, title string) *Certificate {
	t.Helper()
	withADMAuth(t)
	resp, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      title,
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("makeCert: %v", err)
	}
	return &resp.Certificate
}

// buildPDFMultipart собирает multipart-тело с валидным PDF-заголовком.
func buildPDFMultipart(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.pdf"`)
	h.Set("Content-Type", "application/pdf")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("%PDF-1.4 test content"))
	writer.Close()
	return body, writer.FormDataContentType()
}

// ════ CREATE ════

func TestCreate_Success(t *testing.T) {
	withADMAuth(t)
	resp, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      "Go Advanced",
		IssuedDate: time.Now(),
		EntityType: "SCORM_COURSE",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Certificate.Title != "Go Advanced" {
		t.Errorf("expected title 'Go Advanced', got %q", resp.Certificate.Title)
	}
}

func TestCreate_EmptyTitle(t *testing.T) {
	withADMAuth(t)
	_, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      "",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

// ════ GET BY ID ════

func TestGetByID_Success(t *testing.T) {
	cert := makeCert(t, "Find Me")
	resp, err := GetByID(ctx(), cert.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Certificate.ID != cert.ID {
		t.Errorf("expected ID %s, got %s", cert.ID, resp.Certificate.ID)
	}
}

func TestGetByID_InvalidID(t *testing.T) {
	_, err := GetByID(ctx(), "not-a-uuid")
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	_, err := GetByID(ctx(), uuid.New().String())
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

// ════ LIST ════

func TestList_FilterByEmployeeID(t *testing.T) {
	withADMAuth(t)
	targetEmployee := uuid.New()

	resp1, err := Create(ctx(), &CreateRequest{
		EmployeeID: targetEmployee,
		Type:       "EXTERNAL",
		Title:      "Target Cert",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	_, err = Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      "Other Cert",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("create other: %v", err)
	}

	list, err := List(ctx(), &ListParams{EmployeeID: targetEmployee.String()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	for _, c := range list.Certificates {
		if c.EmployeeID != resp1.Certificate.EmployeeID {
			t.Errorf("got cert with wrong employee_id: %s", c.EmployeeID)
		}
	}

	found := false
	for _, c := range list.Certificates {
		if c.ID == resp1.Certificate.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("target certificate not found in filtered list")
	}
}

func TestList_FilterByEntityType(t *testing.T) {
	withADMAuth(t)
	_, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      "Scorm Cert",
		IssuedDate: time.Now(),
		EntityType: "SCORM_COURSE",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := List(ctx(), &ListParams{EntityType: "SCORM_COURSE"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	for _, c := range list.Certificates {
		if c.EntityType != "SCORM_COURSE" {
			t.Errorf("expected SCORM_COURSE, got %s", c.EntityType)
		}
	}
}

func TestList_InvalidEmployeeID(t *testing.T) {
	withADMAuth(t)
	_, err := List(ctx(), &ListParams{EmployeeID: "bad-uuid"})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestList_EMPRoleForbidden(t *testing.T) {
	et.OverrideAuthInfo(auth.UID("emp-user"), &authhandler.AuthData{
		KeycloakUserID: "emp-user",
		Role:           authhandler.RoleEMP,
	})
	_, err := List(ctx(), &ListParams{})
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied for EMP role, got %v", err)
	}
}

// ════ UPDATE ════

func TestUpdate_Success(t *testing.T) {
	cert := makeCert(t, "Original Title")
	withADMAuth(t)
	resp, err := Update(ctx(), cert.ID, &UpdateRequest{
		Type:       "EXTERNAL",
		Title:      "Updated Title",
		IssuedDate: time.Now(),
		EntityType: "SCORM_COURSE",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resp.Certificate.Title != "Updated Title" {
		t.Errorf("expected 'Updated Title', got %q", resp.Certificate.Title)
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	withADMAuth(t)
	_, err := Update(ctx(), "not-a-uuid", &UpdateRequest{
		Type:       "EXTERNAL",
		Title:      "Doesn't matter",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

// ════ DELETE ════

func TestDelete_SuccessSoftDelete(t *testing.T) {
	cert := makeCert(t, "To Delete")
	withADMAuth(t)
	_, err := Delete(ctx(), cert.ID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = GetByID(ctx(), cert.ID)
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound after delete, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	withADMAuth(t)
	_, err := Delete(ctx(), uuid.New().String())
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestDelete_InvalidID(t *testing.T) {
	withADMAuth(t)
	_, err := Delete(ctx(), "not-a-uuid")
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

// ════ UPLOAD ════

// TestUploadFile_ValidPDF проверяет что:
// 1. PDF с правильным magic bytes проходит валидацию
// 2. file_url обновляется в БД (ключ объекта, не локальный путь)
// Encore Object Storage поднимает локальный in-memory бэкенд автоматически — внешний сторадж не нужен.
func TestUploadFile_ValidPDF(t *testing.T) {
	cert := makeCert(t, "Upload Test")
	withADMAuth(t)

	body, contentType := buildPDFMultipart(t)

	req := httptest.NewRequest(http.MethodPost, "/certificates/"+cert.ID+"/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.SetPathValue("id", cert.ID)

	rr := httptest.NewRecorder()
	handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, err := GetByID(ctx(), cert.ID)
	if err != nil {
		t.Fatalf("GetByID after upload: %v", err)
	}
	if updated.Certificate.FileURL == nil || *updated.Certificate.FileURL == "" {
		t.Error("expected file_url to be set after upload")
	}
	// file_url должен быть ключом объекта, не локальным путём
	if updated.Certificate.FileURL != nil {
		key := *updated.Certificate.FileURL
		expectedKey := cert.ID + ".pdf"
		if key != expectedKey {
			t.Errorf("expected object key %q, got %q", expectedKey, key)
		}
	}
}

func TestUploadFile_NotPDF(t *testing.T) {
	cert := makeCert(t, "Upload Wrong Type")
	withADMAuth(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.png"`)
	h.Set("Content-Type", "image/png")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("fake image content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/certificates/"+cert.ID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", cert.ID)

	rr := httptest.NewRecorder()
	handleUpload(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", rr.Code)
	}
}

func TestUploadFile_FakePDFMagicBytes(t *testing.T) {
	cert := makeCert(t, "Fake PDF Magic")
	withADMAuth(t)

	// Файл с расширением .pdf, но без %PDF в начале
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="evil.pdf"`)
	h.Set("Content-Type", "application/pdf")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("this is not a pdf at all"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/certificates/"+cert.ID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", cert.ID)

	rr := httptest.NewRecorder()
	handleUpload(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 for fake PDF magic bytes, got %d", rr.Code)
	}
}

func TestUploadFile_CertNotFound(t *testing.T) {
	withADMAuth(t)

	body, contentType := buildPDFMultipart(t)
	fakeID := uuid.New().String()

	req := httptest.NewRequest(http.MethodPost, "/certificates/"+fakeID+"/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.SetPathValue("id", fakeID)

	rr := httptest.NewRecorder()
	handleUpload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUploadFile_Forbidden_NoAuth(t *testing.T) {
	cert := makeCert(t, "Forbidden Upload")

	// Сброс auth — нет данных
	et.OverrideAuthInfo(auth.UID(""), nil)

	body, contentType := buildPDFMultipart(t)
	req := httptest.NewRequest(http.MethodPost, "/certificates/"+cert.ID+"/upload", body)
	req.Header.Set("Content-Type", contentType)

	rr := httptest.NewRecorder()
	handleUpload(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 without auth, got %d", rr.Code)
	}
}

// ════ DOWNLOAD ════

func TestDownloadFile_NoFile(t *testing.T) {
	// Сертификат без file_url
	cert := makeCert(t, "No File Cert")
	withADMAuth(t)

	req := httptest.NewRequest(http.MethodGet, "/certificates/"+cert.ID+"/download", nil)
	rr := httptest.NewRecorder()
	handleDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when no file attached, got %d", rr.Code)
	}
}

func TestDownloadFile_CertNotFound(t *testing.T) {
	withADMAuth(t)
	fakeID := uuid.New().String()

	req := httptest.NewRequest(http.MethodGet, "/certificates/"+fakeID+"/download", nil)
	rr := httptest.NewRecorder()
	handleDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDownloadFile_Forbidden_NoAuth(t *testing.T) {
	cert := makeCert(t, "Forbidden Download")
	et.OverrideAuthInfo(auth.UID(""), nil)

	req := httptest.NewRequest(http.MethodGet, "/certificates/"+cert.ID+"/download", nil)
	rr := httptest.NewRecorder()
	handleDownload(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
