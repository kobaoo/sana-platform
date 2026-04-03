package organizations

import (
	"context"
	"strings"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
)

// ════ DATABASE ════

var db = sqldb.NewDatabase("organizations", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

// ════ ENDPOINTS ════

// CreateOrg creates a new organization.
//
//encore:api public method=POST path=/organizations
func CreateOrg(ctx context.Context, req *CreateOrgRequest) (*GetOrgResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("name is required").Err()
	}
	if strings.TrimSpace(req.Code) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("code is required").Err()
	}
	if !req.Type.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid organization type").Err()
	}

	org, err := insertOrg(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetOrgResponse{Organization: *org}, nil
}

// ListOrgs returns all active organizations.
//
//encore:api public method=GET path=/organizations
func ListOrgs(ctx context.Context) (*ListOrgsResponse, error) {
	orgs, err := queryActiveOrgs(ctx)
	if err != nil {
		return nil, err
	}

	return &ListOrgsResponse{
		Organizations: orgs,
		Total:         len(orgs),
	}, nil
}

// GetOrg returns a single organization by ID.
//
//encore:api public method=GET path=/organizations/:id
func GetOrg(ctx context.Context, id string) (*GetOrgResponse, error) {
	org, err := queryOrgByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetOrgResponse{Organization: *org}, nil
}

// UpdateOrg partially updates an organization.
//
//encore:api public method=PUT path=/organizations/:id
func UpdateOrg(ctx context.Context, id string, req *UpdateOrgRequest) (*GetOrgResponse, error) {
	if req.Type != nil && !req.Type.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid organization type").Err()
	}

	org, err := updateOrg(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetOrgResponse{Organization: *org}, nil
}

// DeleteOrg soft-deletes an organization by setting is_active=false.
//
//encore:api public method=DELETE path=/organizations/:id
func DeleteOrg(ctx context.Context, id string) (*DeleteOrgResponse, error) {
	if err := softDeleteOrg(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteOrgResponse{Message: "organization deleted successfully"}, nil
}

// ════ INTERNAL ════

func insertOrg(ctx context.Context, req *CreateOrgRequest) (*Organization, error) {
	var org Organization
	err := db.QueryRow(ctx, `
		INSERT INTO organizations (name, code, parent_id, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, code, parent_id, type, is_active, created_at, updated_at
	`, req.Name, req.Code, req.ParentID, req.Type).Scan(
		&org.ID, &org.Name, &org.Code, &org.ParentID,
		&org.Type, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("organization with this code already exists").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to create organization").Cause(err).Err()
	}
	return &org, nil
}

func queryActiveOrgs(ctx context.Context) ([]Organization, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, code, parent_id, type, is_active, created_at, updated_at
		FROM organizations
		WHERE is_active = TRUE
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list organizations").Cause(err).Err()
	}
	defer rows.Close()

	orgs := []Organization{}
	for rows.Next() {
		var org Organization
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Code, &org.ParentID,
			&org.Type, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
		); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan organization").Cause(err).Err()
		}
		orgs = append(orgs, org)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("error iterating organizations").Cause(err).Err()
	}

	return orgs, nil
}

func queryOrgByID(ctx context.Context, id string) (*Organization, error) {
	var org Organization
	err := db.QueryRow(ctx, `
		SELECT id, name, code, parent_id, type, is_active, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, id).Scan(
		&org.ID, &org.Name, &org.Code, &org.ParentID,
		&org.Type, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if isNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("organization not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get organization").Cause(err).Err()
	}
	return &org, nil
}

func updateOrg(ctx context.Context, id string, req *UpdateOrgRequest) (*Organization, error) {
	var org Organization
	err := db.QueryRow(ctx, `
		UPDATE organizations
		SET
			name       = COALESCE($2, name),
			code       = COALESCE($3, code),
			parent_id  = COALESCE($4, parent_id),
			type       = COALESCE($5, type),
			is_active  = COALESCE($6, is_active),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, code, parent_id, type, is_active, created_at, updated_at
	`, id, req.Name, req.Code, req.ParentID, req.Type, req.IsActive).Scan(
		&org.ID, &org.Name, &org.Code, &org.ParentID,
		&org.Type, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if isNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("organization not found").Err()
		}
		if isUniqueViolation(err) {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("organization with this code already exists").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update organization").Cause(err).Err()
	}
	return &org, nil
}

func softDeleteOrg(ctx context.Context, id string) error {
	result, err := db.Exec(ctx, `
		UPDATE organizations
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`, id)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete organization").Cause(err).Err()
	}

	if n := result.RowsAffected(); n == 0 {
		return errs.B().Code(errs.NotFound).Msg("organization not found").Err()
	}

	return nil
}

func isUniqueViolation(err error) bool {
	s := err.Error()
	return strings.Contains(s, "23505") || strings.Contains(s, "unique constraint")
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "sql: no rows in result set")
}
