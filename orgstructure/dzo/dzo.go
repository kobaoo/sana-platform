package dzo

import (
	"context"
	"strings"
	"time"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"encore.app/db/ent"
	"encore.app/db/ent/dzoorganization"
	"encore.app/db/ent/employee"
)

// ════ DATABASE ════

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

// ════ ENDPOINTS ════

// CreateDZO creates a new DZO.
//
//encore:api auth method=POST path=/dzo
func CreateDZO(ctx context.Context, req *CreateDZORequest) (*GetDZOResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Name) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("name is required").Err()
	}
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("client_id is required").Err()
	}

	dzo, err := createDZO(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetDZOResponse{DZO: *dzo}, nil
}

// ListDZO returns all active DZO.
//
//encore:api auth method=GET path=/dzo
func ListDZO(ctx context.Context) (*ListDZOResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	dzos, err := queryAllDZO(ctx)
	if err != nil {
		return nil, err
	}

	return &ListDZOResponse{
		DZOs:  dzos,
		Total: len(dzos),
	}, nil
}

// GetDZO returns DZO by ID.
//
//encore:api auth method=GET path=/dzo/:id
func GetDZO(ctx context.Context, id string) (*GetDZOResponse, error) {

	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	dzo, err := queryDZOByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetDZOResponse{DZO: *dzo}, nil
}

// UpdateDZO updates DZO.
//
//encore:api auth method=PATCH path=/dzo/:id
func UpdateDZO(ctx context.Context, id string, req *UpdateDZORequest) (*GetDZOResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	dzo, err := updateDZO(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetDZOResponse{DZO: *dzo}, nil
}

// DeleteDZO soft deletes DZO.
//
//encore:api auth method=DELETE path=/dzo/:id
func DeleteDZO(ctx context.Context, id string) (*DeleteDZOResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	count, err := deleteDZO(ctx, id)
	if err != nil {
		return nil, err
	}

	return &DeleteDZOResponse{
		Message:        "dzo deleted",
		EmployeesCount: count,
	}, nil
}

// ════ INTERNAL ════

func createDZO(ctx context.Context, req *CreateDZORequest) (*DZO, error) {
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client_id").Err()
	}

	normalizedName := strings.TrimSpace(req.Name)
	if normalizedName == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("name is required").Err()
	}

	// uniqueness check
	exists, err := Client.DzoOrganization.
		Query().
		Where(dzoorganization.NameEQ(normalizedName)).
		Exist(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}
	if exists {
		return nil, errs.B().Code(errs.AlreadyExists).Msg("dzo already exists").Err()
	}

	row, err := Client.DzoOrganization.
		Create().
		SetID(uuid.New()).
		SetClientID(clientID).
		SetName(normalizedName).
		SetNillableShortName(req.ShortName).
		SetNillableBin(req.BIN).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}

	return entToDZO(row, 0), nil
}

func queryActiveDZO(ctx context.Context) ([]DZO, error) {
	rows, err := Client.DzoOrganization.
		Query().
		Where(dzoorganization.IsActiveEQ(true)).
		All(ctx)

	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}

	res := make([]DZO, 0, len(rows))
	for _, r := range rows {
		res = append(res, *entToDZO(r, 0))
	}

	return res, nil
}

func queryAllDZO(ctx context.Context) ([]DZO, error) {
	rows, err := Client.DzoOrganization.
		Query().
		WithEmployees(func(q *ent.EmployeeQuery) {
			q.Where(employee.IsDeletedEQ(false))
		}).
		All(ctx)

	if err != nil {
		return nil, errs.B().Code(errs.Internal).Err()
	}

	res := make([]DZO, 0, len(rows))
	for _, r := range rows {
		employeesCount := len(r.Edges.Employees)
		res = append(res, *entToDZO(r, employeesCount))
	}

	return res, nil
}

func queryDZOByID(ctx context.Context, id string) (*DZO, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Err()
	}

	row, err := Client.DzoOrganization.
		Query().
		Where(dzoorganization.IDEQ(uid)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Err()
		}
		return nil, errs.B().Code(errs.Internal).Err()
	}

	empCount, _ := Client.Employee.
		Query().
		Where(employee.DzoIDEQ(uid), employee.IsDeletedEQ(false)).
		Count(ctx)

	return entToDZO(row, empCount), nil
}

func updateDZO(ctx context.Context, id string, req *UpdateDZORequest) (*DZO, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Err()
	}

	builder := Client.DzoOrganization.UpdateOneID(uid).SetUpdatedAt(time.Now())

	if req.Name != nil {
		normalizedName := strings.TrimSpace(*req.Name)
		if normalizedName == "" {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("name is required").Err()
		}

		exists, err := Client.DzoOrganization.
			Query().
			Where(
				dzoorganization.NameEQ(normalizedName),
				dzoorganization.IDNEQ(uid),
			).
			Exist(ctx)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Err()
		}
		if exists {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("dzo already exists").Err()
		}
		builder.SetName(normalizedName)
	}
	if req.ShortName != nil {
		builder.SetShortName(*req.ShortName)
	}
	if req.BIN != nil {
		builder.SetBin(*req.BIN)
	}
	if req.IsActive != nil {
		builder.SetIsActive(*req.IsActive)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Err()
		}
		return nil, errs.B().Code(errs.Internal).Err()
	}

	empCount, _ := Client.Employee.
		Query().
		Where(employee.DzoIDEQ(uid), employee.IsDeletedEQ(false)).
		Count(ctx)

	return entToDZO(row, empCount), nil
}

func deleteDZO(ctx context.Context, id string) (int, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return 0, errs.B().Code(errs.InvalidArgument).Err()
	}

	count, err := Client.Employee.
		Query().
		Where(employee.DzoIDEQ(uid)).
		Count(ctx)
	if err != nil {
		return 0, errs.B().Code(errs.Internal).Err()
	}

	if count > 0 {
		return count, errs.B().
			Code(errs.FailedPrecondition).
			Msg("cannot delete dzo with employees").
			Err()
	}

	err = Client.DzoOrganization.
		UpdateOneID(uid).
		SetIsActive(false).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return 0, errs.B().Code(errs.NotFound).Err()
		}
		return 0, errs.B().Code(errs.Internal).Err()
	}

	return count, nil
}

func requireRole(ad *authhandler.AuthData, allowed ...authhandler.UserRole) error {
	for _, r := range allowed {
		if ad.Role == r {
			return nil
		}
	}
	return errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

// helper

func entToDZO(e *ent.DzoOrganization, employeesCount int) *DZO {
	return &DZO{
		ID:             e.ID.String(),
		ClientID:       e.ClientID.String(),
		Name:           e.Name,
		ShortName:      e.ShortName,
		BIN:            e.Bin,
		IsActive:       e.IsActive,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
		EmployeesCount: employeesCount,
	}
}
