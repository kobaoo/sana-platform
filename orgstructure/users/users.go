package users

import (
	"context"
	"strings"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/user"
)

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

// GetMe returns the current user, auto-provisioning as EMP on first login.
//
//encore:api auth method=GET path=/users/me
func GetMe(ctx context.Context) (*GetUserResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	u, err := resolveCurrentUser(ctx, ad)
	if err != nil {
		return nil, err
	}
	return &GetUserResponse{User: *u}, nil
}

// GetUser returns a user by internal ID. SA and ADM only; ADM is scoped to own DZO.
//
//encore:api auth method=GET path=/users/id/:id
func GetUser(ctx context.Context, id string) (*GetUserResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanViewUsers(caller.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	u, err := queryUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanAccessUser(caller.Role, caller.DzoID, u.DzoID) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("user is outside your DZO").Err()
	}
	return &GetUserResponse{User: *u}, nil
}

// ListUsers returns active users. SA sees all; ADM sees only their DZO.
//
//encore:api auth method=GET path=/users
func ListUsers(ctx context.Context) (*ListUsersResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanViewUsers(caller.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	var users []User
	if caller.Role == authhandler.RoleSA {
		users, err = queryActiveUsers(ctx)
	} else {
		if caller.DzoID == nil {
			users = []User{}
		} else {
			users, err = queryActiveUsersByDzo(ctx, *caller.DzoID)
		}
	}
	if err != nil {
		return nil, err
	}
	return &ListUsersResponse{Users: users, Total: len(users)}, nil
}

// CreateUser creates a user linked to an existing Keycloak identity. SA only.
//
//encore:api auth method=POST path=/users
func CreateUser(ctx context.Context, req *CreateUserRequest) (*GetUserResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if caller.Role != authhandler.RoleSA {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only SA can create users").Err()
	}
	if strings.TrimSpace(req.KeycloakUserID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("keycloak_user_id is required").Err()
	}
	if strings.TrimSpace(req.Email) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("email is required").Err()
	}
	if !req.Role.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid role").Err()
	}
	u, err := insertUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return &GetUserResponse{User: *u}, nil
}

// RegisterAdmin pre-creates an admin record (is_active=false, is_onboarded=false).
// The record is auto-activated when the admin logs in for the first time. SA only.
//
//encore:api auth method=POST path=/users/register-admin
func RegisterAdmin(ctx context.Context, req *RegisterAdminRequest) (*GetUserResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if caller.Role != authhandler.RoleSA {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only SA can register admins").Err()
	}
	if strings.TrimSpace(req.KeycloakUserID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("keycloak_user_id is required").Err()
	}
	if strings.TrimSpace(req.Email) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("email is required").Err()
	}
	if req.DzoID == nil || strings.TrimSpace(*req.DzoID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id is required for admin registration").Err()
	}
	if _, err := uuid.Parse(*req.DzoID); err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}
	u, err := insertPendingAdmin(ctx, req)
	if err != nil {
		return nil, err
	}
	return &GetUserResponse{User: *u}, nil
}

// AssignRole assigns a role to a user. SA can assign any role; ADM can assign HR only within own DZO.
//
//encore:api auth method=PUT path=/users/id/:id/assign-role
func AssignRole(ctx context.Context, id string, req *AssignRoleRequest) (*GetUserResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanAssignRole(caller.Role, req.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions to assign this role").Err()
	}
	if !req.Role.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid role").Err()
	}
	target, err := queryUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanAccessUser(caller.Role, caller.DzoID, target.DzoID) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("user is outside your DZO").Err()
	}
	if req.Role == authhandler.RoleHR && req.DzoID == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id is required when assigning HR role").Err()
	}
	u, err := updateUserRole(ctx, id, req.Role, req.DzoID)
	if err != nil {
		return nil, err
	}
	// Best-effort: DB is source of truth; Keycloak errors are non-fatal.
	syncRoleToKeycloak(ctx, target.KeycloakUserID, req.Role)
	return &GetUserResponse{User: *u}, nil
}

// RemoveRole resets a user's role to EMP.
//
//encore:api auth method=PUT path=/users/id/:id/remove-role
func RemoveRole(ctx context.Context, id string) (*GetUserResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanManageUsers(caller.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	target, err := queryUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanAccessUser(caller.Role, caller.DzoID, target.DzoID) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("user is outside your DZO").Err()
	}
	u, err := updateUserRole(ctx, id, authhandler.RoleEMP, nil)
	if err != nil {
		return nil, err
	}
	syncRoleToKeycloak(ctx, target.KeycloakUserID, authhandler.RoleEMP)
	return &GetUserResponse{User: *u}, nil
}

// BlockUser sets is_active=false. SA and ADM; ADM scoped to own DZO.
//
//encore:api auth method=PUT path=/users/id/:id/block
func BlockUser(ctx context.Context, id string) (*MessageResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanManageUsers(caller.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	target, err := queryUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanAccessUser(caller.Role, caller.DzoID, target.DzoID) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("user is outside your DZO").Err()
	}
	if err := updateUserActive(ctx, id, false); err != nil {
		return nil, err
	}
	syncEnabledToKeycloak(ctx, target.KeycloakUserID, false)
	return &MessageResponse{Message: "user blocked successfully"}, nil
}

// UnblockUser sets is_active=true. SA only.
//
//encore:api auth method=PUT path=/users/id/:id/unblock
func UnblockUser(ctx context.Context, id string) (*MessageResponse, error) {
	caller, err := resolveAndCheckCaller(ctx)
	if err != nil {
		return nil, err
	}
	if caller.Role != authhandler.RoleSA {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only SA can unblock users").Err()
	}
	target, err := queryUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := updateUserActive(ctx, id, true); err != nil {
		return nil, err
	}
	syncEnabledToKeycloak(ctx, target.KeycloakUserID, true)
	return &MessageResponse{Message: "user unblocked successfully"}, nil
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

// resolveCurrentUser finds or auto-provisions the caller, then enforces the active check.
// Pending admins (is_onboarded=false, is_active=false) are auto-activated on first login.
func resolveCurrentUser(ctx context.Context, ad *authhandler.AuthData) (*User, error) {
	u, err := queryUserByKeycloakID(ctx, ad.KeycloakUserID)
	if err != nil {
		if errs.Code(err) != errs.NotFound {
			return nil, err
		}
		u, err = autoProvision(ctx, ad)
		if err != nil {
			return nil, err
		}
	}

	if IsPendingActivation(u.IsOnboarded, u.IsActive) {
		u, err = activateOnboarding(ctx, u.ID)
		if err != nil {
			return nil, err
		}
	}

	if err := CheckUserAccess(u); err != nil {
		return nil, err
	}
	return u, nil
}

func resolveAndCheckCaller(ctx context.Context) (*User, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	return resolveCurrentUser(ctx, ad)
}

// autoProvision creates a new EMP user from JWT claims.
// SA is the only exception: a fresh SA must be trusted from the JWT to
// bootstrap the system — SA can only be granted by a Keycloak realm admin.
func autoProvision(ctx context.Context, ad *authhandler.AuthData) (*User, error) {
	role := AutoProvisionRole()
	if ad.Role == authhandler.RoleSA {
		role = authhandler.RoleSA
	}

	req := &CreateUserRequest{
		KeycloakUserID: ad.KeycloakUserID,
		Email:          ad.Email,
		Role:           role,
	}
	if ad.DzoID != "" {
		req.DzoID = &ad.DzoID
	}
	return insertUser(ctx, req)
}

func insertUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	builder := Client.User.
		Create().
		SetKeycloakUserID(req.KeycloakUserID).
		SetEmail(req.Email).
		SetRole(string(req.Role))

	if req.DzoID != nil {
		dzoUUID, err := uuid.Parse(*req.DzoID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
		builder = builder.SetDzoID(dzoUUID)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("user with this keycloak_user_id already exists").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to create user").Cause(err).Err()
	}
	return entToUser(row), nil
}

func insertPendingAdmin(ctx context.Context, req *RegisterAdminRequest) (*User, error) {
	dzoUUID, err := uuid.Parse(*req.DzoID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}

	row, err := Client.User.
		Create().
		SetKeycloakUserID(req.KeycloakUserID).
		SetEmail(req.Email).
		SetRole(string(authhandler.RoleADM)).
		SetDzoID(dzoUUID).
		SetIsActive(false).
		SetIsOnboarded(false).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("user with this keycloak_user_id already exists").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to create admin user").Cause(err).Err()
	}
	return entToUser(row), nil
}

func activateOnboarding(ctx context.Context, id string) (*User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row, err := Client.User.
		UpdateOneID(uid).
		SetIsActive(true).
		SetIsOnboarded(true).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("user not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to activate user").Cause(err).Err()
	}
	return entToUser(row), nil
}

func queryUserByID(ctx context.Context, id string) (*User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row, err := Client.User.
		Query().
		Where(user.IDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("user not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get user").Cause(err).Err()
	}
	return entToUser(row), nil
}

func queryUserByKeycloakID(ctx context.Context, kcID string) (*User, error) {
	row, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(kcID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("user not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get user by keycloak id").Cause(err).Err()
	}
	return entToUser(row), nil
}

func queryActiveUsers(ctx context.Context) ([]User, error) {
	rows, err := Client.User.
		Query().
		Where(user.IsActiveEQ(true)).
		Order(ent.Asc(user.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list users").Cause(err).Err()
	}
	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, *entToUser(row))
	}
	return users, nil
}

func queryActiveUsersByDzo(ctx context.Context, dzoID string) ([]User, error) {
	dzoUUID, err := uuid.Parse(dzoID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}

	rows, err := Client.User.
		Query().
		Where(
			user.IsActiveEQ(true),
			user.DzoIDEQ(dzoUUID),
		).
		Order(ent.Asc(user.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list users by dzo").Cause(err).Err()
	}
	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, *entToUser(row))
	}
	return users, nil
}

func updateUserRole(ctx context.Context, id string, role authhandler.UserRole, dzoID *string) (*User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	builder := Client.User.UpdateOneID(uid).SetRole(string(role))
	if dzoID != nil {
		dzoUUID, err := uuid.Parse(*dzoID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
		builder = builder.SetDzoID(dzoUUID)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("user not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update user role").Cause(err).Err()
	}
	return entToUser(row), nil
}

func updateUserActive(ctx context.Context, id string, active bool) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	err = Client.User.
		UpdateOneID(uid).
		SetIsActive(active).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errs.B().Code(errs.NotFound).Msg("user not found").Err()
		}
		return errs.B().Code(errs.Internal).Msg("failed to update user status").Cause(err).Err()
	}
	return nil
}

func entToUser(e *ent.User) *User {
	var dzoID *string
	if e.DzoID != nil {
		s := e.DzoID.String()
		dzoID = &s
	}
	return &User{
		ID:             e.ID.String(),
		KeycloakUserID: e.KeycloakUserID,
		Email:          e.Email,
		Role:           authhandler.UserRole(e.Role),
		DzoID:          dzoID,
		IsActive:       e.IsActive,
		IsOnboarded:    e.IsOnboarded,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}
