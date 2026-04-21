package requests

import (
	"context"
	"strings"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/request"
	"encore.app/db/ent/user"
)

// ════ DATABASE ════
// Client is declared in requests.go and shared across this package.

const entityTypeCertRenewal = "CERTIFICATE_RENEWAL"

// ════ ENDPOINTS ════

//encore:api auth method=POST path=/certificate-requests
func CreateCertificateRenewal(ctx context.Context, req *CreateCertificateRenewalRequest) (*RequestResponse, error) {
	if req.EntityID == uuid.Nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("entity_id is required").Err()
	}

	ad, err := getCertAuthData()
	if err != nil {
		return nil, err
	}
	if !canCreateCertRenewal(string(ad.Role)) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can create certificate renewal requests").Err()
	}

	initiatorID, err := resolveInitiatorID(ctx, ad)
	if err != nil {
		return nil, err
	}

	return insertCertRenewal(ctx, initiatorID, req.EntityID)
}

//encore:api auth method=GET path=/certificate-requests
func ListCertificateRenewals(ctx context.Context, p *ListCertificateRenewalsParams) (*ListRequestsResponse, error) {
	ad, err := getCertAuthData()
	if err != nil {
		return nil, err
	}
	if !canViewCertRenewal(string(ad.Role)) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only SA, ADM, or HR can view certificate renewal requests").Err()
	}

	query := Client.Request.
		Query().
		Where(request.EntityTypeEQ(entityTypeCertRenewal))

	if p != nil && p.InitiatorID != "" {
		id, err := uuid.Parse(p.InitiatorID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid initiator_id format").Err()
		}
		query = query.Where(request.InitiatorIDEQ(id))
	}

	rows, err := query.
		Order(ent.Desc(request.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list certificate renewal requests").Cause(err).Err()
	}

	items := make([]*RequestResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, toResponse(r))
	}

	return &ListRequestsResponse{Items: items}, nil
}

//encore:api auth method=GET path=/certificate-requests/:id
func GetCertificateRenewal(ctx context.Context, id string) (*RequestResponse, error) {
	ad, err := getCertAuthData()
	if err != nil {
		return nil, err
	}
	if !canViewCertRenewal(string(ad.Role)) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only SA, ADM, or HR can view certificate renewal requests").Err()
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	r, err := queryCertRenewalByID(ctx, parsed)
	if err != nil {
		return nil, err
	}
	return toResponse(r), nil
}

//encore:api auth method=PATCH path=/certificate-requests/:id/status
func PatchCertificateRenewalStatus(ctx context.Context, id string, req *PatchCertificateRenewalStatusRequest) (*RequestResponse, error) {
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}

	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if !isValidCertRenewalStatus(status) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("status must be APPROVED or REJECTED").Err()
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	ad, err := getCertAuthData()
	if err != nil {
		return nil, err
	}
	if !canReviewCertRenewal(string(ad.Role)) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only ADM or SA can approve or reject certificate renewal requests").Err()
	}

	return applyCertRenewalStatus(ctx, parsed, status)
}

// ════ INTERNAL ════

func insertCertRenewal(ctx context.Context, initiatorID, entityID uuid.UUID) (*RequestResponse, error) {
	r, err := Client.Request.
		Create().
		SetInitiatorID(initiatorID).
		SetEntityID(entityID).
		SetEntityType(entityTypeCertRenewal).
		SetStep(0).
		SetStatus("PENDING").
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create certificate renewal request").Cause(err).Err()
	}
	return toResponse(r), nil
}

func applyCertRenewalStatus(ctx context.Context, id uuid.UUID, status string) (*RequestResponse, error) {
	existing, err := queryCertRenewalByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.Status != "PENDING" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("заявка уже обработана").Err()
	}

	updated, err := existing.
		Update().
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update certificate renewal status").Cause(err).Err()
	}

	return toResponse(updated), nil
}

func getCertAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func resolveInitiatorID(ctx context.Context, ad *authhandler.AuthData) (uuid.UUID, error) {
	u, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(ad.KeycloakUserID)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to resolve initiator").Cause(err).Err()
		}
		// User not in DB yet — auto-create from token claims on first request.
		builder := Client.User.Create().
			SetKeycloakUserID(ad.KeycloakUserID).
			SetEmail(ad.Email).
			SetRole(string(ad.Role))
		if ad.DzoID != "" {
			if dzoUID, parseErr := uuid.Parse(ad.DzoID); parseErr == nil {
				builder = builder.SetDzoID(dzoUID)
			}
		}
		u, err = builder.Save(ctx)
		if err != nil {
			return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to create user record").Cause(err).Err()
		}
	}
	return u.ID, nil
}

func queryCertRenewalByID(ctx context.Context, id uuid.UUID) (*ent.Request, error) {
	r, err := Client.Request.
		Query().
		Where(
			request.IDEQ(id),
			request.EntityTypeEQ(entityTypeCertRenewal),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("certificate renewal request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get certificate renewal request").Cause(err).Err()
	}
	return r, nil
}

func isValidCertRenewalStatus(status string) bool {
	return status == "APPROVED" || status == "REJECTED"
}

func canCreateCertRenewal(role string) bool {
	return role == "HR"
}

func canViewCertRenewal(role string) bool {
	return role == "SA" || role == "ADM" || role == "HR"
}

func canReviewCertRenewal(role string) bool {
	return role == "ADM" || role == "SA"
}
