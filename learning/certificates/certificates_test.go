package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func ctx() context.Context {
	return context.Background()
}

func TestCreateCertificate_Success(t *testing.T) {
	empID := uuid.New()
	entID := uuid.New()

	resp, err := Create(ctx(), &CreateRequest{
		EmployeeID: empID,
		Type:       "EXTERNAL",
		Title:      "Go Developer Certificate",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   entID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Certificate.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Certificate.Title != "Go Developer Certificate" {
		t.Errorf("expected title 'Go Developer Certificate', got %q", resp.Certificate.Title)
	}
}
