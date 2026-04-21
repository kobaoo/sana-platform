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

	"encore.dev/beta/errs"
	"github.com/google/uuid"
)

func ctx() context.Context {
	return context.Background()
}

func makeCert(t *testing.T, title string) *Certificate {
	t.Helper()
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

func TestCreate_Success(t *testing.T) {
	resp, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "SCORM",
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

func TestDelete_SuccessSoftDelete(t *testing.T) {
	cert := makeCert(t, "To Delete")
	_, err := Delete(ctx(), cert.ID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = GetByID(ctx(), cert.ID)
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound after delete, got %v", err)
	}
}

func TestUploadFile_ValidPDF(t *testing.T) {
	cert := makeCert(t, "Upload Test")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.pdf"`)
	h.Set("Content-Type", "application/pdf")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("%PDF-1.4 test content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/certificates/"+cert.ID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
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
}
