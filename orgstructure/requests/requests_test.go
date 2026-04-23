package requests

import (
	"context"
	"strings"
	"testing"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/user"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	encoreuuid "encore.dev/types/uuid"
	"github.com/google/uuid"
)

func ctx() context.Context {
	return context.Background()
}

func toEncoreUUID(id uuid.UUID) encoreuuid.UUID {
	return encoreuuid.UUID(id)
}

func authCtxFor(id uuid.UUID, role authhandler.UserRole) context.Context {
	keycloakID := strings.ToLower(string(role)) + "-" + id.String()
	return auth.WithContext(context.Background(), auth.UID(keycloakID), &authhandler.AuthData{
		KeycloakUserID: keycloakID,
		Role:           role,
	})
}

// --- helpers ---

func ensureUser(t *testing.T, id uuid.UUID, role string) {
	t.Helper()

	_, err := Client.User.
		Query().
		Where(user.IDEQ(id)).
		Only(ctx())

	if err == nil {
		return
	}
	if !ent.IsNotFound(err) {
		t.Fatalf("query user failed: %v", err)
	}

	unique := id.String()

	_, err = Client.User.
		Create().
		SetID(id).
		SetRole(role).
		SetEmail(strings.ToLower(role) + "-" + unique + "@test.com").
		SetKeycloakUserID(strings.ToLower(role) + "-" + unique).
		SetIsActive(true).
		SetIsOnboarded(true).
		Save(ctx())
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
}

func makeActors(t *testing.T) (uuid.UUID, uuid.UUID) {
	hrID := uuid.New()
	admID := uuid.New()

	ensureUser(t, hrID, "HR")
	ensureUser(t, admID, "ADM")

	return hrID, admID
}

func ensureCompanyAndDZO(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()

	companyID := uuid.New()

	_, err := Client.Company.
		Create().
		SetID(companyID).
		SetName("Test Client " + companyID.String()).
		Save(ctx())
	if err != nil && !ent.IsConstraintError(err) {
		t.Fatalf("create company failed: %v", err)
	}

	return companyID, ensureDZO(t, companyID)
}

func ensureDZO(t *testing.T, companyID uuid.UUID) uuid.UUID {
	t.Helper()

	dzoID := uuid.New()
	_, err := Client.DzoOrganization.
		Create().
		SetID(dzoID).
		SetClientID(companyID).
		SetName("Test DZO " + dzoID.String()).
		Save(ctx())
	if err != nil && !ent.IsConstraintError(err) {
		t.Fatalf("create dzo failed: %v", err)
	}

	return dzoID
}

func ensureEmployee(t *testing.T, companyID, dzoID uuid.UUID, fullName string) uuid.UUID {
	t.Helper()

	employeeID := uuid.New()
	_, err := Client.Employee.
		Create().
		SetID(employeeID).
		SetClientID(companyID).
		SetDzoID(dzoID).
		SetFullName(fullName).
		SetEmail(strings.ToLower(strings.ReplaceAll(fullName, " ", ".")) + "-" + employeeID.String() + "@test.com").
		Save(ctx())
	if err != nil {
		t.Fatalf("create employee failed: %v", err)
	}

	return employeeID
}

func makeRequest(t *testing.T, initiator uuid.UUID) *RequestResponse {
	t.Helper()

	resp, err := CreateRequest(authCtxFor(initiator, authhandler.RoleHR), &CreateRequestRequest{
		EntityID:   uuid.New(),
		EntityType: "TRAINING_EVENT",
	})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}

	return resp
}

// --- tests ---

func TestCreateRequest(t *testing.T) {
	hrID, _ := makeActors(t)

	resp, err := CreateRequest(authCtxFor(hrID, authhandler.RoleHR), &CreateRequestRequest{
		EntityID:   uuid.New(),
		EntityType: "TRAINING_EVENT",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Step != 0 {
		t.Fatalf("expected step 0, got %d", resp.Step)
	}
	if resp.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %s", resp.Status)
	}
}

func TestGetRequest(t *testing.T) {
	hrID, _ := makeActors(t)
	r := makeRequest(t, hrID)

	resp, err := GetRequest(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != r.ID {
		t.Fatalf("id mismatch")
	}
}

func TestInvalidStepJump(t *testing.T) {
	hrID, _ := makeActors(t)
	r := makeRequest(t, hrID)

	_, err := UpdateRequestStep(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		Step: 2,
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestStepFlow(t *testing.T) {
	hrID, admID := makeActors(t)
	r := makeRequest(t, hrID)

	var err error

	r, err = UpdateRequestStep(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		Step: 1,
	})
	if err != nil {
		t.Fatalf("step1 failed: %v", err)
	}

	r, err = UpdateRequestStep(authCtxFor(admID, authhandler.RoleADM), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		Step: 2,
	})
	if err != nil {
		t.Fatalf("step2 failed: %v", err)
	}

	r, err = UpdateRequestStep(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		Step: 3,
	})
	if err != nil {
		t.Fatalf("step3 failed: %v", err)
	}

	if r.Step != 3 {
		t.Fatalf("expected step 3")
	}
}

func TestFinalize(t *testing.T) {
	hrID, admID := makeActors(t)
	r := makeRequest(t, hrID)

	var err error

	r, _ = UpdateRequestStep(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 1})
	r, _ = UpdateRequestStep(authCtxFor(admID, authhandler.RoleADM), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 2})
	r, _ = UpdateRequestStep(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 3})

	final, err := UpdateRequestStatus(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(r.ID), &UpdateRequestStatusRequest{
		Status: "APPROVED",
	})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if final.Status != "APPROVED" {
		t.Fatalf("expected APPROVED")
	}
}

func TestCreateArchiveRequest(t *testing.T) {
	_, admID := makeActors(t)
	companyID, dzoA := ensureCompanyAndDZO(t)
	dzoB := ensureDZO(t, companyID)
	employeeA := ensureEmployee(t, companyID, dzoA, "Archive Employee A")
	employeeB := ensureEmployee(t, companyID, dzoB, "Archive Employee B")

	title := "Archived external learning"
	resp, err := CreateArchiveRequest(authCtxFor(admID, authhandler.RoleADM), &CreateArchiveRequestRequest{
		Kind:        string(RequestKindArchived),
		Title:       &title,
		Category:    "external_learning",
		EmployeeIDs: []uuid.UUID{employeeA, employeeB},
		Contracts: []ArchiveRequestContractInput{
			{DzoID: dzoA, FileName: "contract-a.pdf", FileURL: "s3://contracts/a.pdf"},
			{DzoID: dzoB, FileName: "contract-b.pdf", FileURL: "s3://contracts/b.pdf"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Kind != string(RequestKindArchived) {
		t.Fatalf("expected archived kind, got %s", resp.Kind)
	}
	if resp.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", resp.Status)
	}
	if resp.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
	if len(resp.EmployeeIDs) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(resp.EmployeeIDs))
	}
	if len(resp.Contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(resp.Contracts))
	}
}

func TestCreateArchiveRequestRequiresAdmin(t *testing.T) {
	hrID, _ := makeActors(t)
	companyID, dzoID := ensureCompanyAndDZO(t)
	employeeID := ensureEmployee(t, companyID, dzoID, "Archive Employee HR")

	_, err := CreateArchiveRequest(authCtxFor(hrID, authhandler.RoleHR), &CreateArchiveRequestRequest{
		Kind:        string(RequestKindClosed),
		Category:    "manual_expense",
		EmployeeIDs: []uuid.UUID{employeeID},
		Contracts: []ArchiveRequestContractInput{
			{DzoID: dzoID, FileName: "contract.pdf", FileURL: "s3://contracts/contract.pdf"},
		},
	})
	if err == nil {
		t.Fatal("expected permission error")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestCreateArchiveRequestRequiresContractPerDZO(t *testing.T) {
	_, admID := makeActors(t)
	companyID, dzoA := ensureCompanyAndDZO(t)
	dzoB := ensureDZO(t, companyID)
	employeeA := ensureEmployee(t, companyID, dzoA, "Archive Employee One")
	employeeB := ensureEmployee(t, companyID, dzoB, "Archive Employee Two")

	_, err := CreateArchiveRequest(authCtxFor(admID, authhandler.RoleADM), &CreateArchiveRequestRequest{
		Kind:        string(RequestKindClosed),
		Category:    "manual_expense",
		EmployeeIDs: []uuid.UUID{employeeA, employeeB},
		Contracts: []ArchiveRequestContractInput{
			{DzoID: dzoA, FileName: "contract-a.pdf", FileURL: "s3://contracts/a.pdf"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestArchiveRequestHiddenFromHR(t *testing.T) {
	hrID, admID := makeActors(t)
	companyID, dzoID := ensureCompanyAndDZO(t)
	employeeID := ensureEmployee(t, companyID, dzoID, "Archive Employee Hidden")

	archiveResp, err := CreateArchiveRequest(authCtxFor(admID, authhandler.RoleADM), &CreateArchiveRequestRequest{
		Kind:        string(RequestKindArchived),
		Category:    "manual_expense",
		EmployeeIDs: []uuid.UUID{employeeID},
		Contracts: []ArchiveRequestContractInput{
			{DzoID: dzoID, FileName: "contract.pdf", FileURL: "s3://contracts/contract.pdf"},
		},
	})
	if err != nil {
		t.Fatalf("create archive failed: %v", err)
	}

	listResp, err := ListRequests(authCtxFor(hrID, authhandler.RoleHR))
	if err != nil {
		t.Fatalf("list requests failed: %v", err)
	}
	for _, item := range listResp.Items {
		if item.ID == archiveResp.ID {
			t.Fatal("hr should not see archive request in list")
		}
	}

	_, err = GetRequest(authCtxFor(hrID, authhandler.RoleHR), toEncoreUUID(archiveResp.ID))
	if err == nil {
		t.Fatal("expected permission error when hr opens archive request")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", errs.Code(err))
	}
}
