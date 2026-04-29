// positions/general_position_test.go
package positions

import (
	"context"
	"testing"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

// ════ HELPERS ════

const testClientID = "11111111-1111-1111-1111-111111111111"

func ctx() context.Context {
	return auth.WithContext(
		context.Background(),
		auth.UID("test-user"),
		&authhandler.AuthData{
			Role:      authhandler.RoleSA,
			CompanyID: testClientID,
		},
	)
}

func strPtr(s string) *string {
	return &s
}

func makeGeneralPosition(t *testing.T, name string) *GeneralPosition {
	t.Helper()

	resp, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name:        name,
		Description: strPtr("test description"),
	})
	if err != nil {
		t.Fatalf("makeGeneralPosition: %v", err)
	}

	return &resp.GeneralPosition
}

// ════ CREATE ════

func TestCreateGeneralPosition_Success(t *testing.T) {
	desc := "Position description"

	resp, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name:        "Backend Developer",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPosition.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.GeneralPosition.Name != "Backend Developer" {
		t.Errorf("expected name %q, got %q", "Backend Developer", resp.GeneralPosition.Name)
	}
	if resp.GeneralPosition.Description == nil || *resp.GeneralPosition.Description != desc {
		t.Errorf("expected description %q, got %v", desc, resp.GeneralPosition.Description)
	}
	if resp.GeneralPosition.IsDeleted {
		t.Error("expected newly created position not to be deleted")
	}
	if resp.GeneralPosition.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestCreateGeneralPosition_SuccessWithoutDescription(t *testing.T) {
	resp, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name: "Frontend Developer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPosition.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.GeneralPosition.Name != "Frontend Developer" {
		t.Errorf("expected name %q, got %q", "Frontend Developer", resp.GeneralPosition.Name)
	}
	if resp.GeneralPosition.Description != nil {
		t.Errorf("expected nil description, got %v", resp.GeneralPosition.Description)
	}
}

func TestCreateGeneralPosition_EmptyName(t *testing.T) {
	_, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateGeneralPosition_WhitespaceOnlyName(t *testing.T) {
	_, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name: "   ",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateGeneralPosition_DuplicateName(t *testing.T) {
	makeGeneralPosition(t, "Duplicate Position")

	_, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name: "Duplicate Position",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

// ════ GET ════

func TestGetGeneralPosition_Success(t *testing.T) {
	pos := makeGeneralPosition(t, "Get Position")

	resp, err := GetGeneralPosition(ctx(), pos.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPosition.ID != pos.ID {
		t.Errorf("expected ID %q, got %q", pos.ID, resp.GeneralPosition.ID)
	}
	if resp.GeneralPosition.Name != pos.Name {
		t.Errorf("expected name %q, got %q", pos.Name, resp.GeneralPosition.Name)
	}
	if resp.GeneralPosition.IsDeleted {
		t.Error("expected position not to be deleted")
	}
}

func TestGetGeneralPosition_InvalidIDFormat(t *testing.T) {
	_, err := GetGeneralPosition(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestGetGeneralPosition_NotFound(t *testing.T) {
	_, err := GetGeneralPosition(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestGetGeneralPosition_DeletedNotFound(t *testing.T) {
	pos := makeGeneralPosition(t, "Deleted Get Position")

	if _, err := DeleteGeneralPosition(ctx(), pos.ID); err != nil {
		t.Fatalf("failed to delete position: %v", err)
	}

	_, err := GetGeneralPosition(ctx(), pos.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestListGeneralPositions_ReturnsOnlyNotDeleted(t *testing.T) {
	active := makeGeneralPosition(t, "Active Position")
	deleted := makeGeneralPosition(t, "Deleted Position")

	if _, err := DeleteGeneralPosition(ctx(), deleted.ID); err != nil {
		t.Fatalf("failed to delete position: %v", err)
	}

	resp, err := ListGeneralPositions(ctx(), &ListGeneralPositionsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	for _, p := range resp.GeneralPositions {
		if p.ID == deleted.ID {
			t.Error("deleted position should not appear in list")
		}
		if p.ID == active.ID {
			foundActive = true
		}
	}

	if !foundActive {
		t.Error("active position should appear in list")
	}
}

func TestListGeneralPositions_SearchByName(t *testing.T) {
	target := makeGeneralPosition(t, "Unique Search Position")
	makeGeneralPosition(t, "Another Position")

	resp, err := ListGeneralPositions(ctx(), &ListGeneralPositionsRequest{
		Search: "Unique Search",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, p := range resp.GeneralPositions {
		if p.ID == target.ID {
			found = true
		}
		if p.Name == "Another Position" {
			t.Error("position not matching search should not appear")
		}
	}

	if !found {
		t.Error("expected searched position to appear")
	}
}

func TestListGeneralPositions_SearchByDescription(t *testing.T) {
	desc := "Special backend architecture role"

	respCreate, err := CreateGeneralPosition(ctx(), &CreateGeneralPositionRequest{
		Name:        "Description Search Position",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	resp, err := ListGeneralPositions(ctx(), &ListGeneralPositionsRequest{
		Search: "backend architecture",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, p := range resp.GeneralPositions {
		if p.ID == respCreate.GeneralPosition.ID {
			found = true
		}
	}

	if !found {
		t.Error("expected position to be found by description")
	}
}

func TestListGeneralPositions_TotalMatchesLength(t *testing.T) {
	makeGeneralPosition(t, "Total Position 1")
	makeGeneralPosition(t, "Total Position 2")

	resp, err := ListGeneralPositions(ctx(), &ListGeneralPositionsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != len(resp.GeneralPositions) {
		t.Errorf("Total=%d does not match len=%d", resp.Total, len(resp.GeneralPositions))
	}
}

func TestListGeneralPositions_ReturnsEmptySliceNotNil(t *testing.T) {
	resp, err := ListGeneralPositions(ctx(), &ListGeneralPositionsRequest{Search: "very-unique-search-that-should-not-exist"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPositions == nil {
		t.Error("expected empty slice, got nil")
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

// ════ UPDATE ════

func TestUpdateGeneralPosition_SuccessUpdatesOnlyProvidedFields(t *testing.T) {
	pos := makeGeneralPosition(t, "Before Update")
	newName := "After Update"

	resp, err := UpdateGeneralPosition(ctx(), pos.ID, &UpdateGeneralPositionRequest{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPosition.Name != newName {
		t.Errorf("expected name %q, got %q", newName, resp.GeneralPosition.Name)
	}
	if resp.GeneralPosition.Description == nil || pos.Description == nil {
		t.Fatal("expected descriptions not to be nil")
	}
	if *resp.GeneralPosition.Description != *pos.Description {
		t.Errorf("expected description unchanged, got %v", resp.GeneralPosition.Description)
	}
}

func TestUpdateGeneralPosition_UpdateDescriptionOnly(t *testing.T) {
	pos := makeGeneralPosition(t, "Description Before")
	newDesc := "New description"

	resp, err := UpdateGeneralPosition(ctx(), pos.ID, &UpdateGeneralPositionRequest{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPosition.Name != pos.Name {
		t.Errorf("expected name unchanged, got %q", resp.GeneralPosition.Name)
	}
	if resp.GeneralPosition.Description == nil || *resp.GeneralPosition.Description != newDesc {
		t.Errorf("expected description %q, got %v", newDesc, resp.GeneralPosition.Description)
	}
}

func TestUpdateGeneralPosition_EmptyRequestChangesNothing(t *testing.T) {
	pos := makeGeneralPosition(t, "Stable Position")

	resp, err := UpdateGeneralPosition(ctx(), pos.ID, &UpdateGeneralPositionRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GeneralPosition.Name != pos.Name {
		t.Errorf("name changed: %q -> %q", pos.Name, resp.GeneralPosition.Name)
	}
}

func TestUpdateGeneralPosition_EmptyName(t *testing.T) {
	pos := makeGeneralPosition(t, "Empty Name Update")
	newName := ""

	_, err := UpdateGeneralPosition(ctx(), pos.ID, &UpdateGeneralPositionRequest{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateGeneralPosition_WhitespaceOnlyName(t *testing.T) {
	pos := makeGeneralPosition(t, "Whitespace Name Update")
	newName := "   "

	_, err := UpdateGeneralPosition(ctx(), pos.ID, &UpdateGeneralPositionRequest{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateGeneralPosition_InvalidIDFormat(t *testing.T) {
	newName := "Ghost"

	_, err := UpdateGeneralPosition(ctx(), "not-a-uuid", &UpdateGeneralPositionRequest{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateGeneralPosition_NotFound(t *testing.T) {
	newName := "Ghost"

	_, err := UpdateGeneralPosition(ctx(), "00000000-0000-0000-0000-000000000000", &UpdateGeneralPositionRequest{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestUpdateGeneralPosition_DuplicateName(t *testing.T) {
	makeGeneralPosition(t, "Existing Position Name")
	second := makeGeneralPosition(t, "Second Position Name")

	existingName := "Existing Position Name"
	_, err := UpdateGeneralPosition(ctx(), second.ID, &UpdateGeneralPositionRequest{
		Name: &existingName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

func TestUpdateGeneralPosition_DeletedNotFound(t *testing.T) {
	pos := makeGeneralPosition(t, "Deleted Update Position")
	newName := "Should Not Update"

	if _, err := DeleteGeneralPosition(ctx(), pos.ID); err != nil {
		t.Fatalf("failed to delete position: %v", err)
	}

	_, err := UpdateGeneralPosition(ctx(), pos.ID, &UpdateGeneralPositionRequest{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ DELETE ════

func TestDeleteGeneralPosition_SuccessSoftDeletes(t *testing.T) {
	pos := makeGeneralPosition(t, "Delete Position")

	resp, err := DeleteGeneralPosition(ctx(), pos.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message == "" {
		t.Error("expected non-empty message")
	}

	listResp, err := ListGeneralPositions(ctx(), &ListGeneralPositionsRequest{})

	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	for _, p := range listResp.GeneralPositions {
		if p.ID == pos.ID {
			t.Error("soft-deleted position should not appear in list")
		}
	}

	_, err = GetGeneralPosition(ctx(), pos.ID)
	if err == nil {
		t.Fatal("expected deleted position to be not found")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteGeneralPosition_InvalidIDFormat(t *testing.T) {
	_, err := DeleteGeneralPosition(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestDeleteGeneralPosition_NotFound(t *testing.T) {
	_, err := DeleteGeneralPosition(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteGeneralPosition_DoubleDelete(t *testing.T) {
	pos := makeGeneralPosition(t, "Double Delete Position")

	if _, err := DeleteGeneralPosition(ctx(), pos.ID); err != nil {
		t.Fatalf("first delete failed: %v", err)
	}

	_, err := DeleteGeneralPosition(ctx(), pos.ID)
	if err == nil {
		t.Fatal("expected error on second delete, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound on second delete, got %v", errs.Code(err))
	}
}
