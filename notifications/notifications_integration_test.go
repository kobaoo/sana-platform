package notifications

import (
	"context"
	"testing"
	"time"

	"encore.dev/beta/auth"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
)

// ════ HELPERS ════

func ctx() context.Context {
	return context.Background()
}

func newID() string {
	return uuid.NewString()
}

func makeNotifyRequest(userID string) *NotifyRequest {
	return &NotifyRequest{
		UserID:     userID,
		Type:       TypeRequestCreated,
		EntityType: EntityRequest,
		EntityID:   newID(),
	}
}

func authCtx(userID string) context.Context {
	return auth.WithContext(ctx(), auth.UID(userID), &authhandler.AuthData{
		KeycloakUserID: userID,
		Role:           authhandler.RoleEMP,
	})
}

func countNotifications(t *testing.T, req *NotifyRequest) int {
	t.Helper()

	var count int
	err := db.QueryRow(ctx(),
		`SELECT COUNT(*) FROM notifications
		 WHERE user_id = $1 AND type = $2 AND entity_type = $3 AND entity_id = $4`,
		req.UserID, string(req.Type), string(req.EntityType), req.EntityID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("countNotifications: %v", err)
	}
	return count
}

func insertNotificationAt(t *testing.T, userID string, typ NotificationType, entityID string, createdAt time.Time) {
	t.Helper()

	_, err := db.Exec(ctx(),
		`INSERT INTO notifications
		   (user_id, type, entity_type, entity_id, status, sent_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, type, entity_type, entity_id) DO NOTHING`,
		userID, string(typ), string(EntityRequest), entityID,
		string(StatusSent), createdAt, createdAt,
	)
	if err != nil {
		t.Fatalf("insertNotificationAt: %v", err)
	}
}

// ════ NOTIFY ════

func TestNotifyUserWithEntity_SuccessSavesNotification(t *testing.T) {
	req := makeNotifyRequest(newID())

	resp, err := NotifyUserWithEntity(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Skipped {
		t.Fatal("expected notification to be sent, got skipped")
	}
	if resp.Message != "notification sent" {
		t.Errorf("expected message 'notification sent', got %q", resp.Message)
	}
	if got := countNotifications(t, req); got != 1 {
		t.Fatalf("expected 1 saved notification, got %d", got)
	}

	exists, err := notificationExists(ctx(), req)
	if err != nil {
		t.Fatalf("notificationExists: %v", err)
	}
	if !exists {
		t.Fatal("expected notificationExists to return true")
	}
}

func TestNotifyUserWithEntity_DuplicateSkipped(t *testing.T) {
	req := makeNotifyRequest(newID())

	first, err := NotifyUserWithEntity(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected first notify error: %v", err)
	}
	if first.Skipped {
		t.Fatal("first notification should not be skipped")
	}

	second, err := NotifyUserWithEntity(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected second notify error: %v", err)
	}
	if !second.Skipped {
		t.Fatal("expected duplicate notification to be skipped")
	}
	if second.Message != "duplicate: already notified" {
		t.Errorf("unexpected duplicate message: %q", second.Message)
	}
	if got := countNotifications(t, req); got != 1 {
		t.Fatalf("expected duplicate protection to keep 1 row, got %d", got)
	}
}

func TestSaveNotification_DuplicateConflictDoesNotInsertTwice(t *testing.T) {
	req := makeNotifyRequest(newID())

	if err := saveNotification(ctx(), req); err != nil {
		t.Fatalf("unexpected first save error: %v", err)
	}
	if err := saveNotification(ctx(), req); err != nil {
		t.Fatalf("unexpected duplicate save error: %v", err)
	}
	if got := countNotifications(t, req); got != 1 {
		t.Fatalf("expected 1 row after duplicate save, got %d", got)
	}
}

func TestSaveFailedNotification_SavesFailedStatus(t *testing.T) {
	req := makeNotifyRequest(newID())
	req.Type = TypeCertExpired
	req.EntityType = EntityCertificate

	if err := saveFailedNotification(ctx(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var status string
	err := db.QueryRow(ctx(),
		`SELECT status FROM notifications
		 WHERE user_id = $1 AND type = $2 AND entity_type = $3 AND entity_id = $4`,
		req.UserID, string(req.Type), string(req.EntityType), req.EntityID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != string(StatusFailed) {
		t.Fatalf("expected FAILED status, got %q", status)
	}
}

// ════ LIST ════

func TestListNotifications_ReturnsOnlyRequestedUser(t *testing.T) {
	userID := newID()
	otherUserID := newID()

	req := makeNotifyRequest(userID)
	otherReq := makeNotifyRequest(otherUserID)

	if err := saveNotification(ctx(), req); err != nil {
		t.Fatalf("save notification: %v", err)
	}
	if err := saveNotification(ctx(), otherReq); err != nil {
		t.Fatalf("save other notification: %v", err)
	}

	items, err := listNotifications(ctx(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundOwn := false
	for _, item := range items {
		if item.UserID == otherUserID {
			t.Fatal("listNotifications returned notification for another user")
		}
		if item.UserID == userID && item.EntityID == req.EntityID {
			foundOwn = true
		}
	}
	if !foundOwn {
		t.Fatal("expected own notification to be returned")
	}
}

func TestListNotifications_OrderByCreatedAtDesc(t *testing.T) {
	userID := newID()
	oldEntityID := newID()
	newEntityID := newID()

	insertNotificationAt(t, userID, TypeRequestCreated, oldEntityID, time.Now().Add(-1*time.Hour))
	insertNotificationAt(t, userID, TypeRequestApproved, newEntityID, time.Now())

	items, err := listNotifications(ctx(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected at least 2 notifications, got %d", len(items))
	}
	if items[0].EntityID != newEntityID {
		t.Fatalf("expected newest notification first, got entity_id %q", items[0].EntityID)
	}
}

func TestListNotifications_EmptyReturnsEmptySlice(t *testing.T) {
	userID := uuid.NewString()

	notifications, err := listNotifications(ctx(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifications) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(notifications))
	}
}

// ════ ENDPOINTS ════

func TestListNotifications_ReturnsUserNotifications(t *testing.T) {
	userID := newID()
	otherUserID := newID()

	req := makeNotifyRequest(userID)
	if err := saveNotification(ctx(), req); err != nil {
		t.Fatalf("save notification: %v", err)
	}

	otherReq := makeNotifyRequest(otherUserID)
	if err := saveNotification(ctx(), otherReq); err != nil {
		t.Fatalf("save notification: %v", err)
	}

	notifications, err := listNotifications(ctx(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	if notifications[0].UserID != userID {
		t.Fatalf(
			"expected user_id %q, got %q",
			userID,
			notifications[0].UserID,
		)
	}

	if notifications[0].EntityID != req.EntityID {
		t.Fatal("expected saved notification in response")
	}
}

// ════ EVENTS ════

func TestCertificateExpiryEvent_ToNotifyRequest(t *testing.T) {
	e := CertificateExpiryEvent{
		EmployeeID: newID(),
		CertID:     newID(),
		Title:      "Safety Certificate",
		ExpiryDate: "2026-05-01",
	}

	req := e.ToNotifyRequest()
	if req.UserID != e.EmployeeID {
		t.Errorf("expected user_id %q, got %q", e.EmployeeID, req.UserID)
	}
	if req.Type != TypeCertExpiring {
		t.Errorf("expected type CERT_EXPIRING, got %q", req.Type)
	}
	if req.EntityType != EntityCertificate {
		t.Errorf("expected entity CERTIFICATE, got %q", req.EntityType)
	}
	if req.EntityID != e.CertID {
		t.Errorf("expected entity_id %q, got %q", e.CertID, req.EntityID)
	}
}

func TestRequestEvents_ToNotifyRequest(t *testing.T) {
	requestID := newID()
	approverID := newID()
	initiatorID := newID()

	created := RequestCreatedEvent{ApproverID: approverID, RequestID: requestID}.ToNotifyRequest()
	if created.UserID != approverID || created.Type != TypeRequestCreated || created.EntityType != EntityRequest || created.EntityID != requestID {
		t.Fatal("RequestCreatedEvent mapped to invalid NotifyRequest")
	}

	step := RequestStepUpdatedEvent{ApproverID: approverID, RequestID: requestID, Step: 2}.ToNotifyRequest()
	if step.UserID != approverID || step.Type != TypeRequestStepUpdated || step.EntityType != EntityRequest || step.EntityID != requestID {
		t.Fatal("RequestStepUpdatedEvent mapped to invalid NotifyRequest")
	}

	approved := RequestApprovedEvent{InitiatorID: initiatorID, RequestID: requestID}.ToNotifyRequest()
	if approved.UserID != initiatorID || approved.Type != TypeRequestApproved || approved.EntityType != EntityRequest || approved.EntityID != requestID {
		t.Fatal("RequestApprovedEvent mapped to invalid NotifyRequest")
	}

	cancelled := RequestCancelledEvent{InitiatorID: initiatorID, RequestID: requestID}.ToNotifyRequest()
	if cancelled.UserID != initiatorID || cancelled.Type != TypeRequestCancelled || cancelled.EntityType != EntityRequest || cancelled.EntityID != requestID {
		t.Fatal("RequestCancelledEvent mapped to invalid NotifyRequest")
	}
}
