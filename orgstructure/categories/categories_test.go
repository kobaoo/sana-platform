package categories

import (
	"context"
	"strings"
	"testing"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/google/uuid"
)

func adminCtx() context.Context {
	return auth.WithContext(
		context.Background(),
		auth.UID("categories-admin"),
		&authhandler.AuthData{
			KeycloakUserID: "categories-admin",
			Role:           authhandler.RoleADM,
			CompanyID:      "00000000-0000-0000-0000-000000000001",
		},
	)
}

func employeeCtx() context.Context {
	return auth.WithContext(
		context.Background(),
		auth.UID("categories-employee"),
		&authhandler.AuthData{
			KeycloakUserID: "categories-employee",
			Role:           authhandler.RoleEMP,
			CompanyID:      "00000000-0000-0000-0000-000000000001",
		},
	)
}

func stringPtr(s string) *string {
	return &s
}

func makeCreateCategoryRequest() *CreateCategoryRequest {
	return &CreateCategoryRequest{
		Name:        "Category " + uuid.NewString(),
		Description: stringPtr("Safety-related courses"),
	}
}

func mustCreateCategory(t *testing.T) *Category {
	t.Helper()

	resp, err := CreateCategory(adminCtx(), makeCreateCategoryRequest())
	if err != nil {
		t.Fatalf("CreateCategory setup failed: %v", err)
	}
	if resp.Category == nil {
		t.Fatal("CreateCategory setup returned nil category")
	}

	return resp.Category
}

func requireErrCode(t *testing.T, err error, code errs.ErrCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %v error, got nil", code)
	}
	if errs.Code(err) != code {
		t.Fatalf("expected %v, got %v: %v", code, errs.Code(err), err)
	}
}

func TestCreateCategory_Success(t *testing.T) {
	req := makeCreateCategoryRequest()

	resp, err := CreateCategory(adminCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Category == nil {
		t.Fatal("expected category, got nil")
	}
	if resp.Category.ID == uuid.Nil {
		t.Error("expected generated id")
	}
	if resp.Category.Name != req.Name {
		t.Errorf("expected name %q, got %q", req.Name, resp.Category.Name)
	}
	if resp.Category.Description == nil || *resp.Category.Description != *req.Description {
		t.Errorf("expected description %q, got %v", *req.Description, resp.Category.Description)
	}
}

func TestCreateCategory_EmptyNameReturnsInvalidArgument(t *testing.T) {
	_, err := CreateCategory(adminCtx(), &CreateCategoryRequest{Name: "   "})
	requireErrCode(t, err, errs.InvalidArgument)
}

func TestCreateCategory_NameLongerThan100ReturnsInvalidArgument(t *testing.T) {
	_, err := CreateCategory(adminCtx(), &CreateCategoryRequest{Name: strings.Repeat("a", 101)})
	requireErrCode(t, err, errs.InvalidArgument)
}

func TestListCategories_Success(t *testing.T) {
	created := mustCreateCategory(t)

	resp, err := ListCategories(adminCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Categories == nil {
		t.Fatal("expected non-nil categories slice")
	}

	found := false
	for _, c := range resp.Categories {
		if c.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected created category %s in list", created.ID)
	}
}

func TestGetCategoryByID_Success(t *testing.T) {
	created := mustCreateCategory(t)

	resp, err := GetCategoryByID(adminCtx(), created.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Category == nil {
		t.Fatal("expected category, got nil")
	}
	if resp.Category.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, resp.Category.ID)
	}
}

func TestGetCategoryByID_InvalidUUIDReturnsInvalidArgument(t *testing.T) {
	_, err := GetCategoryByID(adminCtx(), "not-a-uuid")
	requireErrCode(t, err, errs.InvalidArgument)
}

func TestGetCategoryByID_NotFoundReturnsNotFound(t *testing.T) {
	_, err := GetCategoryByID(adminCtx(), uuid.NewString())
	requireErrCode(t, err, errs.NotFound)
}

func TestGetCategoriesByIDs_Success(t *testing.T) {
	first := mustCreateCategory(t)
	second := mustCreateCategory(t)

	resp, err := GetCategoriesByIDs(adminCtx(), &GetCategoriesByIDsRequest{
		CategoryIDs: []string{first.ID.String(), second.ID.String()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Categories == nil {
		t.Fatal("expected non-nil categories slice")
	}

	found := map[uuid.UUID]bool{}
	for _, c := range resp.Categories {
		found[c.ID] = true
	}
	if !found[first.ID] || !found[second.ID] {
		t.Fatalf("expected both categories in response, got %v", found)
	}
}

func TestGetCategoriesByIDs_EmptyListReturnsInvalidArgument(t *testing.T) {
	_, err := GetCategoriesByIDs(adminCtx(), &GetCategoriesByIDsRequest{})
	requireErrCode(t, err, errs.InvalidArgument)
}

func TestUpdateCategory_Success(t *testing.T) {
	created := mustCreateCategory(t)
	name := "Updated " + uuid.NewString()
	description := "Updated description"

	resp, err := UpdateCategory(adminCtx(), created.ID.String(), &UpdateCategoryRequest{
		Name:        &name,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Category.Name != name {
		t.Errorf("expected name %q, got %q", name, resp.Category.Name)
	}
	if resp.Category.Description == nil || *resp.Category.Description != description {
		t.Errorf("expected description %q, got %v", description, resp.Category.Description)
	}
}

func TestUpdateCategory_EmptyNameReturnsInvalidArgument(t *testing.T) {
	created := mustCreateCategory(t)
	empty := "   "

	_, err := UpdateCategory(adminCtx(), created.ID.String(), &UpdateCategoryRequest{Name: &empty})
	requireErrCode(t, err, errs.InvalidArgument)
}

func TestDeleteCategory_Success(t *testing.T) {
	created := mustCreateCategory(t)

	if err := DeleteCategory(adminCtx(), created.ID.String()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := GetCategoryByID(adminCtx(), created.ID.String())
	requireErrCode(t, err, errs.NotFound)
}

func TestCategory_UnsupportedRoleReturnsPermissionDenied(t *testing.T) {
	_, err := CreateCategory(employeeCtx(), makeCreateCategoryRequest())
	requireErrCode(t, err, errs.PermissionDenied)
}
