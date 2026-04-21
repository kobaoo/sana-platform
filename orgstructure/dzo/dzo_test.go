package dzo

import (
	"context"
	"testing"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

func ctx() context.Context {
	return auth.WithContext(
		context.Background(),
		auth.UID("test-user"),
		&authhandler.AuthData{
			Role:      authhandler.RoleADM,
			CompanyID: "00000000-0000-0000-0000-000000000001",
		},
	)
}
func makeDZO(t *testing.T, name string) *DZO {
	t.Helper()

	resp, err := CreateDZO(ctx(), &CreateDZORequest{
		ClientID: "11111111-1111-1111-1111-111111111111",
		Name:     name,
	})
	if err != nil {
		t.Fatalf("makeDZO: %v", err)
	}

	return &resp.DZO
}

// ════ CREATE ════

func TestCreateDZO_Success(t *testing.T) {
	resp, err := CreateDZO(ctx(), &CreateDZORequest{
		ClientID: "11111111-1111-1111-1111-111111111111",
		Name:     "Test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.DZO.ID == "" {
		t.Error("expected ID")
	}
}

func TestCreateDZO_EmptyName(t *testing.T) {
	_, err := CreateDZO(ctx(), &CreateDZORequest{
		ClientID: "11111111-1111-1111-1111-111111111111",
		Name:     "",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument")
	}
}

// ════ GET ════

func TestGetDZO_Success(t *testing.T) {
	dzo := makeDZO(t, "GET")

	resp, err := GetDZO(ctx(), dzo.ID)
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if resp.DZO.ID != dzo.ID {
		t.Error("wrong id")
	}
}

// ════ DELETE ════

func TestDeleteDZO_Success(t *testing.T) {
	dzo := makeDZO(t, "DEL")

	_, err := DeleteDZO(ctx(), dzo.ID)
	if err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestDeleteDZO_NotFound(t *testing.T) {
	_, err := DeleteDZO(ctx(), "00000000-0000-0000-0000-000000000000")

	if err == nil {
		t.Fatal("expected error")
	}

	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound")
	}
}
