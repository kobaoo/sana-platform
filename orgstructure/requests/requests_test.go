package requests

import (
	"context"
	"testing"
	"time"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	encoreuuid "encore.dev/types/uuid"
	"github.com/google/uuid"
)

func ctx() context.Context {
	return context.Background()
}

func authCtxFor(keycloakUserID string, role authhandler.UserRole, dzoID *uuid.UUID) context.Context {
	ad := &authhandler.AuthData{
		KeycloakUserID: keycloakUserID,
		Role:           role,
	}
	if dzoID != nil {
		ad.DzoID = dzoID.String()
	}
	return auth.WithContext(context.Background(), auth.UID(keycloakUserID), ad)
}

func toEncoreUUID(id uuid.UUID) encoreuuid.UUID {
	return encoreuuid.UUID(id)
}

type requestTestFixture struct {
	clientID        string
	adminID         uuid.UUID
	adminKC         string
	hrOneID         uuid.UUID
	hrOneKC         string
	hrTwoID         uuid.UUID
	hrTwoKC         string
	hrThreeID       uuid.UUID
	hrThreeKC       string
	dzoOneID        uuid.UUID
	dzoTwoID        uuid.UUID
	dzoThreeID      uuid.UUID
	empOneID        uuid.UUID
	empTwoID        uuid.UUID
	trainingEventID uuid.UUID
}

func newFixture(t *testing.T) *requestTestFixture {
	t.Helper()

	fx := &requestTestFixture{
		clientID:        uuid.NewString(),
		adminID:         uuid.New(),
		adminKC:         "kc-admin-" + uuid.NewString(),
		hrOneID:         uuid.New(),
		hrOneKC:         "kc-hr-one-" + uuid.NewString(),
		hrTwoID:         uuid.New(),
		hrTwoKC:         "kc-hr-two-" + uuid.NewString(),
		hrThreeID:       uuid.New(),
		hrThreeKC:       "kc-hr-three-" + uuid.NewString(),
		dzoOneID:        uuid.New(),
		dzoTwoID:        uuid.New(),
		dzoThreeID:      uuid.New(),
		empOneID:        uuid.New(),
		empTwoID:        uuid.New(),
		trainingEventID: uuid.New(),
	}

	ensureClient(t, fx.clientID)
	ensureDZO(t, fx.dzoOneID, fx.clientID, "DZO One")
	ensureDZO(t, fx.dzoTwoID, fx.clientID, "DZO Two")
	ensureDZO(t, fx.dzoThreeID, fx.clientID, "DZO Three")
	ensureUser(t, fx.adminID, fx.adminKC, "admin@test.local", "ADM", fx.clientID, nil)
	ensureUser(t, fx.hrOneID, fx.hrOneKC, "hr1@test.local", "HR", fx.clientID, &fx.dzoOneID)
	ensureUser(t, fx.hrTwoID, fx.hrTwoKC, "hr2@test.local", "HR", fx.clientID, &fx.dzoTwoID)
	ensureUser(t, fx.hrThreeID, fx.hrThreeKC, "hr3@test.local", "HR", fx.clientID, &fx.dzoThreeID)
	ensureEmployee(t, fx.empOneID, fx.clientID, fx.dzoOneID, "Employee One", "emp1@test.local")
	ensureEmployee(t, fx.empTwoID, fx.clientID, fx.dzoTwoID, "Employee Two", "emp2@test.local")
	ensureTrainingEvent(t, fx.trainingEventID, fx.dzoOneID, "External Learning")

	return fx
}

func ensureClient(t *testing.T, clientID string) {
	t.Helper()

	_, err := db.Exec(ctx(), `
		INSERT INTO clients (id, name, created_at, is_active)
		VALUES ($1, $2, NOW(), TRUE)
		ON CONFLICT (id) DO NOTHING
	`, mustUUID(t, clientID), "Client "+clientID[:8])
	if err != nil {
		t.Fatalf("insert client: %v", err)
	}
}

func ensureDZO(t *testing.T, dzoID uuid.UUID, clientID, name string) {
	t.Helper()

	_, err := db.Exec(ctx(), `
		INSERT INTO dzo_organizations (id, client_id, name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, dzoID, mustUUID(t, clientID), name)
	if err != nil {
		t.Fatalf("insert dzo: %v", err)
	}
}

func ensureUser(t *testing.T, userID uuid.UUID, keycloakID, email, role, clientID string, dzoID *uuid.UUID) {
	t.Helper()

	_, err := db.Exec(ctx(), `
		INSERT INTO users (id, keycloak_user_id, email, role, dzo_id, is_active, is_onboarded, created_at, updated_at, client_id)
		VALUES ($1, $2, $3, $4, $5, TRUE, TRUE, NOW(), NOW(), $6)
		ON CONFLICT (id) DO NOTHING
	`, userID, keycloakID, email, role, nullableUUIDForTest(dzoID), mustUUID(t, clientID))
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func ensureEmployee(t *testing.T, employeeID uuid.UUID, clientID string, dzoID uuid.UUID, fullName, email string) {
	t.Helper()

	_, err := db.Exec(ctx(), `
		INSERT INTO employees (id, client_id, dzo_id, full_name, email, is_active, is_deleted)
		VALUES ($1, $2, $3, $4, $5, TRUE, FALSE)
		ON CONFLICT (id) DO NOTHING
	`, employeeID, mustUUID(t, clientID), dzoID, fullName, email)
	if err != nil {
		t.Fatalf("insert employee: %v", err)
	}
}

func ensureTrainingEvent(t *testing.T, eventID, dzoID uuid.UUID, title string) {
	t.Helper()

	_, err := db.Exec(ctx(), `
		INSERT INTO training_events (
			id, title, start_date, end_date, location_type, category_id, dzo_id, participants_count
		)
		VALUES ($1, $2, NOW(), NOW(), 'offline', $3, $4, 0)
		ON CONFLICT (id) DO NOTHING
	`, eventID, title, uuid.New(), dzoID)
	if err != nil {
		t.Fatalf("insert training event: %v", err)
	}
}

func mustUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("parse uuid: %v", err)
	}
	return id
}

func nullableUUIDForTest(id *uuid.UUID) interface{} {
	if id == nil {
		return nil
	}
	return *id
}

func makeDraftRequest(t *testing.T, fx *requestTestFixture, employeeIDs []uuid.UUID, dzoIDs []uuid.UUID) *RequestDetail {
	t.Helper()

	employeeStrings := make([]string, 0, len(employeeIDs))
	for _, id := range employeeIDs {
		employeeStrings = append(employeeStrings, id.String())
	}
	dzoStrings := make([]string, 0, len(dzoIDs))
	for _, id := range dzoIDs {
		dzoStrings = append(dzoStrings, id.String())
	}

	deadline := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	costMode := CostModePerEmployee
	cost := 5000.0
	resp, err := CreateAdminRequest(authCtxFor(fx.adminKC, authhandler.RoleADM, nil), &CreateAdminRequestRequest{
		TrainingEventID: fx.trainingEventID.String(),
		EmployeeIDs:     employeeStrings,
		DzoIDs:          dzoStrings,
		CostAmount:      &cost,
		CostMode:        &costMode,
		DeadlineAt:      &deadline,
	})
	if err != nil {
		t.Fatalf("create admin request: %v", err)
	}

	return &resp.Detail
}

func submitDraftRequest(t *testing.T, fx *requestTestFixture, detail *RequestDetail) *RequestDetail {
	t.Helper()

	resp, err := SubmitRequest(authCtxFor(fx.adminKC, authhandler.RoleADM, nil), toEncoreUUID(uuid.MustParse(detail.Request.ID)))
	if err != nil {
		t.Fatalf("submit request: %v", err)
	}
	return &resp.Detail
}

func TestCreateAdminRequest_Success(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, []uuid.UUID{fx.empOneID}, []uuid.UUID{fx.dzoTwoID})

	if detail.Request.RequestType != RequestTypeMain {
		t.Fatalf("expected MAIN request, got %s", detail.Request.RequestType)
	}
	if detail.Request.Status != RequestStatusDraft {
		t.Fatalf("expected DRAFT status, got %s", detail.Request.Status)
	}
	if detail.Request.EmployeesCount != 1 {
		t.Fatalf("expected 1 employee, got %d", detail.Request.EmployeesCount)
	}
	if len(detail.TargetDZOs) != 1 || detail.TargetDZOs[0].ID != fx.dzoTwoID.String() {
		t.Fatalf("expected target dzo %s", fx.dzoTwoID)
	}
}

func TestSubmitRequest_SplitsByHRAndDZO(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, []uuid.UUID{fx.empOneID}, []uuid.UUID{fx.dzoTwoID})
	submitted := submitDraftRequest(t, fx, detail)

	if submitted.Request.Status != RequestStatusInProgress {
		t.Fatalf("expected parent status IN_PROGRESS, got %s", submitted.Request.Status)
	}
	if len(submitted.ChildRequests) != 2 {
		t.Fatalf("expected 2 subrequests, got %d", len(submitted.ChildRequests))
	}

	foundDZOOne := false
	foundDZOTwo := false
	for _, child := range submitted.ChildRequests {
		if child.TargetDzoID != nil && *child.TargetDzoID == fx.dzoOneID.String() {
			foundDZOOne = true
			if child.AssignedHRID == nil || *child.AssignedHRID != fx.hrOneID.String() {
				t.Fatalf("expected hr one assigned to dzo one")
			}
			if child.EmployeesCount != 1 {
				t.Fatalf("expected 1 employee for dzo one child, got %d", child.EmployeesCount)
			}
		}
		if child.TargetDzoID != nil && *child.TargetDzoID == fx.dzoTwoID.String() {
			foundDZOTwo = true
			if child.AssignedHRID == nil || *child.AssignedHRID != fx.hrTwoID.String() {
				t.Fatalf("expected hr two assigned to dzo two")
			}
			if child.EmployeesCount != 0 {
				t.Fatalf("expected 0 employees for dzo two child, got %d", child.EmployeesCount)
			}
		}
	}

	if !foundDZOOne || !foundDZOTwo {
		t.Fatal("expected subrequests for both selected dzo groups")
	}
}

func TestListHRRequests_ReturnsOnlyAssigned(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, []uuid.UUID{fx.empOneID, fx.empTwoID}, nil)
	_ = submitDraftRequest(t, fx, detail)

	hrOneResp, err := ListHRRequests(authCtxFor(fx.hrOneKC, authhandler.RoleHR, &fx.dzoOneID))
	if err != nil {
		t.Fatalf("list hr requests for hr one: %v", err)
	}
	if len(hrOneResp.Items) != 1 {
		t.Fatalf("expected 1 request for hr one, got %d", len(hrOneResp.Items))
	}
	if hrOneResp.Items[0].AssignedHRID == nil || *hrOneResp.Items[0].AssignedHRID != fx.hrOneID.String() {
		t.Fatalf("unexpected assigned hr in hr one list")
	}

	hrTwoResp, err := ListHRRequests(authCtxFor(fx.hrTwoKC, authhandler.RoleHR, &fx.dzoTwoID))
	if err != nil {
		t.Fatalf("list hr requests for hr two: %v", err)
	}
	if len(hrTwoResp.Items) != 1 {
		t.Fatalf("expected 1 request for hr two, got %d", len(hrTwoResp.Items))
	}
}

func TestApproveAndCancelSubrequests_UpdateParentProgress(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, []uuid.UUID{fx.empOneID, fx.empTwoID}, nil)
	submitted := submitDraftRequest(t, fx, detail)

	var hrOneRequestID, hrTwoRequestID string
	for _, child := range submitted.ChildRequests {
		if child.TargetDzoID != nil && *child.TargetDzoID == fx.dzoOneID.String() {
			hrOneRequestID = child.ID
		}
		if child.TargetDzoID != nil && *child.TargetDzoID == fx.dzoTwoID.String() {
			hrTwoRequestID = child.ID
		}
	}

	if _, err := ApproveHRRequest(authCtxFor(fx.hrOneKC, authhandler.RoleHR, &fx.dzoOneID), toEncoreUUID(uuid.MustParse(hrOneRequestID))); err != nil {
		t.Fatalf("approve hr one request: %v", err)
	}
	if _, err := CancelHRRequest(authCtxFor(fx.hrTwoKC, authhandler.RoleHR, &fx.dzoTwoID), toEncoreUUID(uuid.MustParse(hrTwoRequestID))); err != nil {
		t.Fatalf("cancel hr two request: %v", err)
	}

	parent, err := GetRequest(authCtxFor(fx.adminKC, authhandler.RoleADM, nil), toEncoreUUID(uuid.MustParse(detail.Request.ID)))
	if err != nil {
		t.Fatalf("get parent request: %v", err)
	}
	if parent.Detail.Request.Status != RequestStatusApproved {
		t.Fatalf("expected parent status APPROVED after mixed result, got %s", parent.Detail.Request.Status)
	}
	if parent.Detail.Request.ApprovedChildren != 1 || parent.Detail.Request.TotalChildren != 2 {
		t.Fatalf("expected progress 1/2, got %d/%d", parent.Detail.Request.ApprovedChildren, parent.Detail.Request.TotalChildren)
	}
}

func TestApproveEmptySubrequest_FailsUntilHRSetsEmployees(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, nil, []uuid.UUID{fx.dzoThreeID})
	submitted := submitDraftRequest(t, fx, detail)
	if len(submitted.ChildRequests) != 1 {
		t.Fatalf("expected 1 child request, got %d", len(submitted.ChildRequests))
	}

	childID := uuid.MustParse(submitted.ChildRequests[0].ID)
	_, err := ApproveHRRequest(authCtxFor(fx.hrThreeKC, authhandler.RoleHR, &fx.dzoThreeID), toEncoreUUID(childID))
	if err == nil {
		t.Fatal("expected error when approving empty subrequest")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateHRRequestEmployees_ThenApprove(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, nil, []uuid.UUID{fx.dzoTwoID})
	submitted := submitDraftRequest(t, fx, detail)
	childID := uuid.MustParse(submitted.ChildRequests[0].ID)

	updated, err := UpdateHRRequestEmployees(authCtxFor(fx.hrTwoKC, authhandler.RoleHR, &fx.dzoTwoID), toEncoreUUID(childID), &UpdateHRRequestEmployeesRequest{
		EmployeeIDs: []string{fx.empTwoID.String()},
	})
	if err != nil {
		t.Fatalf("update hr employees: %v", err)
	}
	if len(updated.Detail.Employees) != 1 {
		t.Fatalf("expected 1 employee after update, got %d", len(updated.Detail.Employees))
	}

	approved, err := ApproveHRRequest(authCtxFor(fx.hrTwoKC, authhandler.RoleHR, &fx.dzoTwoID), toEncoreUUID(childID))
	if err != nil {
		t.Fatalf("approve hr request: %v", err)
	}
	if approved.Detail.Request.Status != RequestStatusApproved {
		t.Fatalf("expected APPROVED status, got %s", approved.Detail.Request.Status)
	}
}

func TestCreateAdminRequest_RequiresHRCoverage(t *testing.T) {
	fx := newFixture(t)

	noHRDZO := uuid.New()
	ensureDZO(t, noHRDZO, fx.clientID, "No HR DZO")

	_, err := CreateAdminRequest(authCtxFor(fx.adminKC, authhandler.RoleADM, nil), &CreateAdminRequestRequest{
		TrainingEventID: fx.trainingEventID.String(),
		DzoIDs:          []string{noHRDZO.String()},
	})
	if err == nil {
		t.Fatal("expected error when DZO has no HR")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestGetHRRequest_HidesForeignSubrequest(t *testing.T) {
	fx := newFixture(t)

	detail := makeDraftRequest(t, fx, []uuid.UUID{fx.empOneID, fx.empTwoID}, nil)
	submitted := submitDraftRequest(t, fx, detail)

	childID := uuid.MustParse(submitted.ChildRequests[0].ID)
	_, err := GetHRRequest(authCtxFor(fx.hrThreeKC, authhandler.RoleHR, &fx.dzoThreeID), toEncoreUUID(childID))
	if err == nil {
		t.Fatal("expected not found for foreign hr request")
	}
	if errs.Code(err) != errs.NotFound {
		t.Fatalf("expected NotFound, got %v", errs.Code(err))
	}
}
