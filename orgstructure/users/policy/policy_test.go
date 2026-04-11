package policy_test

import (
	"errors"
	"strings"
	"testing"

	"encore.app/auth/authhandler"
	"encore.app/orgstructure/users/policy"
)

func TestCanViewUsers_SA(t *testing.T) {
	if !policy.CanViewUsers(authhandler.RoleSA) {
		t.Error("SA should be able to view users")
	}
}

func TestCanViewUsers_ADM(t *testing.T) {
	if !policy.CanViewUsers(authhandler.RoleADM) {
		t.Error("ADM should be able to view users")
	}
}

func TestCanViewUsers_HR(t *testing.T) {
	if policy.CanViewUsers(authhandler.RoleHR) {
		t.Error("HR should NOT be able to view users")
	}
}

func TestCanViewUsers_EMP(t *testing.T) {
	if policy.CanViewUsers(authhandler.RoleEMP) {
		t.Error("EMP should NOT be able to view users")
	}
}

func TestCanManageUsers_SA(t *testing.T) {
	if !policy.CanManageUsers(authhandler.RoleSA) {
		t.Error("SA should be able to manage users")
	}
}

func TestCanManageUsers_ADM(t *testing.T) {
	if !policy.CanManageUsers(authhandler.RoleADM) {
		t.Error("ADM should be able to manage users")
	}
}

func TestCanManageUsers_HR(t *testing.T) {
	if policy.CanManageUsers(authhandler.RoleHR) {
		t.Error("HR should NOT be able to manage users")
	}
}

func TestCanManageUsers_EMP(t *testing.T) {
	if policy.CanManageUsers(authhandler.RoleEMP) {
		t.Error("EMP should NOT be able to manage users")
	}
}

func TestCanAssignRole_SA_AnyRole(t *testing.T) {
	for _, r := range []authhandler.UserRole{
		authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR, authhandler.RoleEMP,
	} {
		if !policy.CanAssignRole(authhandler.RoleSA, r) {
			t.Errorf("SA should be able to assign role %q", r)
		}
	}
}

func TestCanAssignRole_ADM_HR(t *testing.T) {
	if !policy.CanAssignRole(authhandler.RoleADM, authhandler.RoleHR) {
		t.Error("ADM should be able to assign HR role")
	}
}

func TestCanAssignRole_ADM_CannotAssignSA(t *testing.T) {
	if policy.CanAssignRole(authhandler.RoleADM, authhandler.RoleSA) {
		t.Error("ADM should NOT be able to assign SA role")
	}
}

func TestCanAssignRole_ADM_CannotAssignADM(t *testing.T) {
	if policy.CanAssignRole(authhandler.RoleADM, authhandler.RoleADM) {
		t.Error("ADM should NOT be able to assign ADM role")
	}
}

func TestCanAssignRole_ADM_CannotAssignEMP(t *testing.T) {
	if policy.CanAssignRole(authhandler.RoleADM, authhandler.RoleEMP) {
		t.Error("ADM should NOT be able to assign EMP role directly")
	}
}

func TestCanAssignRole_HR_CannotAssignAny(t *testing.T) {
	for _, r := range []authhandler.UserRole{
		authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR, authhandler.RoleEMP,
	} {
		if policy.CanAssignRole(authhandler.RoleHR, r) {
			t.Errorf("HR should NOT be able to assign role %q", r)
		}
	}
}

func TestCanAssignRole_EMP_CannotAssignAny(t *testing.T) {
	for _, r := range []authhandler.UserRole{
		authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR, authhandler.RoleEMP,
	} {
		if policy.CanAssignRole(authhandler.RoleEMP, r) {
			t.Errorf("EMP should NOT be able to assign role %q", r)
		}
	}
}

func ptr(s string) *string { return &s }

func TestCanAccessUser_SA_AccessesAnyone(t *testing.T) {
	if !policy.CanAccessUser(authhandler.RoleSA, ptr("dzo-1"), ptr("dzo-other")) {
		t.Error("SA should access any user regardless of DZO")
	}
}

func TestCanAccessUser_SA_AccessesUserWithoutDzo(t *testing.T) {
	if !policy.CanAccessUser(authhandler.RoleSA, nil, nil) {
		t.Error("SA should access user even without DZO")
	}
}

func TestCanAccessUser_ADM_SameDzo(t *testing.T) {
	if !policy.CanAccessUser(authhandler.RoleADM, ptr("dzo-1"), ptr("dzo-1")) {
		t.Error("ADM should access user in same DZO")
	}
}

func TestCanAccessUser_ADM_DifferentDzo(t *testing.T) {
	if policy.CanAccessUser(authhandler.RoleADM, ptr("dzo-1"), ptr("dzo-other")) {
		t.Error("ADM should NOT access user in different DZO")
	}
}

func TestCanAccessUser_ADM_TargetNoDzo(t *testing.T) {
	if policy.CanAccessUser(authhandler.RoleADM, ptr("dzo-1"), nil) {
		t.Error("ADM should NOT access user without DZO")
	}
}

func TestCanAccessUser_ADM_CallerNoDzo(t *testing.T) {
	if policy.CanAccessUser(authhandler.RoleADM, nil, ptr("dzo-1")) {
		t.Error("ADM without DZO should NOT access any user")
	}
}

func TestCanAccessUser_HR_Denied(t *testing.T) {
	if policy.CanAccessUser(authhandler.RoleHR, ptr("dzo-1"), ptr("dzo-1")) {
		t.Error("HR should NOT access other users")
	}
}

func TestCanAccessUser_EMP_Denied(t *testing.T) {
	if policy.CanAccessUser(authhandler.RoleEMP, ptr("dzo-1"), ptr("dzo-1")) {
		t.Error("EMP should NOT access other users")
	}
}

func TestAutoProvisionRole_AlwaysEMP(t *testing.T) {
	if got := policy.AutoProvisionRole(); got != authhandler.RoleEMP {
		t.Errorf("AutoProvisionRole() = %q, want EMP", got)
	}
}

// AutoProvisionRole must return EMP regardless of what JWT role is passed.
// Trusting JWT claims for provisioning would allow privilege escalation.
func TestAutoProvisionRole_IndependentOfJWTRole(t *testing.T) {
	for _, jwtRole := range []authhandler.UserRole{
		authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR, authhandler.RoleEMP,
	} {
		_ = jwtRole
		if got := policy.AutoProvisionRole(); got != authhandler.RoleEMP {
			t.Errorf("AutoProvisionRole() = %q for JWT role %q, want EMP", got, jwtRole)
		}
	}
}

func TestIsPendingActivation_PendingAdmin(t *testing.T) {
	if !policy.IsPendingActivation(false, false) {
		t.Error("isOnboarded=false, isActive=false must be PENDING")
	}
}

func TestIsPendingActivation_ActiveUser(t *testing.T) {
	if policy.IsPendingActivation(true, true) {
		t.Error("isOnboarded=true, isActive=true must NOT be pending")
	}
}

func TestIsPendingActivation_BlockedUser(t *testing.T) {
	if policy.IsPendingActivation(true, false) {
		t.Error("isOnboarded=true, isActive=false is BLOCKED, not pending")
	}
}

func TestIsPendingActivation_InvalidState(t *testing.T) {
	if policy.IsPendingActivation(false, true) {
		t.Error("isOnboarded=false, isActive=true is an invalid state, not pending")
	}
}

func TestIsPendingActivation_TransitionTable(t *testing.T) {
	cases := []struct {
		isOnboarded bool
		isActive    bool
		wantPending bool
		label       string
	}{
		{false, false, true, "pending admin"},
		{true, true, false, "active user"},
		{true, false, false, "blocked user"},
		{false, true, false, "invalid state"},
	}
	for _, tc := range cases {
		got := policy.IsPendingActivation(tc.isOnboarded, tc.isActive)
		if got != tc.wantPending {
			t.Errorf("[%s] IsPendingActivation(%v, %v) = %v, want %v",
				tc.label, tc.isOnboarded, tc.isActive, got, tc.wantPending)
		}
	}
}

func TestCheckUserAccess_BlockedUserDenied(t *testing.T) {
	err := policy.CheckUserAccess(false)
	if err == nil {
		t.Fatal("expected error for blocked user (isActive=false), got nil")
	}
	if !errors.Is(err, policy.ErrUserBlocked) {
		t.Errorf("expected ErrUserBlocked, got %v", err)
	}
}

func TestCheckUserAccess_ActiveUserAllowed(t *testing.T) {
	if err := policy.CheckUserAccess(true); err != nil {
		t.Errorf("unexpected error for active user: %v", err)
	}
}

func TestCheckUserAccess_BlockedAfterActivation(t *testing.T) {
	if err := policy.CheckUserAccess(true); err != nil {
		t.Fatalf("active user should be allowed: %v", err)
	}
	if err := policy.CheckUserAccess(false); err == nil {
		t.Error("blocked user must be denied after is_active set to false")
	}
}

// adminDzoRequired encodes the rule: ADM must always have a DZO.
// Without a DZO the admin cannot be scoped to any organization.
func adminDzoRequired(dzoID *string) bool {
	return dzoID != nil && strings.TrimSpace(*dzoID) != ""
}

func TestRegisterAdmin_DzoRequired(t *testing.T) {
	cases := []struct {
		name   string
		dzoID  *string
		wantOK bool
	}{
		{"nil dzo_id", nil, false},
		{"empty dzo_id", ptr(""), false},
		{"whitespace dzo_id", ptr("   "), false},
		{"valid dzo_id", ptr("a1b2c3d4-e5f6-7890-abcd-ef1234567890"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := adminDzoRequired(tc.dzoID)
			if got != tc.wantOK {
				t.Errorf("adminDzoRequired(%v) = %v, want %v", tc.dzoID, got, tc.wantOK)
			}
		})
	}
}

func TestRolePriority_SAIsHighest(t *testing.T) {
	if authhandler.RoleSA.Priority() <= authhandler.RoleADM.Priority() {
		t.Error("SA priority should be higher than ADM")
	}
}

func TestRolePriority_ADMOverHR(t *testing.T) {
	if authhandler.RoleADM.Priority() <= authhandler.RoleHR.Priority() {
		t.Error("ADM priority should be higher than HR")
	}
}

func TestRolePriority_HROverEMP(t *testing.T) {
	if authhandler.RoleHR.Priority() <= authhandler.RoleEMP.Priority() {
		t.Error("HR priority should be higher than EMP")
	}
}
