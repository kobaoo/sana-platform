package requests

import (
	"context"
	"strings"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	encoreuuid "encore.dev/types/uuid"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/request"
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

//encore:api auth method=POST path=/requests
func CreateRequest(ctx context.Context, req *CreateRequestRequest) (*RequestResponse, error) {
	if req.EntityID == uuid.Nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("entity_id is required").Err()
	}
	if strings.TrimSpace(req.EntityType) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("entity_type is required").Err()
	}
	actorID, err := getCurrentActorID(ctx)
	if err != nil {
		return nil, err
	}

	r, err := Client.Request.
		Create().
		SetInitiatorID(actorID).
		SetEntityID(req.EntityID).
		SetEntityType(strings.TrimSpace(req.EntityType)).
		SetStep(0).
		SetStatus("PENDING").
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create request").Cause(err).Err()
	}

	return toResponse(r), nil
}

//encore:api auth method=GET path=/requests
func ListRequests(ctx context.Context) (*ListRequestsResponse, error) {
	rows, err := Client.Request.
		Query().
		Order(ent.Desc(request.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list requests").Cause(err).Err()
	}

	items := make([]*RequestResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, toResponse(r))
	}

	return &ListRequestsResponse{Items: items}, nil
}

//encore:api auth method=GET path=/requests/:id
func GetRequest(ctx context.Context, id encoreuuid.UUID) (*RequestResponse, error) {
	requestID := uuid.UUID(id)

	r, err := Client.Request.
		Query().
		Where(request.IDEQ(requestID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get request").Cause(err).Err()
	}

	return toResponse(r), nil
}

//encore:api auth method=PUT path=/requests/:id/step
func UpdateRequestStep(ctx context.Context, id encoreuuid.UUID, req *UpdateRequestStepRequest) (*RequestResponse, error) {
	requestID := uuid.UUID(id)

	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if req.Step < 0 || req.Step > 3 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("step must be between 0 and 3").Err()
	}
	actorID, err := getCurrentActorID(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := Client.Request.
		Query().
		Where(request.IDEQ(requestID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get request").Cause(err).Err()
	}

	if existing.Status == "APPROVED" || existing.Status == "CANCELLED" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already finalized").Err()
	}

	if req.Step != existing.Step+1 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid step transition").Err()
	}

	role, err := getUserRole(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !canMoveStep(role, existing.Step, req.Step) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("actor is not allowed to move request to this step").Err()
	}

	updated, err := existing.
		Update().
		SetStep(req.Step).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update request step").Cause(err).Err()
	}

	return toResponse(updated), nil
}

//encore:api auth method=PUT path=/requests/:id/status
func UpdateRequestStatus(ctx context.Context, id encoreuuid.UUID, req *UpdateRequestStatusRequest) (*RequestResponse, error) {
	requestID := uuid.UUID(id)

	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	actorID, err := getCurrentActorID(ctx)
	if err != nil {
		return nil, err
	}

	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if !isValidStatus(status) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid status").Err()
	}

	existing, err := Client.Request.
		Query().
		Where(request.IDEQ(requestID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get request").Cause(err).Err()
	}

	if existing.Status == "APPROVED" || existing.Status == "CANCELLED" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already finalized").Err()
	}

	if status != "PENDING" {
		role, err := getUserRole(ctx, actorID)
		if err != nil {
			return nil, err
		}

		if !canFinalize(role, existing.Step) {
			return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can finalize request at step 3").Err()
		}
	}

	updated, err := existing.
		Update().
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update request status").Cause(err).Err()
	}

	return toResponse(updated), nil
}

func toResponse(r *ent.Request) *RequestResponse {
	return &RequestResponse{
		ID:          r.ID,
		InitiatorID: r.InitiatorID,
		EntityID:    r.EntityID,
		EntityType:  r.EntityType,
		Step:        r.Step,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
}

func isValidStatus(status string) bool {
	switch status {
	case "PENDING", "APPROVED", "CANCELLED":
		return true
	default:
		return false
	}
}

func getUserRole(ctx context.Context, actorID uuid.UUID) (string, error) {
	u, err := Client.User.
		Query().
		Where(user.IDEQ(actorID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", errs.B().Code(errs.NotFound).Msg("actor not found").Err()
		}
		return "", errs.B().Code(errs.Internal).Msg("failed to get actor").Cause(err).Err()
	}

	return strings.ToUpper(strings.TrimSpace(u.Role)), nil
}

func getCurrentActorID(ctx context.Context) (uuid.UUID, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return uuid.Nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}

	u, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(ad.KeycloakUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.Nil, errs.B().Code(errs.NotFound).Msg("actor not found").Err()
		}
		return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to resolve actor").Cause(err).Err()
	}
	return u.ID, nil
}

func canMoveStep(role string, currentStep, nextStep int) bool {
	switch role {
	case "HR":
		return (currentStep == 0 && nextStep == 1) || (currentStep == 2 && nextStep == 3)
	case "ADM":
		return currentStep == 1 && nextStep == 2
	default:
		return false
	}
}

func canFinalize(role string, step int) bool {
	return role == "HR" && step == 3
}
