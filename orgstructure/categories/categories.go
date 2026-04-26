package categories

import (
	"context"
	"strings"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/category"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

//encore:api auth method=POST path=/categories
func CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*GetCategoryResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleHR, authhandler.RoleADM, authhandler.RoleSA); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if err := validateCreateCategoryRequest(req); err != nil {
		return nil, err
	}

	c, err := createCategory(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetCategoryResponse{Category: c}, nil
}

//encore:api auth method=GET path=/categories
func ListCategories(ctx context.Context) (*ListCategoriesResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleHR, authhandler.RoleADM, authhandler.RoleSA); err != nil {
		return nil, err
	}

	categories, err := listCategories(ctx)
	if err != nil {
		return nil, err
	}

	return &ListCategoriesResponse{Categories: categories}, nil
}

//encore:api auth method=GET path=/categories/:category_id
func GetCategoryByID(ctx context.Context, category_id string) (*GetCategoryResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleHR, authhandler.RoleADM, authhandler.RoleSA); err != nil {
		return nil, err
	}

	c, err := getCategoryByID(ctx, category_id)
	if err != nil {
		return nil, err
	}

	return &GetCategoryResponse{Category: c}, nil
}

//encore:api auth method=POST path=/categories/by-ids
func GetCategoriesByIDs(ctx context.Context, req *GetCategoriesByIDsRequest) (*GetCategoriesByIDsResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleHR, authhandler.RoleADM, authhandler.RoleSA); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if len(req.CategoryIDs) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("category_ids cannot be empty").Err()
	}

	ids, err := parseCategoryIDs(req.CategoryIDs)
	if err != nil {
		return nil, err
	}

	categories, err := getCategoriesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	return &GetCategoriesByIDsResponse{Categories: categories}, nil
}

//encore:api auth method=PATCH path=/categories/:category_id
func UpdateCategory(ctx context.Context, category_id string, req *UpdateCategoryRequest) (*GetCategoryResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleHR, authhandler.RoleADM, authhandler.RoleSA); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if err := validateUpdateCategoryRequest(req); err != nil {
		return nil, err
	}

	c, err := updateCategory(ctx, category_id, req)
	if err != nil {
		return nil, err
	}

	return &GetCategoryResponse{Category: c}, nil
}

//encore:api auth method=DELETE path=/categories/:category_id
func DeleteCategory(ctx context.Context, category_id string) error {
	ad, err := getAuthData()
	if err != nil {
		return err
	}
	if err := requireRole(ad, authhandler.RoleHR, authhandler.RoleADM, authhandler.RoleSA); err != nil {
		return err
	}

	return deleteCategory(ctx, category_id)
}

func validateCreateCategoryRequest(req *CreateCategoryRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("name is required").Err()
	}
	if len(name) > 100 {
		return errs.B().Code(errs.InvalidArgument).Msg("name is too long").Err()
	}
	return nil
}

func validateUpdateCategoryRequest(req *UpdateCategoryRequest) error {
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return errs.B().Code(errs.InvalidArgument).Msg("name cannot be empty").Err()
		}
		if len(name) > 100 {
			return errs.B().Code(errs.InvalidArgument).Msg("name is too long").Err()
		}
	}
	return nil
}

func createCategory(ctx context.Context, req *CreateCategoryRequest) (*Category, error) {
	builder := Client.Category.
		Create().
		SetName(strings.TrimSpace(req.Name))

	if req.Description != nil {
		builder = builder.SetDescription(strings.TrimSpace(*req.Description))
	}

	row, err := builder.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create category").Cause(err).Err()
	}

	return entToCategory(row), nil
}

func listCategories(ctx context.Context) ([]*Category, error) {
	rows, err := Client.Category.
		Query().
		Order(ent.Asc(category.FieldName)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list categories").Cause(err).Err()
	}

	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, entToCategory(row))
	}

	return categories, nil
}

func getCategoryByID(ctx context.Context, categoryID string) (*Category, error) {
	id, err := parseCategoryID(categoryID)
	if err != nil {
		return nil, err
	}

	row, err := Client.Category.
		Query().
		Where(category.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("category not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get category").Cause(err).Err()
	}

	return entToCategory(row), nil
}

func getCategoriesByIDs(ctx context.Context, ids []uuid.UUID) ([]*Category, error) {
	rows, err := Client.Category.
		Query().
		Where(category.IDIn(ids...)).
		Order(ent.Asc(category.FieldName)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to get categories").Cause(err).Err()
	}

	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, entToCategory(row))
	}

	return categories, nil
}

func updateCategory(ctx context.Context, categoryID string, req *UpdateCategoryRequest) (*Category, error) {
	id, err := parseCategoryID(categoryID)
	if err != nil {
		return nil, err
	}

	builder := Client.Category.UpdateOneID(id)
	if req.Name != nil {
		builder = builder.SetName(strings.TrimSpace(*req.Name))
	}
	if req.Description != nil {
		builder = builder.SetDescription(strings.TrimSpace(*req.Description))
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("category not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update category").Cause(err).Err()
	}

	return entToCategory(row), nil
}

func deleteCategory(ctx context.Context, categoryID string) error {
	id, err := parseCategoryID(categoryID)
	if err != nil {
		return err
	}

	err = Client.Category.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errs.B().Code(errs.NotFound).Msg("category not found").Err()
		}
		return errs.B().Code(errs.Internal).Msg("failed to delete category").Cause(err).Err()
	}

	return nil
}

func parseCategoryID(categoryID string) (uuid.UUID, error) {
	id, err := uuid.Parse(categoryID)
	if err != nil {
		return uuid.Nil, errs.B().Code(errs.InvalidArgument).Msg("invalid category_id format").Err()
	}
	return id, nil
}

func parseCategoryIDs(categoryIDs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		id, err := parseCategoryID(categoryID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func entToCategory(row *ent.Category) *Category {
	if row == nil {
		return nil
	}
	return &Category{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
	}
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func requireRole(ad *authhandler.AuthData, allowed ...authhandler.UserRole) error {
	for _, role := range allowed {
		if ad.Role == role {
			return nil
		}
	}
	return errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
}
