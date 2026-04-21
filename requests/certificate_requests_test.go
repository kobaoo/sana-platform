package requests

import (
	"testing"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/et"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
)

// makeAuthUser creates a DB user with a deterministic KC ID, then injects auth so
// auth.Data() returns the given role for the rest of the test.
// Returns the internal UUID and the KC ID.
func makeAuthUser(t *testing.T, role authhandler.UserRole) (uuid.UUID, string) {
	t.Helper()

	id := uuid.New()
	kcID := string(role) + "-" + id.String()

	_, err := Client.User.
		Create().
		SetID(id).
		SetRole(string(role)).
		SetEmail(string(role) + "-" + id.String() + "@test.com").
		SetKeycloakUserID(kcID).
		SetIsActive(true).
		SetIsOnboarded(true).
		Save(ctx())
	if err != nil {
		t.Fatalf("makeAuthUser(%q): %v", role, err)
	}

	et.OverrideAuthInfo(auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Role:           role,
	})

	return id, kcID
}

// ════ ТЗ TESTS — endpoint-level (5 обязательных сценариев) ════

// ТЗ-1: HR creates request via endpoint → success, status PENDING
func TestCreateCertificateRenewal_HRSuccess(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	resp, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %s", resp.Status)
	}
	if resp.InitiatorID != hrID {
		t.Fatalf("initiator_id mismatch: got %v, want %v", resp.InitiatorID, hrID)
	}
}

// ТЗ-2: create with empty entity_id via endpoint → InvalidArgument
// Валидация entity_id происходит до auth check, поэтому auth не нужен.
func TestCreateCertificateRenewal_EmptyEntityID(t *testing.T) {
	_, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected error for nil entity_id")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

// ТЗ-3: get nonexistent ID via endpoint → NotFound
func TestGetCertificateRenewal_NotFound(t *testing.T) {
	makeAuthUser(t, authhandler.RoleHR)

	_, err := GetCertificateRenewal(ctx(), uuid.New().String())
	if err == nil {
		t.Fatal("expected NotFound error")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected NotFound, got %v", errs.Code(err))
	}
}

// ТЗ-4: ADM changes PENDING → APPROVED via endpoint → success
func TestPatchCertificateRenewalStatus_Approved(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	cert, err := insertCertRenewal(ctx(), hrID, uuid.New())
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	makeAuthUser(t, authhandler.RoleADM)

	resp, err := PatchCertificateRenewalStatus(ctx(), cert.ID.String(), &PatchCertificateRenewalStatusRequest{
		Status: "APPROVED",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "APPROVED" {
		t.Fatalf("expected APPROVED, got %s", resp.Status)
	}
}

// ТЗ-5: second status change on already closed request via endpoint → InvalidArgument
func TestPatchCertificateRenewalStatus_AlreadyFinalized(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	cert, err := insertCertRenewal(ctx(), hrID, uuid.New())
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	makeAuthUser(t, authhandler.RoleADM)

	if _, err = PatchCertificateRenewalStatus(ctx(), cert.ID.String(), &PatchCertificateRenewalStatusRequest{
		Status: "APPROVED",
	}); err != nil {
		t.Fatalf("first transition failed: %v", err)
	}

	_, err = PatchCertificateRenewalStatus(ctx(), cert.ID.String(), &PatchCertificateRenewalStatusRequest{
		Status: "REJECTED",
	})
	if err == nil {
		t.Fatal("expected error on second transition")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

// ════ ROLE CHECK TESTS — endpoint-level ════

// не-HR не может создать request
func TestCreateCertificateRenewal_ADMDenied(t *testing.T) {
	makeAuthUser(t, authhandler.RoleADM)

	_, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected PermissionDenied")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

// EMP не может list certificate requests
func TestListCertificateRenewals_EMPDenied(t *testing.T) {
	makeAuthUser(t, authhandler.RoleEMP)

	_, err := ListCertificateRenewals(ctx(), nil)
	if err == nil {
		t.Fatal("expected PermissionDenied")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

// не-ADM/SA не может patch status
func TestPatchCertificateRenewalStatus_HRDenied(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	cert, err := insertCertRenewal(ctx(), hrID, uuid.New())
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err = PatchCertificateRenewalStatus(ctx(), cert.ID.String(), &PatchCertificateRenewalStatusRequest{
		Status: "APPROVED",
	})
	if err == nil {
		t.Fatal("expected PermissionDenied")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

// ════ ДОПОЛНИТЕЛЬНЫЕ TESTS — helper-level ════

// Чистые функции для role checks (быстрее endpoint-level тестов)
func TestCanCreateCertRenewal_OnlyHR(t *testing.T) {
	cases := map[string]bool{"HR": true, "ADM": false, "SA": false, "EMP": false, "": false}
	for role, want := range cases {
		if got := canCreateCertRenewal(role); got != want {
			t.Errorf("canCreateCertRenewal(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanViewCertRenewal_Roles(t *testing.T) {
	for _, role := range []string{"SA", "ADM", "HR"} {
		if !canViewCertRenewal(role) {
			t.Errorf("expected %q to be allowed", role)
		}
	}
	for _, role := range []string{"EMP", ""} {
		if canViewCertRenewal(role) {
			t.Errorf("expected %q to be denied", role)
		}
	}
}

func TestCanReviewCertRenewal_Roles(t *testing.T) {
	for _, role := range []string{"ADM", "SA"} {
		if !canReviewCertRenewal(role) {
			t.Errorf("expected %q to be allowed", role)
		}
	}
	for _, role := range []string{"HR", "EMP", ""} {
		if canReviewCertRenewal(role) {
			t.Errorf("expected %q to be denied", role)
		}
	}
}

// Get возвращает 404 если ID принадлежит Epic 4 (другой entity_type)
func TestGetCertificateRenewal_WrongEntityType(t *testing.T) {
	makeAuthUser(t, authhandler.RoleHR)

	tr, err := CreateRequest(ctx(), &CreateRequestRequest{
		EntityID:   uuid.New(),
		EntityType: "TRAINING_EVENT",
	})
	if err != nil {
		t.Fatalf("create training event request: %v", err)
	}

	_, err = queryCertRenewalByID(ctx(), tr.ID)
	if err == nil {
		t.Fatal("expected NotFound when querying TRAINING_EVENT as cert renewal")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected NotFound, got %v", errs.Code(err))
	}
}

// List фильтрует по initiator_id через endpoint
func TestListCertificateRenewals_FiltersByInitiatorID(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	admID := uuid.New()
	ensureUser(t, admID, "ADM")

	if _, err := insertCertRenewal(ctx(), hrID, uuid.New()); err != nil {
		t.Fatalf("setup hr record: %v", err)
	}
	if _, err := insertCertRenewal(ctx(), admID, uuid.New()); err != nil {
		t.Fatalf("setup adm record: %v", err)
	}

	resp, err := ListCertificateRenewals(ctx(), &ListCertificateRenewalsParams{
		InitiatorID: hrID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, item := range resp.Items {
		if item.InitiatorID != hrID {
			t.Errorf("filter returned wrong record: initiator_id %v", item.InitiatorID)
		}
	}
}

