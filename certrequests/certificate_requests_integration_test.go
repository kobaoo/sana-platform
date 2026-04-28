package certrequests

import (
	"testing"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/et"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
	"encore.app/db/ent/user"
)

// ════ CREATE INTEGRATION TESTS ════

func TestCreateCertificateRenewal_HRWithoutDzoDenied(t *testing.T) {
	makeAuthUser(t, authhandler.RoleHR)

	_, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected PermissionDenied")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestCreateCertificateRenewal_CertificateNotFound(t *testing.T) {
	_, kcID := makeAuthUser(t, authhandler.RoleHR)
	dzoID := makeDzo(t)

	et.OverrideAuthInfo(auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Role:           authhandler.RoleHR,
		DzoID:          dzoID.String(),
	})

	_, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected NotFound")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestCreateCertificateRenewal_InactiveCertificateNotFound(t *testing.T) {
	_, kcID := makeAuthUser(t, authhandler.RoleHR)
	dzoID := makeDzo(t)
	empID := makeEmployee(t, dzoID)
	certID := makeCertInDB(t, empID)

	_, err := Client.Certificate.
		UpdateOneID(certID).
		SetIsActive(false).
		Save(ctx())
	if err != nil {
		t.Fatalf("failed to deactivate certificate: %v", err)
	}

	et.OverrideAuthInfo(auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Role:           authhandler.RoleHR,
		DzoID:          dzoID.String(),
	})

	_, err = CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: certID,
	})
	if err == nil {
		t.Fatal("expected NotFound")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestCreateCertificateRenewal_CertificateOutsideDzoDenied(t *testing.T) {
	_, kcID := makeAuthUser(t, authhandler.RoleHR)
	hrDzoID := makeDzo(t)
	otherDzoID := makeDzo(t)
	empID := makeEmployee(t, otherDzoID)
	certID := makeCertInDB(t, empID)

	et.OverrideAuthInfo(auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Role:           authhandler.RoleHR,
		DzoID:          hrDzoID.String(),
	})

	_, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: certID,
	})
	if err == nil {
		t.Fatal("expected PermissionDenied")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestCreateCertificateRenewal_AutoCreatesInitiatorWhenMissing(t *testing.T) {
	dzoID := makeDzo(t)
	empID := makeEmployee(t, dzoID)
	certID := makeCertInDB(t, empID)
	kcID := "hr-autocreate-" + uuid.NewString()
	email := kcID + "@test.com"

	et.OverrideAuthInfo(auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Email:          email,
		Role:           authhandler.RoleHR,
		DzoID:          dzoID.String(),
	})

	resp, err := CreateCertificateRenewal(ctx(), &CreateCertificateRenewalRequest{
		EntityID: certID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %s", resp.Status)
	}

	u, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(kcID)).
		Only(ctx())
	if err != nil {
		t.Fatalf("expected auto-created user, got error: %v", err)
	}
	if resp.InitiatorID != u.ID {
		t.Fatalf("initiator_id mismatch: got %v, want %v", resp.InitiatorID, u.ID)
	}
}

// ════ LIST INTEGRATION TESTS ════

func TestListCertificateRenewals_InvalidInitiatorID(t *testing.T) {
	makeAuthUser(t, authhandler.RoleADM)

	_, err := ListCertificateRenewals(ctx(), &ListCertificateRenewalsParams{
		InitiatorID: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected InvalidArgument")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListCertificateRenewals_ReturnsOnlyCertificateRenewals(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	certReq, err := insertCertRenewal(ctx(), hrID, uuid.New())
	if err != nil {
		t.Fatalf("setup certificate renewal: %v", err)
	}

	_, err = Client.Request.Create().
		SetInitiatorID(hrID).
		SetEntityID(uuid.New()).
		SetEntityType("TRAINING_EVENT").
		SetStep(0).
		SetStatus("PENDING").
		Save(ctx())
	if err != nil {
		t.Fatalf("setup non-certificate request: %v", err)
	}

	resp, err := ListCertificateRenewals(ctx(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundCertReq := false
	for _, item := range resp.Items {
		if item.EntityType != entityTypeCertRenewal {
			t.Fatalf("expected only %s, got %s", entityTypeCertRenewal, item.EntityType)
		}
		if item.ID == certReq.ID {
			foundCertReq = true
		}
	}
	if !foundCertReq {
		t.Fatal("expected certificate renewal request in list")
	}
}

// ════ GET INTEGRATION TESTS ════

func TestGetCertificateRenewal_InvalidIDFormat(t *testing.T) {
	makeAuthUser(t, authhandler.RoleHR)

	_, err := GetCertificateRenewal(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected InvalidArgument")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

// ════ PATCH INTEGRATION TESTS ════

func TestPatchCertificateRenewalStatus_NilBody(t *testing.T) {
	makeAuthUser(t, authhandler.RoleADM)

	_, err := PatchCertificateRenewalStatus(ctx(), uuid.NewString(), nil)
	if err == nil {
		t.Fatal("expected InvalidArgument")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestPatchCertificateRenewalStatus_InvalidIDFormat(t *testing.T) {
	makeAuthUser(t, authhandler.RoleADM)

	_, err := PatchCertificateRenewalStatus(ctx(), "not-a-uuid", &PatchCertificateRenewalStatusRequest{
		Status: "APPROVED",
	})
	if err == nil {
		t.Fatal("expected InvalidArgument")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestPatchCertificateRenewalStatus_InvalidStatus(t *testing.T) {
	makeAuthUser(t, authhandler.RoleADM)

	_, err := PatchCertificateRenewalStatus(ctx(), uuid.NewString(), &PatchCertificateRenewalStatusRequest{
		Status: "IN_REVIEW",
	})
	if err == nil {
		t.Fatal("expected InvalidArgument")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestPatchCertificateRenewalStatus_NotFound(t *testing.T) {
	makeAuthUser(t, authhandler.RoleADM)

	_, err := PatchCertificateRenewalStatus(ctx(), uuid.NewString(), &PatchCertificateRenewalStatusRequest{
		Status: "APPROVED",
	})
	if err == nil {
		t.Fatal("expected NotFound")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestPatchCertificateRenewalStatus_RejectsWithWhitespaceLowercase(t *testing.T) {
	hrID, _ := makeAuthUser(t, authhandler.RoleHR)

	certReq, err := insertCertRenewal(ctx(), hrID, uuid.New())
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	makeAuthUser(t, authhandler.RoleSA)

	resp, err := PatchCertificateRenewalStatus(ctx(), certReq.ID.String(), &PatchCertificateRenewalStatusRequest{
		Status: " rejected ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "REJECTED" {
		t.Fatalf("expected REJECTED, got %s", resp.Status)
	}
}
