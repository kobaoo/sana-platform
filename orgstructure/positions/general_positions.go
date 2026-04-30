package positions

import (
	"context"
	"strings"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/dzopositiontitle"
	"encore.app/db/ent/generalposition"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

// DATABASE
var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

// CreateGeneralPosition creates a new general position.
//
//encore:api auth method=POST path=/general-positions
func CreateGeneralPosition(ctx context.Context, req *CreateGeneralPositionRequest) (*GetGeneralPositionResponse, error) {
	if err := requireRole(authhandler.RoleSA); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Name) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("name is required").Err()
	}

	pos, err := insertGeneralPosition(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetGeneralPositionResponse{GeneralPosition: *pos}, nil
}

// ListGeneralPositions returns all not deleted general positions. SA, ADM  only.
//
//encore:api auth method=GET path=/general-positions
func ListGeneralPositions(ctx context.Context, params *ListGeneralPositionsRequest) (*ListGeneralPositionsResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	positions, err := queryGeneralPositions(ctx, params)
	if err != nil {
		return nil, err
	}

	return &ListGeneralPositionsResponse{
		GeneralPositions: positions,
		Total:            len(positions),
	}, nil
}

// GetGeneralPosition returns a single general position by ID. SA, ADM only.
//
//encore:api auth method=GET path=/general-positions/:id
func GetGeneralPosition(ctx context.Context, id string) (*GetGeneralPositionResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	pos, err := queryGeneralPosition(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetGeneralPositionResponse{GeneralPosition: *pos}, nil
}

// UpdateGeneralPosition partially updates a general position. SA only.
//
//encore:api auth method=PATCH path=/general-positions/:id
func UpdateGeneralPosition(ctx context.Context, id string, req *UpdateGeneralPositionRequest) (*GetGeneralPositionResponse, error) {
	if err := requireRole(authhandler.RoleSA); err != nil {
		return nil, err
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("name cannot be empty").Err()
	}

	pos, err := patchGeneralPosition(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetGeneralPositionResponse{GeneralPosition: *pos}, nil
}

// DeleteGeneralPosition soft-deletes a general position. SA
//
//encore:api auth method=DELETE path=/general-positions/:id
func DeleteGeneralPosition(ctx context.Context, id string) (*DeleteGeneralPositionResponse, error) {
	if err := requireRole(authhandler.RoleSA); err != nil {
		return nil, err
	}

	if err := deleteGeneralPosition(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteGeneralPositionResponse{
		Message: "general position deleted successfully",
	}, nil
}

// INTERNAL
func insertGeneralPosition(ctx context.Context, req *CreateGeneralPositionRequest) (*GeneralPosition, error) {
	exists, err := Client.GeneralPosition.
		Query().
		Where(
			generalposition.Name(req.Name),
			generalposition.IsDeletedEQ(false)).
		Exist(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to get existing general position").Err()
	}
	if exists {
		return nil, errs.B().Code(errs.AlreadyExists).Msg("general position already exists").Err()
	}
	builder := Client.GeneralPosition.
		Create().
		SetName(req.Name)

	if req.Description != nil {
		builder.SetDescription(*req.Description)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create general position").Err()
	}
	return entToGeneralPosition(row), nil
}
func queryGeneralPositions(ctx context.Context, req *ListGeneralPositionsRequest) ([]GeneralPosition, error) {
	query := Client.GeneralPosition.
		Query().
		Where(generalposition.IsDeletedEQ(false))

	if req.Search != "" {
		query.Where(
			generalposition.Or(
				generalposition.NameContains(req.Search),
				generalposition.DescriptionContains(req.Search),
			),
		)
	}

	rows, err := query.
		Order(ent.Asc(generalposition.FieldName)).
		All(ctx)

	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query general positions").Err()
	}
	pos := make([]GeneralPosition, 0, len(rows))
	for _, row := range rows {
		pos = append(pos, *entToGeneralPosition(row))
	}
	return pos, nil
}

func queryGeneralPosition(ctx context.Context, id string) (*GeneralPosition, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid general position id").Err()
	}
	row, err := Client.GeneralPosition.
		Query().
		Where(generalposition.ID(uid),
			generalposition.IsDeletedEQ(false)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, errs.B().Code(errs.NotFound).Msg("general position not found").Err()
	}
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query general position").Err()
	}
	return entToGeneralPosition(row), nil
}

func patchGeneralPosition(ctx context.Context, id string, req *UpdateGeneralPositionRequest) (*GeneralPosition, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid general position id").Err()
	}
	if req.Name == nil && req.Description == nil {
		return queryGeneralPosition(ctx, id)
	}
	builder := Client.GeneralPosition.
		Update().
		Where(
			generalposition.ID(uid),
			generalposition.IsDeletedEQ(false),
		)

	if req.Name != nil {
		exists, err := Client.GeneralPosition.Query().Where(generalposition.Name(*req.Name)).Exist(ctx)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to get existing general position").Err()
		}
		if exists {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("general position already exists").Err()
		}
		builder.SetName(*req.Name)
	}

	if req.Description != nil {
		builder.SetDescription(*req.Description)
	}

	count, err := builder.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update general position").Err()
	}
	if count == 0 {
		return nil, errs.B().Code(errs.NotFound).Msg("general position not found").Err()
	}

	return queryGeneralPosition(ctx, id)
}

func deleteGeneralPosition(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid general position id").Err()
	}
	countDpt, err := Client.DzoPositionTitle.
		Query().
		Where(dzopositiontitle.GeneralPositionIDEQ(uid)).
		Count(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to query general position").Err()
	}
	if countDpt > 0 {
		return errs.B().Code(errs.AlreadyExists).Msg("general position has linked dzo position titles").Err()
	}
	count, err := Client.GeneralPosition.
		Update().
		Where(
			generalposition.ID(uid),
			generalposition.IsDeletedEQ(false),
		).
		SetIsDeleted(true).
		Save(ctx)

	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete general position").Err()
	}
	if count == 0 {
		return errs.B().Code(errs.NotFound).Msg("general position not found").Err()
	}
	return nil
}

// ════ AUTH HELPERS ════

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func requireRole(allowed ...authhandler.UserRole) error {
	ad, err := getAuthData()
	if err != nil {
		return err
	}
	for _, r := range allowed {
		if ad.Role == r {
			return nil
		}
	}
	return errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
}

//Helpers

func entToGeneralPosition(row *ent.GeneralPosition) *GeneralPosition {
	return &GeneralPosition{
		ID:          row.ID.String(),
		Description: row.Description,
		Name:        row.Name,
		IsDeleted:   row.IsDeleted,
		CreatedAt:   row.CreatedAt,
	}
}
