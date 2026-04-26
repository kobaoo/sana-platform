// Package events tests.
//
// Imports encore.dev/storage/sqldb — run with `encore test ./courses/events/...`.
package events

import (
	"context"
	"strings"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
)

// ════ FIXTURES ════

// makeClient inserts a fresh clients row and returns its ID.
func makeClient(t *testing.T) uuid.UUID {
	t.Helper()

	row, err := Client.Company.
		Create().
		SetName("Test Co " + uuid.New().String()[:8]).
		Save(context.Background())
	if err != nil {
		t.Fatalf("makeClient: %v", err)
	}
	return row.ID
}

// makeUser inserts a user row with the given role and returns its ID.
func makeUser(t *testing.T, role authhandler.UserRole) uuid.UUID {
	t.Helper()

	id := uuid.New()
	suffix := id.String()[:8]
	_, err := Client.User.
		Create().
		SetID(id).
		SetRole(string(role)).
		SetEmail(strings.ToLower(string(role)) + "-" + suffix + "@test.com").
		SetKeycloakUserID(strings.ToLower(string(role)) + "-" + suffix).
		SetIsActive(true).
		SetIsOnboarded(true).
		Save(context.Background())
	if err != nil {
		t.Fatalf("makeUser: %v", err)
	}
	return id
}

// makeDzo inserts a DZO row bound to the given client and returns its ID.
func makeDzo(t *testing.T, clientID uuid.UUID) uuid.UUID {
	t.Helper()
	suffix := uuid.New().String()[:6]
	row, err := Client.DzoOrganization.
		Create().
		SetClientID(clientID).
		SetName("DZO " + suffix).
		SetIsActive(true).
		Save(context.Background())
	if err != nil {
		t.Fatalf("makeDzo: %v", err)
	}
	return row.ID
}

// makeEmployee inserts an employee bound to the given client/DZO.
func makeEmployee(t *testing.T, clientID, dzoID uuid.UUID) uuid.UUID {
	t.Helper()
	suffix := uuid.New().String()[:8]
	row, err := Client.Employee.
		Create().
		SetClientID(clientID).
		SetDzoID(dzoID).
		SetFullName("Emp " + suffix).
		SetEmail("emp-" + suffix + "@test.com").
		SetIsActive(true).
		Save(context.Background())
	if err != nil {
		t.Fatalf("makeEmployee: %v", err)
	}
	return row.ID
}

// withRole returns a context carrying AuthData for the given role and company.
func withRole(role authhandler.UserRole, companyID uuid.UUID) context.Context {
	uid := strings.ToLower(string(role)) + "-" + uuid.New().String()[:8]
	return auth.WithContext(context.Background(), auth.UID(uid), &authhandler.AuthData{
		KeycloakUserID: uid,
		Role:           role,
		CompanyID:      companyID.String(),
	})
}

// fixture bundles the IDs commonly needed by event tests.
type fixture struct {
	ClientID uuid.UUID
	DzoID    uuid.UUID
	HostID   uuid.UUID
	AdminCtx context.Context
	HRCtx    context.Context
	EmpCtx   context.Context
}

func setup(t *testing.T) fixture {
	t.Helper()
	clientID := makeClient(t)
	hostID := makeUser(t, authhandler.RoleHR)
	dzoID := makeDzo(t, clientID)
	return fixture{
		ClientID: clientID,
		DzoID:    dzoID,
		HostID:   hostID,
		AdminCtx: withRole(authhandler.RoleADM, clientID),
		HRCtx:    withRole(authhandler.RoleHR, clientID),
		EmpCtx:   withRole(authhandler.RoleEMP, clientID),
	}
}

func validCreateRequest(f fixture) *CreateEventRequest {
	return &CreateEventRequest{
		Title:           "Webinar " + uuid.New().String()[:6],
		EventDate:       time.Now().Add(48 * time.Hour),
		HostID:          f.HostID.String(),
		ZoomLink:        "https://zoom.us/j/123456789",
		MaxParticipants: 10,
	}
}

func makeEvent(t *testing.T, f fixture) *Event {
	t.Helper()
	resp, err := CreateEvent(f.AdminCtx, validCreateRequest(f))
	if err != nil {
		t.Fatalf("makeEvent: %v", err)
	}
	return &resp.Event
}

func makeCancelledEvent(t *testing.T, f fixture) *Event {
	t.Helper()
	ev := makeEvent(t, f)
	if _, err := DeleteEvent(f.AdminCtx, ev.ID); err != nil {
		t.Fatalf("makeCancelledEvent: %v", err)
	}
	return ev
}

func emptyListParams() *ListEventsParams {
	return &ListEventsParams{}
}

// ════ CREATE ════

func TestCreateEvent_Success(t *testing.T) {
	f := setup(t)

	resp, err := CreateEvent(f.AdminCtx, validCreateRequest(f))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Event.Status != StatusActive {
		t.Errorf("expected ACTIVE, got %s", resp.Event.Status)
	}
	if resp.Event.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Event.ClientID != f.ClientID.String() {
		t.Errorf("expected client_id %s, got %s", f.ClientID, resp.Event.ClientID)
	}
	if resp.Event.MaxParticipants != 10 {
		t.Errorf("expected max_participants 10, got %d", resp.Event.MaxParticipants)
	}
	if resp.Event.AvailableSlots != 10 {
		t.Errorf("expected 10 available slots, got %d", resp.Event.AvailableSlots)
	}
	if resp.Event.ZoomLink != "https://zoom.us/j/123456789" {
		t.Errorf("expected zoom_link, got %s", resp.Event.ZoomLink)
	}
}

func TestCreateEvent_WithEmployeeIDs(t *testing.T) {
	f := setup(t)
	emp1 := makeEmployee(t, f.ClientID, f.DzoID)
	emp2 := makeEmployee(t, f.ClientID, f.DzoID)

	req := validCreateRequest(f)
	req.EmployeeIDs = []string{emp1.String(), emp2.String()}

	resp, err := CreateEvent(f.AdminCtx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Event.ParticipantsCount != 2 {
		t.Errorf("expected 2 participants, got %d", resp.Event.ParticipantsCount)
	}
	if resp.Event.AvailableSlots != 8 {
		t.Errorf("expected 8 available slots, got %d", resp.Event.AvailableSlots)
	}
	if len(resp.Event.Participants) != 2 {
		t.Errorf("expected 2 participant rows, got %d", len(resp.Event.Participants))
	}
}

func TestCreateEvent_EmployeeIDsDeduped(t *testing.T) {
	f := setup(t)
	emp := makeEmployee(t, f.ClientID, f.DzoID)

	req := validCreateRequest(f)
	req.EmployeeIDs = []string{emp.String(), emp.String()}

	resp, err := CreateEvent(f.AdminCtx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Event.ParticipantsCount != 1 {
		t.Errorf("expected 1 participant after dedupe, got %d", resp.Event.ParticipantsCount)
	}
}

func TestCreateEvent_EmployeeIDsExceedMax(t *testing.T) {
	f := setup(t)
	emps := make([]string, 3)
	for i := range emps {
		emps[i] = makeEmployee(t, f.ClientID, f.DzoID).String()
	}

	req := validCreateRequest(f)
	req.MaxParticipants = 2
	req.EmployeeIDs = emps

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument when employees exceed max, got %v", errs.Code(err))
	}
}

func TestCreateEvent_EmployeeFromAnotherClient(t *testing.T) {
	f := setup(t)
	otherClient := makeClient(t)
	otherDzo := makeDzo(t, otherClient)
	alien := makeEmployee(t, otherClient, otherDzo)

	req := validCreateRequest(f)
	req.EmployeeIDs = []string{alien.String()}

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument for foreign employee, got %v", errs.Code(err))
	}
}

func TestCreateEvent_MissingTitle(t *testing.T) {
	f := setup(t)

	req := validCreateRequest(f)
	req.Title = ""

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_MissingZoomLink(t *testing.T) {
	f := setup(t)

	req := validCreateRequest(f)
	req.ZoomLink = ""

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument for missing zoom_link, got %v", errs.Code(err))
	}
}

func TestCreateEvent_MissingMaxParticipants(t *testing.T) {
	f := setup(t)

	req := validCreateRequest(f)
	req.MaxParticipants = 0

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument for missing max_participants, got %v", errs.Code(err))
	}
}

func TestCreateEvent_MissingDate(t *testing.T) {
	f := setup(t)

	req := validCreateRequest(f)
	req.EventDate = time.Time{}

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_PastDateRejected(t *testing.T) {
	f := setup(t)

	req := validCreateRequest(f)
	req.EventDate = time.Now().Add(-24 * time.Hour)

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_PermissionDenied(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.EmpCtx, validCreateRequest(f))
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestCreateEvent_InvalidHost(t *testing.T) {
	f := setup(t)

	req := validCreateRequest(f)
	req.HostID = uuid.New().String()

	_, err := CreateEvent(f.AdminCtx, req)
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument for unknown host, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestListEvents_AdminSeesActiveAndCancelled(t *testing.T) {
	f := setup(t)
	makeEvent(t, f)          // ACTIVE
	makeCancelledEvent(t, f) // CANCELLED

	resp, err := ListEvents(f.AdminCtx, emptyListParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total < 2 {
		t.Errorf("expected at least 2 events, got %d", resp.Total)
	}

	var hasActive, hasCancelled bool
	for _, ev := range resp.Events {
		switch ev.Status {
		case StatusActive:
			hasActive = true
		case StatusCancelled:
			hasCancelled = true
		}
	}
	if !hasActive {
		t.Error("expected at least one ACTIVE event in admin list")
	}
	if !hasCancelled {
		t.Error("expected at least one CANCELLED event in admin list")
	}
}

func TestListEvents_EmployeeSeesActiveOnly(t *testing.T) {
	f := setup(t)
	cancelled := makeCancelledEvent(t, f)
	active := makeEvent(t, f)

	resp, err := ListEvents(f.EmpCtx, emptyListParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, ev := range resp.Events {
		if ev.Status != StatusActive {
			t.Errorf("non-active event %s leaked: %s", ev.ID, ev.Status)
		}
		if ev.ID == cancelled.ID {
			t.Errorf("cancelled event %s leaked into employee feed", cancelled.ID)
		}
	}

	found := false
	for _, ev := range resp.Events {
		if ev.ID == active.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected active event %s in employee list", active.ID)
	}
}

func TestListEvents_DateRangeFilter(t *testing.T) {
	f := setup(t)

	// event today
	req := validCreateRequest(f)
	req.EventDate = time.Now().Add(2 * time.Hour)
	respToday, err := CreateEvent(f.AdminCtx, req)
	if err != nil {
		t.Fatalf("create today: %v", err)
	}

	// event in 10 days
	req2 := validCreateRequest(f)
	req2.EventDate = time.Now().Add(10 * 24 * time.Hour)
	respFuture, err := CreateEvent(f.AdminCtx, req2)
	if err != nil {
		t.Fatalf("create future: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	resp, err := ListEvents(f.AdminCtx, &ListEventsParams{From: today, To: today})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	foundToday, foundFuture := false, false
	for _, ev := range resp.Events {
		if ev.ID == respToday.Event.ID {
			foundToday = true
		}
		if ev.ID == respFuture.Event.ID {
			foundFuture = true
		}
	}
	if !foundToday {
		t.Error("expected today's event in date-filtered list")
	}
	if foundFuture {
		t.Error("did not expect future event in today's filtered list")
	}
}

func TestListEvents_StatusFilter(t *testing.T) {
	f := setup(t)
	makeEvent(t, f)
	cancelled := makeCancelledEvent(t, f)

	resp, err := ListEvents(f.AdminCtx, &ListEventsParams{Status: "CANCELLED"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ev := range resp.Events {
		if ev.Status != StatusCancelled {
			t.Errorf("non-cancelled event leaked into CANCELLED filter: %s", ev.Status)
		}
	}
	found := false
	for _, ev := range resp.Events {
		if ev.ID == cancelled.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected cancelled event in filtered list")
	}
}

// ════ GET ════

func TestGetEvent_AdminSuccess(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	resp, err := GetEvent(f.AdminCtx, ev.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Event.ID != ev.ID {
		t.Errorf("wrong id")
	}
}

func TestGetEvent_IncludesParticipants(t *testing.T) {
	f := setup(t)
	emp := makeEmployee(t, f.ClientID, f.DzoID)

	req := validCreateRequest(f)
	req.EmployeeIDs = []string{emp.String()}
	created, err := CreateEvent(f.AdminCtx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := GetEvent(f.AdminCtx, created.Event.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.Event.ParticipantsCount != 1 {
		t.Errorf("expected 1 participant, got %d", resp.Event.ParticipantsCount)
	}
	if len(resp.Event.Participants) != 1 {
		t.Fatalf("expected 1 participant row, got %d", len(resp.Event.Participants))
	}
	if resp.Event.Participants[0].EmployeeID != emp.String() {
		t.Errorf("wrong employee in participants")
	}
}

func TestGetEvent_EmployeeCannotSeeCancelled(t *testing.T) {
	f := setup(t)
	cancelled := makeCancelledEvent(t, f)

	_, err := GetEvent(f.EmpCtx, cancelled.ID)
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound for employee fetching CANCELLED, got %v", errs.Code(err))
	}
}

func TestGetEvent_EmployeeCannotSeeParticipants(t *testing.T) {
	f := setup(t)
	emp := makeEmployee(t, f.ClientID, f.DzoID)

	req := validCreateRequest(f)
	req.EmployeeIDs = []string{emp.String()}
	created, err := CreateEvent(f.AdminCtx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := GetEvent(f.EmpCtx, created.Event.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.Event.Participants != nil {
		t.Errorf("employees must not see participant list, got %d", len(resp.Event.Participants))
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	f := setup(t)

	_, err := GetEvent(f.AdminCtx, uuid.New().String())
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ UPDATE ════

func TestUpdateEvent_Success(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	newTitle := "Updated Title"
	resp, err := UpdateEvent(f.AdminCtx, ev.ID, &UpdateEventRequest{Title: &newTitle})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Event.Title != newTitle {
		t.Errorf("title not updated")
	}
}

func TestUpdateEvent_ReplaceParticipants(t *testing.T) {
	f := setup(t)
	emp1 := makeEmployee(t, f.ClientID, f.DzoID)
	emp2 := makeEmployee(t, f.ClientID, f.DzoID)

	req := validCreateRequest(f)
	req.EmployeeIDs = []string{emp1.String()}
	created, err := CreateEvent(f.AdminCtx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newList := []string{emp2.String()}
	resp, err := UpdateEvent(f.AdminCtx, created.Event.ID, &UpdateEventRequest{EmployeeIDs: &newList})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resp.Event.ParticipantsCount != 1 {
		t.Errorf("expected 1 participant after replace, got %d", resp.Event.ParticipantsCount)
	}
	if len(resp.Event.Participants) != 1 || resp.Event.Participants[0].EmployeeID != emp2.String() {
		t.Errorf("expected emp2 to be the only participant")
	}
}

func TestUpdateEvent_PermissionDenied(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	newTitle := "Hack"
	_, err := UpdateEvent(f.EmpCtx, ev.ID, &UpdateEventRequest{Title: &newTitle})
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

// ════ DELETE ════

func TestDeleteEvent_SoftDelete(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	if _, err := DeleteEvent(f.AdminCtx, ev.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := GetEvent(f.AdminCtx, ev.ID)
	if err != nil {
		t.Fatalf("admin should still see cancelled event: %v", err)
	}
	if resp.Event.Status != StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", resp.Event.Status)
	}
}

func TestDeleteEvent_NotFound(t *testing.T) {
	f := setup(t)

	_, err := DeleteEvent(f.AdminCtx, uuid.New().String())
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ MATERIALS ════

func TestSetMaterials_Success(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	resp, err := SetEventMaterials(f.AdminCtx, ev.ID, &SetMaterialsRequest{
		MaterialsURL: "https://drive.google.com/folders/abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Event.MaterialsURL == nil || *resp.Event.MaterialsURL != "https://drive.google.com/folders/abc" {
		t.Errorf("materials_url not set, got %v", resp.Event.MaterialsURL)
	}
}

func TestSetMaterials_EmptyURLRejected(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	_, err := SetEventMaterials(f.AdminCtx, ev.ID, &SetMaterialsRequest{MaterialsURL: "  "})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestClearMaterials_Success(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	if _, err := SetEventMaterials(f.AdminCtx, ev.ID, &SetMaterialsRequest{
		MaterialsURL: "https://onedrive.live.com/folder",
	}); err != nil {
		t.Fatalf("set: %v", err)
	}

	resp, err := ClearEventMaterials(f.AdminCtx, ev.ID)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if resp.Event.MaterialsURL != nil {
		t.Errorf("expected nil materials_url, got %v", *resp.Event.MaterialsURL)
	}
}

func TestSetMaterials_PermissionDenied(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	_, err := SetEventMaterials(f.EmpCtx, ev.ID, &SetMaterialsRequest{
		MaterialsURL: "https://drive.google.com/folders/x",
	})
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

// ════ LIST HOSTS ════

// makeAdmin inserts an ADM user bound to a client with explicit
// is_active / is_onboarded flags.
func makeAdmin(t *testing.T, clientID uuid.UUID, isActive, isOnboarded bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	suffix := id.String()[:8]
	_, err := Client.User.
		Create().
		SetID(id).
		SetRole(string(authhandler.RoleADM)).
		SetEmail("adm-" + suffix + "@test.com").
		SetKeycloakUserID("adm-" + suffix).
		SetClientID(clientID).
		SetIsActive(isActive).
		SetIsOnboarded(isOnboarded).
		Save(context.Background())
	if err != nil {
		t.Fatalf("makeAdmin: %v", err)
	}
	return id
}

// makeEmployeeUser inserts a User + Employee linked together, returns
// (employeeID, authCtx) where authCtx carries the user's keycloak_user_id.
// The auth context resolves through lookupEmployeeID → empID.
func makeEmployeeUser(t *testing.T, clientID, dzoID uuid.UUID) (uuid.UUID, context.Context) {
	t.Helper()
	suffix := uuid.New().String()[:8]
	kcID := "emp-" + suffix

	userID := uuid.New()
	if _, err := Client.User.
		Create().
		SetID(userID).
		SetRole(string(authhandler.RoleEMP)).
		SetEmail(kcID + "@test.com").
		SetKeycloakUserID(kcID).
		SetClientID(clientID).
		SetIsActive(true).
		SetIsOnboarded(true).
		Save(context.Background()); err != nil {
		t.Fatalf("makeEmployeeUser: user: %v", err)
	}

	row, err := Client.Employee.
		Create().
		SetClientID(clientID).
		SetDzoID(dzoID).
		SetFullName("Emp " + suffix).
		SetEmail(kcID + "@test.com").
		SetIsActive(true).
		SetUserID(userID).
		Save(context.Background())
	if err != nil {
		t.Fatalf("makeEmployeeUser: employee: %v", err)
	}

	c := auth.WithContext(context.Background(), auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Role:           authhandler.RoleEMP,
		CompanyID:      clientID.String(),
	})
	return row.ID, c
}

// ════ EMPLOYEE REGISTRATION FLOW ════

func TestRegisterForEvent_Success(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)
	_, empCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)

	resp, err := RegisterForEvent(empCtx, ev.ID)
	if err != nil {
		t.Fatalf("RegisterForEvent: %v", err)
	}
	if !resp.Event.IsRegistered {
		t.Errorf("expected is_registered=true")
	}
	if resp.Event.ZoomLink == "" {
		t.Errorf("expected zoom_link to be visible after registration")
	}
	if resp.Event.AvailableSlots != ev.MaxParticipants-1 {
		t.Errorf("expected available_slots=%d, got %d",
			ev.MaxParticipants-1, resp.Event.AvailableSlots)
	}
}

func TestRegisterForEvent_AlreadyRegistered(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)
	_, empCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)

	if _, err := RegisterForEvent(empCtx, ev.ID); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := RegisterForEvent(empCtx, ev.ID)
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

func TestRegisterForEvent_NoSlots(t *testing.T) {
	f := setup(t)

	resp, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:           "Tiny Webinar " + uuid.New().String()[:6],
		EventDate:       time.Now().Add(48 * time.Hour),
		HostID:          f.HostID.String(),
		ZoomLink:        "https://zoom.us/j/9",
		MaxParticipants: 1,
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	ev := resp.Event

	_, firstCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)
	_, secondCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)

	if _, err := RegisterForEvent(firstCtx, ev.ID); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err = RegisterForEvent(secondCtx, ev.ID)
	if errs.Code(err) != errs.FailedPrecondition {
		t.Errorf("expected FailedPrecondition (no slots), got %v", errs.Code(err))
	}
}

func TestUnregisterFromEvent_Success(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)
	_, empCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)

	if _, err := RegisterForEvent(empCtx, ev.ID); err != nil {
		t.Fatalf("register: %v", err)
	}
	resp, err := UnregisterFromEvent(empCtx, ev.ID)
	if err != nil {
		t.Fatalf("UnregisterFromEvent: %v", err)
	}
	if resp.Event.IsRegistered {
		t.Errorf("expected is_registered=false after unregister")
	}
	if resp.Event.ZoomLink != "" {
		t.Errorf("expected zoom_link redacted after unregister")
	}
	if resp.Event.AvailableSlots != ev.MaxParticipants {
		t.Errorf("expected available_slots restored to %d, got %d",
			ev.MaxParticipants, resp.Event.AvailableSlots)
	}
}

func TestUnregisterFromEvent_NotRegistered(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)
	_, empCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)

	_, err := UnregisterFromEvent(empCtx, ev.ID)
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestGetEvent_EMPSeesIsRegisteredAndHidesZoomUntilEnrolled(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)
	_, empCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)

	before, err := GetEvent(empCtx, ev.ID)
	if err != nil {
		t.Fatalf("GetEvent before: %v", err)
	}
	if before.Event.IsRegistered {
		t.Errorf("expected is_registered=false before enroll")
	}
	if before.Event.ZoomLink != "" {
		t.Errorf("expected zoom_link hidden for non-registered EMP")
	}

	if _, err := RegisterForEvent(empCtx, ev.ID); err != nil {
		t.Fatalf("register: %v", err)
	}
	after, err := GetEvent(empCtx, ev.ID)
	if err != nil {
		t.Fatalf("GetEvent after: %v", err)
	}
	if !after.Event.IsRegistered {
		t.Errorf("expected is_registered=true after enroll")
	}
	if after.Event.ZoomLink == "" {
		t.Errorf("expected zoom_link visible to registered EMP")
	}
}

func TestRegisterForEvent_PermissionDenied(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	_, err := RegisterForEvent(f.AdminCtx, ev.ID)
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied for ADM, got %v", errs.Code(err))
	}
}

// Pending admins (registered via RegisterAdmin, not yet logged in:
// is_active=false, is_onboarded=false) must appear in the host picker
// so SA can pre-assign them as event hosts.
func TestListHosts_IncludesPendingAdmin(t *testing.T) {
	clientID := makeClient(t)

	activeID := makeAdmin(t, clientID, true, true)
	pendingID := makeAdmin(t, clientID, false, false)
	blockedID := makeAdmin(t, clientID, false, true)

	ctx := withRole(authhandler.RoleADM, clientID)
	resp, err := ListHosts(ctx, &ListHostsParams{Limit: 50})
	if err != nil {
		t.Fatalf("ListHosts: %v", err)
	}

	got := make(map[string]bool, len(resp.Hosts))
	for _, h := range resp.Hosts {
		got[h.ID] = true
	}

	if !got[activeID.String()] {
		t.Errorf("active admin %s missing from hosts", activeID)
	}
	if !got[pendingID.String()] {
		t.Errorf("pending admin %s missing from hosts (regression: should be selectable)", pendingID)
	}
	if got[blockedID.String()] {
		t.Errorf("blocked admin %s leaked into hosts", blockedID)
	}
}

// ════ COMPLETE EVENT + ATTENDANCE FLOW ════

// makeAdminUser inserts a User+Employee for an SA/ADM/HR caller and returns
// (userID, ctx). Used so attendance updates can stamp reviewed_by.
func makeAdminUser(t *testing.T, clientID uuid.UUID, role authhandler.UserRole) (uuid.UUID, context.Context) {
	t.Helper()
	suffix := uuid.New().String()[:8]
	kcID := strings.ToLower(string(role)) + "-" + suffix
	uid := uuid.New()
	if _, err := Client.User.
		Create().
		SetID(uid).
		SetRole(string(role)).
		SetEmail(kcID + "@test.com").
		SetKeycloakUserID(kcID).
		SetClientID(clientID).
		SetIsActive(true).
		SetIsOnboarded(true).
		Save(context.Background()); err != nil {
		t.Fatalf("makeAdminUser: %v", err)
	}
	c := auth.WithContext(context.Background(), auth.UID(kcID), &authhandler.AuthData{
		KeycloakUserID: kcID,
		Role:           role,
		CompanyID:      clientID.String(),
	})
	return uid, c
}

func TestCompleteEvent_Success(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	resp, err := CompleteEvent(f.AdminCtx, ev.ID)
	if err != nil {
		t.Fatalf("CompleteEvent: %v", err)
	}
	if resp.Event.Status != StatusCompleted {
		t.Errorf("expected status COMPLETED, got %s", resp.Event.Status)
	}
}

func TestCompleteEvent_AlreadyCompleted(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	if _, err := CompleteEvent(f.AdminCtx, ev.ID); err != nil {
		t.Fatalf("first complete: %v", err)
	}
	_, err := CompleteEvent(f.AdminCtx, ev.ID)
	if errs.Code(err) != errs.FailedPrecondition {
		t.Errorf("expected FailedPrecondition on re-complete, got %v", errs.Code(err))
	}
}

func TestCompleteEvent_PermissionDenied(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	_, err := CompleteEvent(f.EmpCtx, ev.ID)
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestListAttendance_ReturnsAllParticipants(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)
	emp1, ctx1 := makeEmployeeUser(t, f.ClientID, f.DzoID)
	emp2, ctx2 := makeEmployeeUser(t, f.ClientID, f.DzoID)
	if _, err := RegisterForEvent(ctx1, ev.ID); err != nil {
		t.Fatalf("register emp1: %v", err)
	}
	if _, err := RegisterForEvent(ctx2, ev.ID); err != nil {
		t.Fatalf("register emp2: %v", err)
	}

	resp, err := ListAttendance(f.AdminCtx, ev.ID, &ListAttendanceParams{})
	if err != nil {
		t.Fatalf("ListAttendance: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected 2 participants, got %d", resp.Total)
	}
	got := make(map[string]string, len(resp.Participants))
	for _, p := range resp.Participants {
		got[p.EmployeeID] = p.AttendanceStatus
	}
	if got[emp1.String()] != "PENDING" || got[emp2.String()] != "PENDING" {
		t.Errorf("expected both PENDING, got %v", got)
	}
}

func TestListAttendance_SearchByName(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	// Create two employees with predictable distinct names.
	suffix1 := uuid.New().String()[:6]
	suffix2 := uuid.New().String()[:6]
	makeNamed := func(name, suffix string) (uuid.UUID, context.Context) {
		kcID := "emp-" + suffix
		uid := uuid.New()
		if _, err := Client.User.Create().
			SetID(uid).SetRole(string(authhandler.RoleEMP)).
			SetEmail(kcID + "@test.com").SetKeycloakUserID(kcID).
			SetClientID(f.ClientID).SetIsActive(true).SetIsOnboarded(true).
			Save(context.Background()); err != nil {
			t.Fatalf("create user: %v", err)
		}
		row, err := Client.Employee.Create().
			SetClientID(f.ClientID).SetDzoID(f.DzoID).
			SetFullName(name).SetEmail(kcID + "@test.com").
			SetIsActive(true).SetUserID(uid).
			Save(context.Background())
		if err != nil {
			t.Fatalf("create emp: %v", err)
		}
		c := auth.WithContext(context.Background(), auth.UID(kcID), &authhandler.AuthData{
			KeycloakUserID: kcID, Role: authhandler.RoleEMP, CompanyID: f.ClientID.String(),
		})
		return row.ID, c
	}
	alice, aliceCtx := makeNamed("Alice "+suffix1, suffix1)
	_, bobCtx := makeNamed("Bob "+suffix2, suffix2)
	if _, err := RegisterForEvent(aliceCtx, ev.ID); err != nil {
		t.Fatalf("register alice: %v", err)
	}
	if _, err := RegisterForEvent(bobCtx, ev.ID); err != nil {
		t.Fatalf("register bob: %v", err)
	}

	resp, err := ListAttendance(f.AdminCtx, ev.ID, &ListAttendanceParams{Search: "alice"})
	if err != nil {
		t.Fatalf("ListAttendance: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 result for 'alice', got %d", resp.Total)
	}
	if resp.Participants[0].EmployeeID != alice.String() {
		t.Errorf("expected alice's row, got %s", resp.Participants[0].EmployeeID)
	}
}

func TestUpdateAttendance_Success(t *testing.T) {
	f := setup(t)
	_, adminCtx := makeAdminUser(t, f.ClientID, authhandler.RoleADM)

	ev, err := CreateEvent(adminCtx, &CreateEventRequest{
		Title:           "Webinar " + uuid.New().String()[:6],
		EventDate:       time.Now().Add(48 * time.Hour),
		HostID:          f.HostID.String(),
		ZoomLink:        "https://zoom.us/j/x",
		MaxParticipants: 5,
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	emp1, ctx1 := makeEmployeeUser(t, f.ClientID, f.DzoID)
	emp2, ctx2 := makeEmployeeUser(t, f.ClientID, f.DzoID)
	if _, err := RegisterForEvent(ctx1, ev.Event.ID); err != nil {
		t.Fatalf("register emp1: %v", err)
	}
	if _, err := RegisterForEvent(ctx2, ev.Event.ID); err != nil {
		t.Fatalf("register emp2: %v", err)
	}
	if _, err := CompleteEvent(adminCtx, ev.Event.ID); err != nil {
		t.Fatalf("complete: %v", err)
	}

	resp, err := UpdateAttendance(adminCtx, ev.Event.ID, &UpdateAttendanceRequest{
		Updates: []AttendanceUpdate{
			{EmployeeID: emp1.String(), Attended: true},
			{EmployeeID: emp2.String(), Attended: false},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAttendance: %v", err)
	}
	if resp.Updated != 2 {
		t.Errorf("expected updated=2, got %d", resp.Updated)
	}

	listResp, err := ListAttendance(adminCtx, ev.Event.ID, &ListAttendanceParams{})
	if err != nil {
		t.Fatalf("ListAttendance: %v", err)
	}
	got := make(map[string]string, len(listResp.Participants))
	for _, p := range listResp.Participants {
		got[p.EmployeeID] = p.AttendanceStatus
	}
	if got[emp1.String()] != "ATTENDED" {
		t.Errorf("expected emp1=ATTENDED, got %s", got[emp1.String()])
	}
	if got[emp2.String()] != "MISSED" {
		t.Errorf("expected emp2=MISSED, got %s", got[emp2.String()])
	}
}

func TestUpdateAttendance_RejectsNonCompleted(t *testing.T) {
	f := setup(t)
	_, adminCtx := makeAdminUser(t, f.ClientID, authhandler.RoleADM)
	ev := makeEvent(t, f)
	emp, empCtx := makeEmployeeUser(t, f.ClientID, f.DzoID)
	if _, err := RegisterForEvent(empCtx, ev.ID); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := UpdateAttendance(adminCtx, ev.ID, &UpdateAttendanceRequest{
		Updates: []AttendanceUpdate{{EmployeeID: emp.String(), Attended: true}},
	})
	if errs.Code(err) != errs.FailedPrecondition {
		t.Errorf("expected FailedPrecondition on ACTIVE event, got %v", errs.Code(err))
	}
}

func TestUpdateAttendance_UnknownEmployeeIgnored(t *testing.T) {
	f := setup(t)
	_, adminCtx := makeAdminUser(t, f.ClientID, authhandler.RoleADM)
	ev, err := CreateEvent(adminCtx, &CreateEventRequest{
		Title:           "Webinar " + uuid.New().String()[:6],
		EventDate:       time.Now().Add(48 * time.Hour),
		HostID:          f.HostID.String(),
		ZoomLink:        "https://zoom.us/j/x",
		MaxParticipants: 5,
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if _, err := CompleteEvent(adminCtx, ev.Event.ID); err != nil {
		t.Fatalf("complete: %v", err)
	}

	resp, err := UpdateAttendance(adminCtx, ev.Event.ID, &UpdateAttendanceRequest{
		Updates: []AttendanceUpdate{
			{EmployeeID: uuid.New().String(), Attended: true},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAttendance: %v", err)
	}
	if resp.Updated != 0 {
		t.Errorf("expected updated=0 for unknown employee, got %d", resp.Updated)
	}
}

func TestUpdateAttendance_PermissionDenied(t *testing.T) {
	f := setup(t)
	ev := makeEvent(t, f)

	_, err := UpdateAttendance(f.EmpCtx, ev.ID, &UpdateAttendanceRequest{
		Updates: []AttendanceUpdate{{EmployeeID: uuid.New().String(), Attended: true}},
	})
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}
