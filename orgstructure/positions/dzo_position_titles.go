package positions

import (
	"context"
	"strings"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/dzoorganization"
	"encore.app/db/ent/dzopositiontitle"
	"encore.dev/beta/errs"
	"github.com/google/uuid"
)

//ENDPOINTS

// CreateDzoPositionTitle creates a new local DZO-specific position title. SA and ADM only.
//
//encore:api auth method=POST path=/dzo-position-titles
func CreateDzoPositionTitle(ctx context.Context, req *CreateDzoPositionTitleRequest) (*GetDzoPositionTitleResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	clientID := ad.CompanyID

	if strings.TrimSpace(req.DzoID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id is required").Err()
	}
	dzoUID, err := uuid.Parse(req.DzoID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}
	// ADM can only create employees in DZOs within their own client.
	if ad.Role == authhandler.RoleADM {
		if err := checkDzoExistsForClient(ctx, dzoUID, ad.CompanyID); err != nil {
			return nil, err
		}
	} else {
		if err := checkDzoExists(ctx, dzoUID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(req.LocalTitle) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("local_title is required").Err()
	}
	if strings.TrimSpace(clientID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("client_id is required").Err()
	}

	title, err := insertDzoPositionTitle(ctx, req, clientID)
	if err != nil {
		return nil, err
	}

	return &GetDzoPositionTitleResponse{PositionTitle: *title}, nil
}

// ListDzoPositionTitles returns all not deleted DZO position titles. SA, ADM  only.
//
//encore:api auth method=GET path=/dzo-position-titles
func ListDzoPositionTitles(ctx context.Context, params *ListDzoPositionTitlesRequest) (*ListDzoPositionTitlesResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}
	clientId := ""
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if ad.Role == authhandler.RoleADM {
		clientId = ad.CompanyID
	}
	titles, err := queryDzoPositionTitles(ctx, params, clientId)
	if err != nil {
		return nil, err
	}

	return &ListDzoPositionTitlesResponse{
		PositionTitles: titles,
		Total:          len(titles),
	}, nil
}

// GetDzoPositionTitle returns a single DZO position title by ID. SA, ADM only.
//
//encore:api auth method=GET path=/dzo-position-titles/:id
func GetDzoPositionTitle(ctx context.Context, id string) (*GetDzoPositionTitleResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	clientID := ""
	if ad.Role == authhandler.RoleADM {
		clientID = ad.CompanyID
	}

	title, err := queryDzoPositionTitle(ctx, id, clientID)
	if err != nil {
		return nil, err
	}

	return &GetDzoPositionTitleResponse{PositionTitle: *title}, nil
}

// UpdateDzoPositionTitle partially updates a DZO position title. SA and ADM only.
//
//encore:api auth method=PATCH path=/dzo-position-titles/:id
func UpdateDzoPositionTitle(ctx context.Context, id string, req *UpdateDzoPositionTitleRequest) (*GetDzoPositionTitleResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	clientID := ""
	if ad.Role == authhandler.RoleADM {
		clientID = ad.CompanyID
	}
	if req.LocalTitle != nil && strings.TrimSpace(*req.LocalTitle) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("local_title cannot be empty").Err()
	}

	title, err := patchDzoPositionTitle(ctx, id, req, clientID)
	if err != nil {
		return nil, err
	}

	return &GetDzoPositionTitleResponse{PositionTitle: *title}, nil
}

// DeleteDzoPositionTitle soft-deletes a DZO position title. SA and ADM only.
//
//encore:api auth method=DELETE path=/dzo-position-titles/:id
func DeleteDzoPositionTitle(ctx context.Context, id string) (*DeleteDzoPositionTitleResponse, error) {
	if err := requireRole(authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}
	clientId := ""
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if ad.Role == authhandler.RoleADM {
		clientId = ad.CompanyID
	}

	if err := deleteDzoPositionTitle(ctx, id, clientId); err != nil {
		return nil, err
	}

	return &DeleteDzoPositionTitleResponse{
		Message:        "dzo position title deleted successfully",
		EmployeesCount: 0,
	}, nil
}

// INTERNAL

func insertDzoPositionTitle(ctx context.Context, req *CreateDzoPositionTitleRequest, clientID string) (*DzoPositionTitle, error) {
	dzoID, err := uuid.Parse(req.DzoID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo id").Err()
	}
	clientUid, err := uuid.Parse(clientID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client id").Err()
	}

	exists, err := Client.DzoPositionTitle.Query().
		Where(
			dzopositiontitle.DzoIDEQ(dzoID),
			dzopositiontitle.ClientID(clientUid),
			dzopositiontitle.LocalTitleEQ(req.LocalTitle),
			dzopositiontitle.IsDeletedEQ(false),
		).
		Exist(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to check existing dzo position title").Err()
	}
	if exists {
		return nil, errs.B().Code(errs.AlreadyExists).Msg("dzo position title already exists").Err()
	}

	builder := Client.DzoPositionTitle.
		Create().
		SetDzoID(dzoID).
		SetClientID(clientUid).
		SetLocalTitle(req.LocalTitle)

	if req.GeneralPositionID != nil {
		generalPositionID, err := uuid.Parse(*req.GeneralPositionID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid general position id").Err()
		}
		builder.SetGeneralPositionID(generalPositionID)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create dzo position title").Err()
	}

	return entToDzoPositionTitle(row), nil
}

func queryDzoPositionTitles(ctx context.Context, req *ListDzoPositionTitlesRequest, clientID string) ([]DzoPositionTitle, error) {
	query := Client.DzoPositionTitle.
		Query().
		Where(dzopositiontitle.IsDeletedEQ(false)).
		WithDzo().
		WithGeneralPosition()

	if clientID != "" {
		clientUid, err := uuid.Parse(clientID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client id").Err()
		}
		query = query.Where(dzopositiontitle.ClientIDEQ(clientUid))
	}

	if req != nil {
		if req.Search != "" {
			query.Where(dzopositiontitle.LocalTitleContains(req.Search))
		}

		if req.DzoID != "" {
			dzoID, err := uuid.Parse(req.DzoID)
			if err != nil {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo id").Err()
			}
			query.Where(dzopositiontitle.DzoIDEQ(dzoID))
		}

		if req.GeneralPositionID != "" {
			generalPositionID, err := uuid.Parse(req.GeneralPositionID)
			if err != nil {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid general position id").Err()
			}
			query.Where(dzopositiontitle.GeneralPositionIDEQ(generalPositionID))
		}
	}

	rows, err := query.
		Order(ent.Asc(dzopositiontitle.FieldLocalTitle)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query dzo position titles").Err()
	}

	titles := make([]DzoPositionTitle, 0, len(rows))
	for _, row := range rows {
		titles = append(titles, *entToDzoPositionTitle(row))
	}

	return titles, nil
}

func queryDzoPositionTitle(ctx context.Context, id, clientID string) (*DzoPositionTitle, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo position title id").Err()
	}
	query := Client.DzoPositionTitle.
		Query().
		Where(
			dzopositiontitle.ID(uid),
			dzopositiontitle.IsDeletedEQ(false),
		).
		WithDzo().
		WithGeneralPosition()
	if clientID != "" {
		clientUID, err := uuid.Parse(clientID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client id").Err()
		}
		query = query.Where(dzopositiontitle.ClientIDEQ(clientUID))
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, errs.B().Code(errs.NotFound).Msg("dzo position title not found").Err()
	}
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query dzo position title").Err()
	}

	return entToDzoPositionTitle(row), nil
}

func patchDzoPositionTitle(ctx context.Context, id string, req *UpdateDzoPositionTitleRequest, clientID string) (*DzoPositionTitle, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo position title id").Err()
	}

	if req.LocalTitle == nil &&
		req.GeneralPositionID == nil &&
		req.IsActive == nil {
		return queryDzoPositionTitle(ctx, id, clientID)
	}

	query := Client.DzoPositionTitle.
		Query().
		Where(
			dzopositiontitle.ID(uid),
			dzopositiontitle.IsDeletedEQ(false),
		)
	if clientID != "" {
		clientUID, err := uuid.Parse(clientID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client id").Err()
		}
		query = query.Where(dzopositiontitle.ClientIDEQ(clientUID))
	}
	current, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, errs.B().Code(errs.NotFound).Msg("dzo position title not found").Err()
	}
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query dzo position title").Err()
	}

	builder := Client.DzoPositionTitle.
		Update().
		Where(
			dzopositiontitle.ID(uid),
			dzopositiontitle.IsDeletedEQ(false),
		)
	if clientID != "" {
		clientUID, err := uuid.Parse(clientID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client id").Err()
		}
		builder = builder.Where(dzopositiontitle.ClientIDEQ(clientUID))
	}
	if req.LocalTitle != nil {
		exists, err := Client.DzoPositionTitle.Query().
			Where(
				dzopositiontitle.DzoIDEQ(current.DzoID),
				dzopositiontitle.LocalTitleEQ(*req.LocalTitle),
				dzopositiontitle.IsDeletedEQ(false),
				dzopositiontitle.IDNEQ(uid),
			).
			Exist(ctx)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to check existing dzo position title").Err()
		}
		if exists {
			return nil, errs.B().Code(errs.AlreadyExists).Msg("dzo position title already exists").Err()
		}

		builder.SetLocalTitle(*req.LocalTitle)
	}

	if req.GeneralPositionID != nil {
		generalPositionID, err := uuid.Parse(*req.GeneralPositionID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid general position id").Err()
		}
		builder.SetGeneralPositionID(generalPositionID)
	}

	if req.IsActive != nil {
		builder.SetIsActive(*req.IsActive)
	}

	count, err := builder.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update dzo position title").Err()
	}
	if count == 0 {
		return nil, errs.B().Code(errs.NotFound).Msg("dzo position title not found").Err()
	}

	return queryDzoPositionTitle(ctx, id, clientID)
}

func deleteDzoPositionTitle(ctx context.Context, id, clientID string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid dzo position title id").Err()
	}

	builder := Client.DzoPositionTitle.
		Update().
		Where(
			dzopositiontitle.ID(uid),
			dzopositiontitle.IsDeletedEQ(false),
		)

	if clientID != "" {
		clientUID, err := uuid.Parse(clientID)
		if err != nil {
			return errs.B().Code(errs.InvalidArgument).Msg("invalid client id").Err()
		}
		builder = builder.Where(dzopositiontitle.ClientIDEQ(clientUID))
	}
	count, err := builder.
		SetIsDeleted(true).
		SetIsActive(false).
		Save(ctx)

	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete dzo position title").Err()
	}
	if count == 0 {
		return errs.B().Code(errs.NotFound).Msg("dzo position title not found").Err()
	}

	return nil
}
func checkDzoExists(ctx context.Context, dzoID uuid.UUID) error {
	exists, err := Client.DzoOrganization.
		Query().
		Where(
			dzoorganization.IDEQ(dzoID),
			dzoorganization.IsActiveEQ(true),
		).
		Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to validate dzo_id").Cause(err).Err()
	}
	if !exists {
		return errs.B().Code(errs.InvalidArgument).Msg("dzo not found").Err()
	}
	return nil
}

// checkDzoExistsForClient ensures the DZO exists and belongs to the given client.
// Used to prevent ADM from creating employees in DZOs outside their client.
func checkDzoExistsForClient(ctx context.Context, dzoID uuid.UUID, clientID string) error {
	clientUID, err := uuid.Parse(clientID)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid company_id in token").Err()
	}
	exists, err := Client.DzoOrganization.
		Query().
		Where(
			dzoorganization.IDEQ(dzoID),
			dzoorganization.IsActiveEQ(true),
			dzoorganization.ClientIDEQ(clientUID),
		).
		Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to validate dzo_id").Cause(err).Err()
	}
	if !exists {
		return errs.B().Code(errs.InvalidArgument).Msg("dzo not found or not in your client").Err()
	}
	return nil
}

// HELPERS
func entToDzoPositionTitle(row *ent.DzoPositionTitle) *DzoPositionTitle {
	var generalPositionID *string
	if row.GeneralPositionID != nil {
		id := row.GeneralPositionID.String()
		generalPositionID = &id
	}

	dpt := &DzoPositionTitle{
		ID:                row.ID.String(),
		DzoID:             row.DzoID.String(),
		ClientID:          row.ClientID.String(),
		GeneralPositionID: generalPositionID,
		LocalTitle:        row.LocalTitle,
		IsActive:          row.IsActive,
		IsDeleted:         row.IsDeleted,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
	if row.Edges.Dzo != nil {
		dpt.DzoName = &row.Edges.Dzo.Name
	}
	if row.Edges.GeneralPosition != nil {
		dpt.GeneralPositionName = &row.Edges.GeneralPosition.Name
	}
	return dpt
}
