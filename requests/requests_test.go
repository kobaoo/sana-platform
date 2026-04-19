package requests

import (
	"context"
	"strings"
	"testing"

	"encore.app/db/ent"
	"encore.app/db/ent/user"
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

func makeRequest(t *testing.T, initiator uuid.UUID) *RequestResponse {
	t.Helper()

	resp, err := CreateRequest(ctx(), &CreateRequestRequest{
		InitiatorID: initiator,
		EntityID:    uuid.New(),
		EntityType:  "TRAINING_EVENT",
	})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}

	return resp
}

// --- tests ---

func TestCreateRequest(t *testing.T) {
	hrID, _ := makeActors(t)

	resp, err := CreateRequest(ctx(), &CreateRequestRequest{
		InitiatorID: hrID,
		EntityID:    uuid.New(),
		EntityType:  "TRAINING_EVENT",
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

	resp, err := GetRequest(ctx(), toEncoreUUID(r.ID))
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

	_, err := UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		ActorID: hrID,
		Step:    2,
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

	r, err = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		ActorID: hrID,
		Step:    1,
	})
	if err != nil {
		t.Fatalf("step1 failed: %v", err)
	}

	r, err = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		ActorID: admID,
		Step:    2,
	})
	if err != nil {
		t.Fatalf("step2 failed: %v", err)
	}

	r, err = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{
		ActorID: hrID,
		Step:    3,
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

	r, _ = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{ActorID: hrID, Step: 1})
	r, _ = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{ActorID: admID, Step: 2})
	r, _ = UpdateRequestStep(ctx(), toEncoreUUID(r.ID), &UpdateRequestStepRequest{ActorID: hrID, Step: 3})

	final, err := UpdateRequestStatus(ctx(), toEncoreUUID(r.ID), &UpdateRequestStatusRequest{
		ActorID: hrID,
		Status:  "APPROVED",
	})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if final.Status != "APPROVED" {
		t.Fatalf("expected APPROVED")
	}
}
