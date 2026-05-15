package certificates

import (
	"testing"
	"time"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/et"
	"github.com/google/uuid"
)

// ════ HELPERS ════

func withCertAuth(t *testing.T, role authhandler.UserRole, keycloakUserID string, dzoID string) {
	t.Helper()
	if keycloakUserID == "" {
		keycloakUserID = uuid.NewString()
	}
	et.OverrideAuthInfo(auth.UID(keycloakUserID), &authhandler.AuthData{
		KeycloakUserID: keycloakUserID,
		Email:          keycloakUserID + "@example.com",
		Role:           role,
		DzoID:          dzoID,
	})
}

func makeCertOrgData(t *testing.T, role string) (employeeID uuid.UUID, dzoID uuid.UUID, keycloakUserID string) {
	t.Helper()

	clientRow, err := Client.Company.Create().
		SetName("Test Client " + uuid.NewString()).
		SetDomain(uuid.NewString() + ".local").
		SetLanguage("ru").
		SetUserLimit(100).
		Save(ctx())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	dzoRow, err := Client.DzoOrganization.Create().
		SetClientID(clientRow.ID).
		SetName("Test DZO " + uuid.NewString()).
		Save(ctx())
	if err != nil {
		t.Fatalf("create dzo: %v", err)
	}

	keycloakUserID = uuid.NewString()
	email := keycloakUserID + "@example.com"

	userRow, err := Client.User.Create().
		SetKeycloakUserID(keycloakUserID).
		SetEmail(email).
		SetRole(role).
		SetDzoID(dzoRow.ID).
		SetClientID(clientRow.ID).
		Save(ctx())
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	empRow, err := Client.Employee.Create().
		SetClientID(clientRow.ID).
		SetDzoID(dzoRow.ID).
		SetFullName("Test Employee " + uuid.NewString()).
		SetEmail(email).
		SetUserID(userRow.ID).
		Save(ctx())
	if err != nil {
		t.Fatalf("create employee: %v", err)
	}

	return empRow.ID, dzoRow.ID, keycloakUserID
}

func makeCertForEmployee(t *testing.T, employeeID uuid.UUID, title string, expiryDate *time.Time) *Certificate {
	t.Helper()
	withADMAuth(t)

	resp, err := Create(ctx(), &CreateRequest{
		EmployeeID: employeeID,
		Type:       "EXTERNAL",
		Title:      title,
		IssuedDate: time.Now().AddDate(0, -1, 0),
		ExpiryDate: expiryDate,
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return &resp.Certificate
}

// ════ CREATE ════

func TestCreate_EMPDenied(t *testing.T) {
	withCertAuth(t, authhandler.RoleEMP, "", "")

	_, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      "Employee Denied",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestCreate_MissingIssuedDate(t *testing.T) {
	withADMAuth(t)

	_, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "EXTERNAL",
		Title:      "No Date",
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreate_InvalidType(t *testing.T) {
	withADMAuth(t)

	_, err := Create(ctx(), &CreateRequest{
		EmployeeID: uuid.New(),
		Type:       "INTERNAL",
		Title:      "Wrong Type",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestList_EMPDenied(t *testing.T) {
	withCertAuth(t, authhandler.RoleEMP, "", "")

	_, err := List(ctx(), &ListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestList_HRWithoutDzoDenied(t *testing.T) {
	withCertAuth(t, authhandler.RoleHR, "", "")

	_, err := List(ctx(), &ListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestList_HRScopesCertificatesByDzo(t *testing.T) {
	empA, dzoA, _ := makeCertOrgData(t, "EMP")
	empB, _, _ := makeCertOrgData(t, "EMP")

	certA := makeCertForEmployee(t, empA, "DZO A Cert", nil)
	certB := makeCertForEmployee(t, empB, "DZO B Cert", nil)

	withCertAuth(t, authhandler.RoleHR, "", dzoA.String())

	resp, err := List(ctx(), &ListParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundA := false
	for _, c := range resp.Certificates {
		if c.ID == certB.ID {
			t.Fatal("HR should not see certificate from another DZO")
		}
		if c.ID == certA.ID {
			foundA = true
		}
	}
	if !foundA {
		t.Fatal("expected HR to see certificate from own DZO")
	}
}

// ════ MY CERTIFICATES ════

func TestMyCertificates_NoLinkedEmployeeReturnsEmpty(t *testing.T) {
	withCertAuth(t, authhandler.RoleEMP, uuid.NewString(), "")

	resp, err := MyCertificates(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
	if len(resp.Certificates) != 0 {
		t.Fatalf("expected 0 certificates, got %d", len(resp.Certificates))
	}
}

func TestMyCertificates_ReturnsOnlyCurrentEmployeeActiveCertificates(t *testing.T) {
	empID, _, keycloakUserID := makeCertOrgData(t, "EMP")
	active := makeCertForEmployee(t, empID, "Current Employee Active", nil)
	deleted := makeCertForEmployee(t, empID, "Current Employee Deleted", nil)
	_, err := Delete(ctx(), deleted.ID)
	if err != nil {
		t.Fatalf("delete certificate: %v", err)
	}

	otherEmpID, _, _ := makeCertOrgData(t, "EMP")
	other := makeCertForEmployee(t, otherEmpID, "Other Employee Cert", nil)

	withCertAuth(t, authhandler.RoleEMP, keycloakUserID, "")

	resp, err := MyCertificates(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	for _, c := range resp.Certificates {
		if c.ID == deleted.ID {
			t.Fatal("deleted certificate should not be returned")
		}
		if c.ID == other.ID {
			t.Fatal("other employee certificate should not be returned")
		}
		if c.ID == active.ID {
			foundActive = true
		}
	}
	if !foundActive {
		t.Fatal("expected active certificate for current employee")
	}
}

// ════ UPDATE ════

func TestUpdate_NotFound(t *testing.T) {
	withADMAuth(t)

	_, err := Update(ctx(), uuid.NewString(), &UpdateRequest{
		Type:       "EXTERNAL",
		Title:      "Not Found",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestUpdate_DeletedCertificateNotFound(t *testing.T) {
	cert := makeCert(t, "Deleted Update")
	_, err := Delete(ctx(), cert.ID)
	if err != nil {
		t.Fatalf("delete certificate: %v", err)
	}

	withADMAuth(t)
	_, err = Update(ctx(), cert.ID, &UpdateRequest{
		Type:       "EXTERNAL",
		Title:      "Should Not Update",
		IssuedDate: time.Now(),
		EntityType: "TRAINING_EVENT",
		EntityID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ LIST EXPIRING ════

func TestListExpiring_MissingDzoID(t *testing.T) {
	withADMAuth(t)

	_, err := ListExpiring(ctx(), &ExpiringParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListExpiring_InvalidDzoID(t *testing.T) {
	withADMAuth(t)

	_, err := ListExpiring(ctx(), &ExpiringParams{DzoID: "not-a-uuid"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListExpiring_HRWrongDzoDenied(t *testing.T) {
	_, dzoA, _ := makeCertOrgData(t, "EMP")
	_, dzoB, _ := makeCertOrgData(t, "EMP")

	withCertAuth(t, authhandler.RoleHR, "", dzoA.String())

	_, err := ListExpiring(ctx(), &ExpiringParams{DzoID: dzoB.String(), Days: 180})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestListExpiring_ReturnsOnlyExpiringCertificatesInDzo(t *testing.T) {
	empID, dzoID, _ := makeCertOrgData(t, "EMP")
	otherEmpID, _, _ := makeCertOrgData(t, "EMP")

	soon := time.Now().AddDate(0, 0, 10)
	later := time.Now().AddDate(0, 0, 200)
	past := time.Now().AddDate(0, 0, -10)

	expiring := makeCertForEmployee(t, empID, "Expiring Soon", &soon)
	tooLate := makeCertForEmployee(t, empID, "Too Late", &later)
	expired := makeCertForEmployee(t, empID, "Already Expired", &past)
	otherDzo := makeCertForEmployee(t, otherEmpID, "Other DZO", &soon)

	withADMAuth(t)
	resp, err := ListExpiring(ctx(), &ExpiringParams{DzoID: dzoID.String(), Days: 180})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundExpiring := false
	for _, c := range resp.Certificates {
		switch c.ID {
		case expiring.ID:
			foundExpiring = true
		case tooLate.ID:
			t.Fatal("certificate outside days threshold should not be returned")
		case expired.ID:
			t.Fatal("expired certificate should not be returned")
		case otherDzo.ID:
			t.Fatal("certificate from another DZO should not be returned")
		}
	}
	if !foundExpiring {
		t.Fatal("expected expiring certificate to be returned")
	}
}

// ════ HR CONTACT ════

func TestMyHRContact_NoDzoReturnsEmpty(t *testing.T) {
	withCertAuth(t, authhandler.RoleSA, "", "")

	resp, err := MyHRContact(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Contacts) != 0 {
		t.Fatalf("expected 0 contacts, got %d", len(resp.Contacts))
	}
}

func TestMyHRContact_ReturnsActiveHRContactsInCallerDzo(t *testing.T) {
	_, dzoID, _ := makeCertOrgData(t, "EMP")

	clientRow, err := Client.Company.Create().
		SetName("HR Client " + uuid.NewString()).
		SetDomain(uuid.NewString() + ".local").
		SetLanguage("ru").
		SetUserLimit(100).
		Save(ctx())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	hrKC := "hr-" + uuid.NewString()
	hrEmail := hrKC + "@example.com"
	hrUser, err := Client.User.Create().
		SetKeycloakUserID(hrKC).
		SetEmail(hrEmail).
		SetRole("HR").
		SetDzoID(dzoID).
		SetClientID(clientRow.ID).
		Save(ctx())
	if err != nil {
		t.Fatalf("create hr user: %v", err)
	}

	hrPhone := "1234"
	_, err = Client.Employee.Create().
		SetClientID(clientRow.ID).
		SetDzoID(dzoID).
		SetFullName("Visible HR").
		SetEmail(hrEmail).
		SetInternalPhone(hrPhone).
		SetUserID(hrUser.ID).
		Save(ctx())
	if err != nil {
		t.Fatalf("create hr employee: %v", err)
	}

	withCertAuth(t, authhandler.RoleEMP, "", dzoID.String())

	resp, err := MyHRContact(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, contact := range resp.Contacts {
		if contact.Email == hrEmail {
			found = true
			if contact.Name != "Visible HR" {
				t.Errorf("expected HR name Visible HR, got %q", contact.Name)
			}
			if contact.Phone == nil || *contact.Phone != hrPhone {
				t.Errorf("expected HR phone %q, got %v", hrPhone, contact.Phone)
			}
		}
	}
	if !found {
		t.Fatal("expected HR contact to be returned")
	}
}
