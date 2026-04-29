package positions

import (
	"context"
	"testing"

	"encore.app/db/ent/dzoorganization"
	"encore.dev/beta/errs"
	"github.com/google/uuid"
)

// ════ HELPERS ════

const testDzoID = "22222222-2222-2222-2222-222222222222"

func makeDzo(t *testing.T, id string) {
	t.Helper()

	dzoID, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("invalid dzo id: %v", err)
	}

	clientID, err := uuid.Parse(testClientID)
	if err != nil {
		t.Fatalf("invalid client id: %v", err)
	}

	exists, err := Client.DzoOrganization.Query().
		Where(dzoorganization.ID(dzoID)).
		Exist(context.Background())
	if err != nil {
		t.Fatalf("failed to check dzo: %v", err)
	}
	if exists {
		return
	}

	_, err = Client.DzoOrganization.
		Create().
		SetID(dzoID).
		SetClientID(clientID).
		SetName("Test DZO").
		SetIsActive(true).
		Save(context.Background())
	if err != nil {
		t.Fatalf("failed to create dzo: %v", err)
	}
}

func makeDzoPositionTitle(t *testing.T, localTitle string) *DzoPositionTitle {
	t.Helper()

	makeDzo(t, testDzoID)

	resp, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      testDzoID,
		LocalTitle: localTitle,
	})
	if err != nil {
		t.Fatalf("makeDzoPositionTitle: %v", err)
	}

	return &resp.PositionTitle
}

// ════ CREATE ════

func TestCreateDzoPositionTitle_Success(t *testing.T) {
	makeDzo(t, testDzoID)

	resp, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      testDzoID,
		LocalTitle: "Backend Developer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.PositionTitle.DzoID != testDzoID {
		t.Errorf("expected dzo_id %q, got %q", testDzoID, resp.PositionTitle.DzoID)
	}
	if resp.PositionTitle.LocalTitle != "Backend Developer" {
		t.Errorf("expected local_title %q, got %q", "Backend Developer", resp.PositionTitle.LocalTitle)
	}
	if !resp.PositionTitle.IsActive {
		t.Error("expected newly created title to be active")
	}
	if resp.PositionTitle.IsDeleted {
		t.Error("expected newly created title not to be deleted")
	}
	if resp.PositionTitle.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestCreateDzoPositionTitle_WithGeneralPosition(t *testing.T) {
	makeDzo(t, testDzoID)

	gp := makeGeneralPosition(t, "General For DZO Title")

	resp, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:             testDzoID,
		GeneralPositionID: &gp.ID,
		LocalTitle:        "Local Backend Developer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.GeneralPositionID == nil {
		t.Fatal("expected general_position_id, got nil")
	}
	if *resp.PositionTitle.GeneralPositionID != gp.ID {
		t.Errorf("expected general_position_id %q, got %q", gp.ID, *resp.PositionTitle.GeneralPositionID)
	}
}

func TestCreateDzoPositionTitle_EmptyDzoID(t *testing.T) {
	_, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      "",
		LocalTitle: "Developer",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateDzoPositionTitle_EmptyLocalTitle(t *testing.T) {
	makeDzo(t, testDzoID)

	_, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      testDzoID,
		LocalTitle: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateDzoPositionTitle_WhitespaceOnlyLocalTitle(t *testing.T) {
	makeDzo(t, testDzoID)

	_, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      testDzoID,
		LocalTitle: "   ",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateDzoPositionTitle_InvalidDzoID(t *testing.T) {
	_, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      "not-a-uuid",
		LocalTitle: "Developer",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateDzoPositionTitle_InvalidGeneralPositionID(t *testing.T) {
	makeDzo(t, testDzoID)

	invalidID := "not-a-uuid"

	_, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:             testDzoID,
		GeneralPositionID: &invalidID,
		LocalTitle:        "Developer",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateDzoPositionTitle_DuplicateLocalTitleInSameDzo(t *testing.T) {
	makeDzoPositionTitle(t, "Duplicate Local Title")

	_, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:      testDzoID,
		LocalTitle: "Duplicate Local Title",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

// ════ GET ════

func TestGetDzoPositionTitle_Success(t *testing.T) {
	title := makeDzoPositionTitle(t, "Get DZO Title")

	resp, err := GetDzoPositionTitle(ctx(), title.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.ID != title.ID {
		t.Errorf("expected ID %q, got %q", title.ID, resp.PositionTitle.ID)
	}
	if resp.PositionTitle.LocalTitle != title.LocalTitle {
		t.Errorf("expected local_title %q, got %q", title.LocalTitle, resp.PositionTitle.LocalTitle)
	}
	if resp.PositionTitle.IsDeleted {
		t.Error("expected title not to be deleted")
	}
}

func TestGetDzoPositionTitle_InvalidIDFormat(t *testing.T) {
	_, err := GetDzoPositionTitle(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestGetDzoPositionTitle_NotFound(t *testing.T) {
	_, err := GetDzoPositionTitle(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestGetDzoPositionTitle_DeletedNotFound(t *testing.T) {
	title := makeDzoPositionTitle(t, "Deleted Get DZO Title")

	if _, err := DeleteDzoPositionTitle(ctx(), title.ID); err != nil {
		t.Fatalf("failed to delete title: %v", err)
	}

	_, err := GetDzoPositionTitle(ctx(), title.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestListDzoPositionTitles_ReturnsOnlyNotDeleted(t *testing.T) {
	active := makeDzoPositionTitle(t, "Active DZO Title")
	deleted := makeDzoPositionTitle(t, "Deleted DZO Title")

	if _, err := DeleteDzoPositionTitle(ctx(), deleted.ID); err != nil {
		t.Fatalf("failed to delete title: %v", err)
	}

	resp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	for _, title := range resp.PositionTitles {
		if title.ID == deleted.ID {
			t.Error("deleted title should not appear in list")
		}
		if title.ID == active.ID {
			foundActive = true
		}
	}

	if !foundActive {
		t.Error("active title should appear in list")
	}
}

func TestListDzoPositionTitles_SearchByLocalTitle(t *testing.T) {
	target := makeDzoPositionTitle(t, "Unique Local Search Title")
	makeDzoPositionTitle(t, "Another Local Title")

	resp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{
		Search: "Unique Local Search",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, title := range resp.PositionTitles {
		if title.ID == target.ID {
			found = true
		}
		if title.LocalTitle == "Another Local Title" {
			t.Error("title not matching search should not appear")
		}
	}

	if !found {
		t.Error("expected searched title to appear")
	}
}

func TestListDzoPositionTitles_FilterByDzoID(t *testing.T) {
	target := makeDzoPositionTitle(t, "Filter By DZO Title")

	resp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{
		DzoID: testDzoID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, title := range resp.PositionTitles {
		if title.ID == target.ID {
			found = true
		}
		if title.DzoID != testDzoID {
			t.Errorf("expected only dzo_id %q, got %q", testDzoID, title.DzoID)
		}
	}

	if !found {
		t.Error("expected filtered title to appear")
	}
}

func TestListDzoPositionTitles_FilterByInvalidDzoID(t *testing.T) {
	_, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{
		DzoID: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListDzoPositionTitles_FilterByGeneralPositionID(t *testing.T) {
	makeDzo(t, testDzoID)

	gp := makeGeneralPosition(t, "Filter General Position")
	targetResp, err := CreateDzoPositionTitle(ctx(), &CreateDzoPositionTitleRequest{
		DzoID:             testDzoID,
		GeneralPositionID: &gp.ID,
		LocalTitle:        "Filtered General Local Title",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	resp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{
		GeneralPositionID: gp.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, title := range resp.PositionTitles {
		if title.ID == targetResp.PositionTitle.ID {
			found = true
		}
		if title.GeneralPositionID == nil || *title.GeneralPositionID != gp.ID {
			t.Errorf("expected only general_position_id %q, got %v", gp.ID, title.GeneralPositionID)
		}
	}

	if !found {
		t.Error("expected filtered title to appear")
	}
}

func TestListDzoPositionTitles_FilterByInvalidGeneralPositionID(t *testing.T) {
	_, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{
		GeneralPositionID: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListDzoPositionTitles_TotalMatchesLength(t *testing.T) {
	makeDzoPositionTitle(t, "Total DZO Title 1")
	makeDzoPositionTitle(t, "Total DZO Title 2")

	resp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != len(resp.PositionTitles) {
		t.Errorf("Total=%d does not match len=%d", resp.Total, len(resp.PositionTitles))
	}
}

func TestListDzoPositionTitles_ReturnsEmptySliceNotNil(t *testing.T) {
	resp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{
		Search: "very-unique-dzo-title-that-should-not-exist",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitles == nil {
		t.Error("expected empty slice, got nil")
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

// ════ UPDATE ════

func TestUpdateDzoPositionTitle_SuccessUpdatesOnlyProvidedFields(t *testing.T) {
	title := makeDzoPositionTitle(t, "Before DZO Update")
	newTitle := "After DZO Update"

	resp, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		LocalTitle: &newTitle,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.LocalTitle != newTitle {
		t.Errorf("expected local_title %q, got %q", newTitle, resp.PositionTitle.LocalTitle)
	}
	if resp.PositionTitle.DzoID != title.DzoID {
		t.Errorf("expected dzo_id unchanged, got %q", resp.PositionTitle.DzoID)
	}
}

func TestUpdateDzoPositionTitle_UpdateGeneralPositionOnly(t *testing.T) {
	title := makeDzoPositionTitle(t, "General Position Before")
	gp := makeGeneralPosition(t, "Updated General Position")

	resp, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		GeneralPositionID: &gp.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.LocalTitle != title.LocalTitle {
		t.Errorf("expected local_title unchanged, got %q", resp.PositionTitle.LocalTitle)
	}
	if resp.PositionTitle.GeneralPositionID == nil || *resp.PositionTitle.GeneralPositionID != gp.ID {
		t.Errorf("expected general_position_id %q, got %v", gp.ID, resp.PositionTitle.GeneralPositionID)
	}
}

func TestUpdateDzoPositionTitle_UpdateIsActiveOnly(t *testing.T) {
	title := makeDzoPositionTitle(t, "Active Before Update")
	isActive := false

	resp, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.IsActive {
		t.Error("expected is_active false")
	}
	if resp.PositionTitle.LocalTitle != title.LocalTitle {
		t.Errorf("expected local_title unchanged, got %q", resp.PositionTitle.LocalTitle)
	}
}

func TestUpdateDzoPositionTitle_EmptyRequestChangesNothing(t *testing.T) {
	title := makeDzoPositionTitle(t, "Stable DZO Title")

	resp, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PositionTitle.LocalTitle != title.LocalTitle {
		t.Errorf("local_title changed: %q -> %q", title.LocalTitle, resp.PositionTitle.LocalTitle)
	}
}

func TestUpdateDzoPositionTitle_EmptyLocalTitle(t *testing.T) {
	title := makeDzoPositionTitle(t, "Empty DZO Title Update")
	newTitle := ""

	_, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		LocalTitle: &newTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateDzoPositionTitle_WhitespaceOnlyLocalTitle(t *testing.T) {
	title := makeDzoPositionTitle(t, "Whitespace DZO Title Update")
	newTitle := "   "

	_, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		LocalTitle: &newTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateDzoPositionTitle_InvalidIDFormat(t *testing.T) {
	newTitle := "Ghost"

	_, err := UpdateDzoPositionTitle(ctx(), "not-a-uuid", &UpdateDzoPositionTitleRequest{
		LocalTitle: &newTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateDzoPositionTitle_InvalidGeneralPositionID(t *testing.T) {
	title := makeDzoPositionTitle(t, "Invalid General Position Update")
	invalidID := "not-a-uuid"

	_, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		GeneralPositionID: &invalidID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateDzoPositionTitle_NotFound(t *testing.T) {
	newTitle := "Ghost"

	_, err := UpdateDzoPositionTitle(ctx(), "00000000-0000-0000-0000-000000000000", &UpdateDzoPositionTitleRequest{
		LocalTitle: &newTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestUpdateDzoPositionTitle_DuplicateLocalTitle(t *testing.T) {
	makeDzoPositionTitle(t, "Existing DZO Local Title")
	second := makeDzoPositionTitle(t, "Second DZO Local Title")

	existingTitle := "Existing DZO Local Title"
	_, err := UpdateDzoPositionTitle(ctx(), second.ID, &UpdateDzoPositionTitleRequest{
		LocalTitle: &existingTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

func TestUpdateDzoPositionTitle_DeletedNotFound(t *testing.T) {
	title := makeDzoPositionTitle(t, "Deleted DZO Update Title")
	newTitle := "Should Not Update"

	if _, err := DeleteDzoPositionTitle(ctx(), title.ID); err != nil {
		t.Fatalf("failed to delete title: %v", err)
	}

	_, err := UpdateDzoPositionTitle(ctx(), title.ID, &UpdateDzoPositionTitleRequest{
		LocalTitle: &newTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ DELETE ════

func TestDeleteDzoPositionTitle_SuccessSoftDeletes(t *testing.T) {
	title := makeDzoPositionTitle(t, "Delete DZO Title")

	resp, err := DeleteDzoPositionTitle(ctx(), title.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message == "" {
		t.Error("expected non-empty message")
	}

	listResp, err := ListDzoPositionTitles(ctx(), &ListDzoPositionTitlesRequest{})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	for _, item := range listResp.PositionTitles {
		if item.ID == title.ID {
			t.Error("soft-deleted title should not appear in list")
		}
	}

	_, err = GetDzoPositionTitle(ctx(), title.ID)
	if err == nil {
		t.Fatal("expected deleted title to be not found")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteDzoPositionTitle_InvalidIDFormat(t *testing.T) {
	_, err := DeleteDzoPositionTitle(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestDeleteDzoPositionTitle_NotFound(t *testing.T) {
	_, err := DeleteDzoPositionTitle(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteDzoPositionTitle_DoubleDelete(t *testing.T) {
	title := makeDzoPositionTitle(t, "Double Delete DZO Title")

	if _, err := DeleteDzoPositionTitle(ctx(), title.ID); err != nil {
		t.Fatalf("first delete failed: %v", err)
	}

	_, err := DeleteDzoPositionTitle(ctx(), title.ID)
	if err == nil {
		t.Fatal("expected error on second delete, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound on second delete, got %v", errs.Code(err))
	}
}
