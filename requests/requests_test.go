package requests

import (
	"context"
	"strings"
	"testing"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/et"
	encoreuuid "encore.dev/types/uuid"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/user"
)

func ctx() context.Context {
	return context.Background()
}

func toEncoreUUID(id uuid.UUID) encoreuuid.UUID {
	return encoreuuid.UUID(id)
}

// ensureUser creates a DB user without injecting auth context.
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

// makeRequest creates a TRAINING_EVENT request using the current auth context.
func makeRequest(t *testing.T) *RequestResponse {
	t.Helper()

	resp, err := CreateRequest(ctx(), &CreateRequestRequest{
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
	makeAuthUser(t, authhandler.RoleHR)

	resp, err := CreateRequest(ctx(), &CreateRequestRequest{
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
	makeAuthUser(t, authhandler.RoleHR)
	r := makeRequest(t)

	resp, err := GetRequest(ctx(), toEncoreUUID(r.ID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != r.ID {
		t.Fatalf("id mismatch")
	}
}

func TestInvalidStepJump(t *testing.T) {
	makeAuthUser(t, authhandler.RoleHR)
	r := makeRequest(t)

	_, err := UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
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
	_, hrKcID := makeAuthUser(t, authhandler.RoleHR)
	_, admKcID := makeAuthUser(t, authhandler.RoleADM)

	et.OverrideAuthInfo(auth.UID(hrKcID), &authhandler.AuthData{
		KeycloakUserID: hrKcID,
		Role:           authhandler.RoleHR,
	})
	r := makeRequest(t)

	var err error

	// Step 1: HR
	r, err = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 1})
	if err != nil {
		t.Fatalf("step1 failed: %v", err)
	}

	// Step 2: ADM
	et.OverrideAuthInfo(auth.UID(admKcID), &authhandler.AuthData{
		KeycloakUserID: admKcID,
		Role:           authhandler.RoleADM,
	})
	r, err = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 2})
	if err != nil {
		t.Fatalf("step2 failed: %v", err)
	}

	// Step 3: HR
	et.OverrideAuthInfo(auth.UID(hrKcID), &authhandler.AuthData{
		KeycloakUserID: hrKcID,
		Role:           authhandler.RoleHR,
	})
	r, err = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 3})
	if err != nil {
		t.Fatalf("step3 failed: %v", err)
	}

	if r.Step != 3 {
		t.Fatalf("expected step 3, got %d", r.Step)
	}
}

func TestFinalize(t *testing.T) {
	_, hrKcID := makeAuthUser(t, authhandler.RoleHR)
	_, admKcID := makeAuthUser(t, authhandler.RoleADM)

	et.OverrideAuthInfo(auth.UID(hrKcID), &authhandler.AuthData{
		KeycloakUserID: hrKcID,
		Role:           authhandler.RoleHR,
	})
	r := makeRequest(t)

	var err error

	r, _ = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 1})

	et.OverrideAuthInfo(auth.UID(admKcID), &authhandler.AuthData{
		KeycloakUserID: admKcID,
		Role:           authhandler.RoleADM,
	})
	r, _ = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 2})

	et.OverrideAuthInfo(auth.UID(hrKcID), &authhandler.AuthData{
		KeycloakUserID: hrKcID,
		Role:           authhandler.RoleHR,
	})
	r, _ = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{Step: 3})

	final, err := UpdateRequestStatus(ctx(), toEncoreUUID(r.ID), &UpdateRequestStatusRequest{
		Status: "APPROVED",
	})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if final.Status != "APPROVED" {
		t.Fatalf("expected APPROVED, got %s", final.Status)
	}
}
