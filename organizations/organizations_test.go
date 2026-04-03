package organizations

import (
	"context"
	"testing"

	"encore.dev/beta/errs"
)

// ════ HELPERS ════

func ctx() context.Context {
	return context.Background()
}

func makeOrg(t *testing.T, name, code string, orgType OrgType) *Organization {
	t.Helper()
	resp, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name: name,
		Code: code,
		Type: orgType,
	})
	if err != nil {
		t.Fatalf("makeOrg: %v", err)
	}
	return &resp.Organization
}

// ════ CREATE ════

func TestCreateOrg_Success(t *testing.T) {
	resp, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name: "Acme Corp",
		Code: "ACME",
		Type: OrgTypeCompany,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Organization.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Organization.Name != "Acme Corp" {
		t.Errorf("expected name 'Acme Corp', got %q", resp.Organization.Name)
	}
}

func TestCreateOrg_SuccessWithParentID(t *testing.T) {
	parent := makeOrg(t, "Parent Co", "PARENTCO", OrgTypeCompany)

	resp, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name:     "Child Dept",
		Code:     "CHILDDEPT",
		Type:     OrgTypeDepartment,
		ParentID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Organization.ParentID == nil || *resp.Organization.ParentID != parent.ID {
		t.Errorf("expected parent_id %q, got %v", parent.ID, resp.Organization.ParentID)
	}
}

func TestCreateOrg_EmptyName(t *testing.T) {
	_, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name: "",
		Code: "NONAME",
		Type: OrgTypeCompany,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateOrg_EmptyCode(t *testing.T) {
	_, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name: "No Code Org",
		Code: "",
		Type: OrgTypeCompany,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateOrg_InvalidType(t *testing.T) {
	_, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name: "Bad Type Org",
		Code: "BADTYPE",
		Type: OrgType("invalid"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateOrg_DuplicateCode(t *testing.T) {
	makeOrg(t, "Original Org", "DUPCODE", OrgTypeCompany)

	_, err := CreateOrg(ctx(), &CreateOrgRequest{
		Name: "Duplicate Org",
		Code: "DUPCODE",
		Type: OrgTypeSubsidiary,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

// ════ GET ════

func TestGetOrg_Success(t *testing.T) {
	org := makeOrg(t, "Get Me", "GETME", OrgTypeCompany)

	resp, err := GetOrg(ctx(), org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Organization.ID != org.ID {
		t.Errorf("expected ID %q, got %q", org.ID, resp.Organization.ID)
	}
}

func TestGetOrg_NotFound(t *testing.T) {
	_, err := GetOrg(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestListOrgs_ReturnsOnlyActiveOrgs(t *testing.T) {
	active := makeOrg(t, "Active Org", "ACTIVELIST", OrgTypeCompany)
	toDelete := makeOrg(t, "To Delete", "DELETELIST", OrgTypeSubsidiary)

	if _, err := DeleteOrg(ctx(), toDelete.ID); err != nil {
		t.Fatalf("failed to delete org: %v", err)
	}

	resp, err := ListOrgs(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	for _, o := range resp.Organizations {
		if o.ID == toDelete.ID {
			t.Errorf("deleted org should not appear in list")
		}
		if o.ID == active.ID {
			foundActive = true
		}
	}
	if !foundActive {
		t.Error("active org should appear in list")
	}
}

func TestListOrgs_ReturnsEmptySliceNotNil(t *testing.T) {
	resp, err := ListOrgs(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Organizations == nil {
		t.Error("expected []Organization{}, got nil")
	}
}

// ════ UPDATE ════

func TestUpdateOrg_SuccessUpdatesOnlyProvidedFields(t *testing.T) {
	org := makeOrg(t, "Before Update", "BEFOREUPD", OrgTypeCompany)
	newName := "After Update"

	resp, err := UpdateOrg(ctx(), org.ID, &UpdateOrgRequest{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Organization.Name != newName {
		t.Errorf("expected name %q, got %q", newName, resp.Organization.Name)
	}
	if resp.Organization.Code != org.Code {
		t.Errorf("expected code %q to be unchanged, got %q", org.Code, resp.Organization.Code)
	}
	if resp.Organization.Type != org.Type {
		t.Errorf("expected type %q to be unchanged, got %q", org.Type, resp.Organization.Type)
	}
}

func TestUpdateOrg_NotFound(t *testing.T) {
	newName := "Ghost"
	_, err := UpdateOrg(ctx(), "00000000-0000-0000-0000-000000000000", &UpdateOrgRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestUpdateOrg_DuplicateCode(t *testing.T) {
	makeOrg(t, "First Org", "FIRSTORG", OrgTypeCompany)
	second := makeOrg(t, "Second Org", "SECONDORG", OrgTypeSubsidiary)

	existingCode := "FIRSTORG"
	_, err := UpdateOrg(ctx(), second.ID, &UpdateOrgRequest{Code: &existingCode})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

// ════ DELETE ════

func TestDeleteOrg_SuccessSoftDeletes(t *testing.T) {
	org := makeOrg(t, "To Soft Delete", "SOFTDEL", OrgTypeCompany)

	resp, err := DeleteOrg(ctx(), org.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}

	// Should disappear from ListOrgs
	listResp, err := ListOrgs(ctx())
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	for _, o := range listResp.Organizations {
		if o.ID == org.ID {
			t.Error("soft-deleted org should not appear in list")
		}
	}

	// But GetOrg should still find it
	getResp, err := GetOrg(ctx(), org.ID)
	if err != nil {
		t.Fatalf("GetOrg after soft delete should succeed: %v", err)
	}
	if getResp.Organization.IsActive {
		t.Error("org should be inactive after soft delete")
	}
}

func TestDeleteOrg_NotFound(t *testing.T) {
	_, err := DeleteOrg(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteOrg_DoubleDelete(t *testing.T) {
	org := makeOrg(t, "Double Delete", "DOUBLEDEL", OrgTypeCompany)

	if _, err := DeleteOrg(ctx(), org.ID); err != nil {
		t.Fatalf("first delete failed: %v", err)
	}

	_, err := DeleteOrg(ctx(), org.ID)
	if err == nil {
		t.Fatal("expected error on second delete, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound on second delete, got %v", errs.Code(err))
	}
}
