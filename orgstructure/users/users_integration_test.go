//go:build integration
package users

import (
	"context"
	"testing"

	"encore.dev/beta/errs"
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

func newEmail() string {
	return uuid.NewString() + "@example.com"
}

func validDzoID() string {
	return uuid.NewString()
}

func makeUser(t *testing.T, role authhandler.UserRole, dzoID *string) *User {
	t.Helper()

	resp, err := insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		Role:           role,
		DzoID:          dzoID,
	})
	if err != nil {
		t.Fatalf("makeUser: %v", err)
	}
	return resp
}

func makePendingAdmin(t *testing.T, dzoID string) *User {
	t.Helper()

	resp, err := insertPendingAdmin(ctx(), &RegisterAdminRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		DzoID:          &dzoID,
	})
	if err != nil {
		t.Fatalf("makePendingAdmin: %v", err)
	}
	return resp
}

// ════ CREATE / INSERT ════

func TestInsertUser_Success(t *testing.T) {
	u, err := insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		Role:           authhandler.RoleEMP,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if u.Role != authhandler.RoleEMP {
		t.Errorf("expected role EMP, got %q", u.Role)
	}
	if !u.IsActive {
		t.Error("expected user to be active by default")
	}
	if !u.IsOnboarded {
		t.Error("expected user to be onboarded by default")
	}
}

func TestInsertUser_SuccessWithDzo(t *testing.T) {
	dzoID := validDzoID()

	u, err := insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		Role:           authhandler.RoleHR,
		DzoID:          &dzoID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.DzoID == nil || *u.DzoID != dzoID {
		t.Errorf("expected dzo_id %q, got %v", dzoID, u.DzoID)
	}
}

func TestInsertUser_InvalidDzoFormat(t *testing.T) {
	badDzoID := "not-a-uuid"

	_, err := insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		Role:           authhandler.RoleEMP,
		DzoID:          &badDzoID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestInsertUser_DuplicateKeycloakUserID(t *testing.T) {
	kcID := newID()

	_, err := insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: kcID,
		Email:          newEmail(),
		Role:           authhandler.RoleEMP,
	})
	if err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	_, err = insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: kcID,
		Email:          newEmail(),
		Role:           authhandler.RoleEMP,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

func TestInsertPendingAdmin_Success(t *testing.T) {
	dzoID := validDzoID()

	u, err := insertPendingAdmin(ctx(), &RegisterAdminRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		DzoID:          &dzoID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Role != authhandler.RoleADM {
		t.Errorf("expected role ADM, got %q", u.Role)
	}
	if u.IsActive {
		t.Error("expected pending admin to be inactive")
	}
	if u.IsOnboarded {
		t.Error("expected pending admin to be non-onboarded")
	}
	if u.DzoID == nil || *u.DzoID != dzoID {
		t.Errorf("expected dzo_id %q, got %v", dzoID, u.DzoID)
	}
}

func TestInsertPendingAdmin_InvalidDzoFormat(t *testing.T) {
	badDzoID := "not-a-uuid"

	_, err := insertPendingAdmin(ctx(), &RegisterAdminRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		DzoID:          &badDzoID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestInsertPendingAdmin_DuplicateKeycloakUserID(t *testing.T) {
	kcID := newID()
	dzo1 := validDzoID()
	dzo2 := validDzoID()

	_, err := insertPendingAdmin(ctx(), &RegisterAdminRequest{
		KeycloakUserID: kcID,
		Email:          newEmail(),
		DzoID:          &dzo1,
	})
	if err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	_, err = insertPendingAdmin(ctx(), &RegisterAdminRequest{
		KeycloakUserID: kcID,
		Email:          newEmail(),
		DzoID:          &dzo2,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %v", errs.Code(err))
	}
}

// ════ GET / QUERY ════

func TestQueryUserByID_Success(t *testing.T) {
	created := makeUser(t, authhandler.RoleEMP, nil)

	u, err := queryUserByID(ctx(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, u.ID)
	}
}

func TestQueryUserByID_InvalidIDFormat(t *testing.T) {
	_, err := queryUserByID(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestQueryUserByID_NotFound(t *testing.T) {
	_, err := queryUserByID(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestQueryUserByKeycloakID_Success(t *testing.T) {
	kcID := newID()

	created, err := insertUser(ctx(), &CreateUserRequest{
		KeycloakUserID: kcID,
		Email:          newEmail(),
		Role:           authhandler.RoleEMP,
	})
	if err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	u, err := queryUserByKeycloakID(ctx(), kcID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, u.ID)
	}
}

func TestQueryUserByKeycloakID_NotFound(t *testing.T) {
	_, err := queryUserByKeycloakID(ctx(), newID())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ LIST ════

func TestQueryActiveUsers_ReturnsOnlyActiveUsers(t *testing.T) {
	active := makeUser(t, authhandler.RoleEMP, nil)
	inactive := makeUser(t, authhandler.RoleEMP, nil)

	if err := updateUserActive(ctx(), inactive.ID, false); err != nil {
		t.Fatalf("failed to deactivate user: %v", err)
	}

	users, err := queryActiveUsers(ctx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	for _, u := range users {
		if u.ID == inactive.ID {
			t.Error("inactive user should not appear in active users list")
		}
		if u.ID == active.ID {
			foundActive = true
		}
	}
	if !foundActive {
		t.Error("active user should appear in active users list")
	}
}

func TestQueryActiveUsersByDzo_ReturnsOnlyActiveUsersForDzo(t *testing.T) {
	dzoA := validDzoID()
	dzoB := validDzoID()

	a1 := makeUser(t, authhandler.RoleEMP, &dzoA)
	a2 := makeUser(t, authhandler.RoleHR, &dzoA)
	b1 := makeUser(t, authhandler.RoleEMP, &dzoB)

	if err := updateUserActive(ctx(), a2.ID, false); err != nil {
		t.Fatalf("failed to deactivate user: %v", err)
	}

	users, err := queryActiveUsersByDzo(ctx(), dzoA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundA1 := false
	for _, u := range users {
		if u.ID == a2.ID {
			t.Error("inactive DZO user should not appear in list")
		}
		if u.ID == b1.ID {
			t.Error("user from another DZO should not appear in list")
		}
		if u.ID == a1.ID {
			foundA1 = true
		}
	}
	if !foundA1 {
		t.Error("expected active same-DZO user to appear")
	}
}

func TestQueryActiveUsersByDzo_InvalidDzoFormat(t *testing.T) {
	_, err := queryActiveUsersByDzo(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

// ════ UPDATE ════

func TestUpdateUserRole_Success(t *testing.T) {
	u := makeUser(t, authhandler.RoleEMP, nil)
	dzoID := validDzoID()

	updated, err := updateUserRole(ctx(), u.ID, authhandler.RoleHR, &dzoID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Role != authhandler.RoleHR {
		t.Errorf("expected role HR, got %q", updated.Role)
	}
	if updated.DzoID == nil || *updated.DzoID != dzoID {
		t.Errorf("expected dzo_id %q, got %v", dzoID, updated.DzoID)
	}
}

func TestUpdateUserRole_InvalidIDFormat(t *testing.T) {
	_, err := updateUserRole(ctx(), "not-a-uuid", authhandler.RoleEMP, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateUserRole_InvalidDzoFormat(t *testing.T) {
	u := makeUser(t, authhandler.RoleEMP, nil)
	badDzoID := "not-a-uuid"

	_, err := updateUserRole(ctx(), u.ID, authhandler.RoleHR, &badDzoID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateUserRole_NotFound(t *testing.T) {
	_, err := updateUserRole(ctx(), "00000000-0000-0000-0000-000000000000", authhandler.RoleEMP, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestUpdateUserActive_Success(t *testing.T) {
	u := makeUser(t, authhandler.RoleEMP, nil)

	if err := updateUserActive(ctx(), u.ID, false); err != nil {
		t.Fatalf("unexpected error deactivating user: %v", err)
	}

	got, err := queryUserByID(ctx(), u.ID)
	if err != nil {
		t.Fatalf("unexpected error querying user: %v", err)
	}
	if got.IsActive {
		t.Error("expected user to be inactive")
	}

	if err := updateUserActive(ctx(), u.ID, true); err != nil {
		t.Fatalf("unexpected error activating user: %v", err)
	}

	got, err = queryUserByID(ctx(), u.ID)
	if err != nil {
		t.Fatalf("unexpected error querying user: %v", err)
	}
	if !got.IsActive {
		t.Error("expected user to be active")
	}
}

func TestUpdateUserActive_InvalidIDFormat(t *testing.T) {
	err := updateUserActive(ctx(), "not-a-uuid", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateUserActive_NotFound(t *testing.T) {
	err := updateUserActive(ctx(), "00000000-0000-0000-0000-000000000000", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

// ════ ONBOARDING / RESOLUTION ════

func TestActivateOnboarding_Success(t *testing.T) {
	dzoID := validDzoID()
	pending := makePendingAdmin(t, dzoID)

	activated, err := activateOnboarding(ctx(), pending.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !activated.IsActive {
		t.Error("expected user to be active after onboarding")
	}
	if !activated.IsOnboarded {
		t.Error("expected user to be onboarded after activation")
	}
}

func TestActivateOnboarding_InvalidIDFormat(t *testing.T) {
	_, err := activateOnboarding(ctx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestActivateOnboarding_NotFound(t *testing.T) {
	_, err := activateOnboarding(ctx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestResolveCurrentUser_AutoProvisionsEMP(t *testing.T) {
	ad := &authhandler.AuthData{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		Role:           authhandler.RoleEMP,
	}

	u, err := resolveCurrentUser(ctx(), ad)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.KeycloakUserID != ad.KeycloakUserID {
		t.Errorf("expected keycloak_user_id %q, got %q", ad.KeycloakUserID, u.KeycloakUserID)
	}
	if u.Role != authhandler.RoleEMP {
		t.Errorf("expected role EMP, got %q", u.Role)
	}
}

func TestResolveCurrentUser_AutoProvisionsSA(t *testing.T) {
	ad := &authhandler.AuthData{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		Role:           authhandler.RoleSA,
	}

	u, err := resolveCurrentUser(ctx(), ad)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Role != authhandler.RoleSA {
		t.Errorf("expected role SA, got %q", u.Role)
	}
}

func TestResolveCurrentUser_ActivatesPendingAdmin(t *testing.T) {
	dzoID := validDzoID()

	pending, err := insertPendingAdmin(ctx(), &RegisterAdminRequest{
		KeycloakUserID: newID(),
		Email:          newEmail(),
		DzoID:          &dzoID,
	})
	if err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	ad := &authhandler.AuthData{
		KeycloakUserID: pending.KeycloakUserID,
		Email:          pending.Email,
		Role:           authhandler.RoleADM,
		DzoID:          dzoID,
	}

	u, err := resolveCurrentUser(ctx(), ad)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !u.IsActive {
		t.Error("expected pending admin to become active on first login")
	}
	if !u.IsOnboarded {
		t.Error("expected pending admin to become onboarded on first login")
	}
}

func TestResolveCurrentUser_BlockedUserDenied(t *testing.T) {
	u := makeUser(t, authhandler.RoleEMP, nil)

	if err := updateUserActive(ctx(), u.ID, false); err != nil {
		t.Fatalf("failed to block user: %v", err)
	}

	ad := &authhandler.AuthData{
		KeycloakUserID: u.KeycloakUserID,
		Email:          u.Email,
		Role:           u.Role,
	}

	_, err := resolveCurrentUser(ctx(), ad)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}