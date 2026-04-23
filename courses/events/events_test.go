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
	HostID   uuid.UUID
	AdminCtx context.Context
	HRCtx    context.Context
	EmpCtx   context.Context
}

func setup(t *testing.T) fixture {
	t.Helper()
	clientID := makeClient(t)
	hostID := makeUser(t, authhandler.RoleHR)
	return fixture{
		ClientID: clientID,
		HostID:   hostID,
		AdminCtx: withRole(authhandler.RoleADM, clientID),
		HRCtx:    withRole(authhandler.RoleHR, clientID),
		EmpCtx:   withRole(authhandler.RoleEMP, clientID),
	}
}

func makeEvent(t *testing.T, f fixture) *Event {
	t.Helper()
	resp, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:     "Webinar " + uuid.New().String()[:6],
		EventDate: time.Now().Add(48 * time.Hour),
		HostID:    f.HostID.String(),
	})
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

// ════ CREATE ════

func TestCreateEvent_Success(t *testing.T) {
	f := setup(t)

	resp, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:     "Live Webinar",
		EventDate: time.Now().Add(24 * time.Hour),
		HostID:    f.HostID.String(),
	})
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
}

func TestCreateEvent_MissingTitle(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		EventDate: time.Now().Add(24 * time.Hour),
		HostID:    f.HostID.String(),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_MissingDate(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:  "No Date",
		HostID: f.HostID.String(),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_MissingHost(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:     "No Host",
		EventDate: time.Now().Add(24 * time.Hour),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_PastDateRejected(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:     "Past Webinar",
		EventDate: time.Now().Add(-24 * time.Hour),
		HostID:    f.HostID.String(),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateEvent_PermissionDenied(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.EmpCtx, &CreateEventRequest{
		Title:     "Forbidden",
		EventDate: time.Now().Add(24 * time.Hour),
		HostID:    f.HostID.String(),
	})
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestCreateEvent_InvalidHost(t *testing.T) {
	f := setup(t)

	_, err := CreateEvent(f.AdminCtx, &CreateEventRequest{
		Title:     "Bad Host",
		EventDate: time.Now().Add(24 * time.Hour),
		HostID:    uuid.New().String(),
	})
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument for unknown host, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestListEvents_AdminSeesActiveAndCancelled(t *testing.T) {
	f := setup(t)
	makeEvent(t, f)          // ACTIVE
	makeCancelledEvent(t, f) // CANCELLED

	resp, err := ListEvents(f.AdminCtx)
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

	resp, err := ListEvents(f.EmpCtx)
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

func TestGetEvent_EmployeeCannotSeeCancelled(t *testing.T) {
	f := setup(t)
	cancelled := makeCancelledEvent(t, f)

	_, err := GetEvent(f.EmpCtx, cancelled.ID)
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound for employee fetching CANCELLED, got %v", errs.Code(err))
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
