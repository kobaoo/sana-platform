package users

// Thin wrappers around encore.app/orgstructure/users/policy so the users
// package can call rule functions without an extra import alias.

import (
	"encore.dev/beta/errs"

	"encore.app/auth/authhandler"
	"encore.app/orgstructure/users/policy"
)

func CanViewUsers(role authhandler.UserRole) bool {
	return policy.CanViewUsers(role)
}

func CanManageUsers(role authhandler.UserRole) bool {
	return policy.CanManageUsers(role)
}

func CanAssignRole(callerRole, targetRole authhandler.UserRole) bool {
	return policy.CanAssignRole(callerRole, targetRole)
}

func CanAccessUser(callerRole authhandler.UserRole, callerDzoID, targetDzoID *string) bool {
	return policy.CanAccessUser(callerRole, callerDzoID, targetDzoID)
}

func AutoProvisionRole() authhandler.UserRole {
	return policy.AutoProvisionRole()
}

// CheckUserAccess converts policy.ErrUserBlocked into an Encore PermissionDenied error.
func CheckUserAccess(u *User) error {
	if err := policy.CheckUserAccess(u.IsActive); err != nil {
		return errs.B().Code(errs.PermissionDenied).Msg(err.Error()).Err()
	}
	return nil
}
