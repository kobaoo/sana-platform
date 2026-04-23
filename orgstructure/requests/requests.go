package requests

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	encoreuuid "encore.dev/types/uuid"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
	dbent "encore.app/db/ent"
	"encore.app/db/ent/employee"
	"encore.app/db/ent/request"
	"encore.app/db/ent/requestdzocontract"
	"encore.app/db/ent/requestemployee"
	"encore.app/db/ent/user"
)

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *dbent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return dbent.NewClient(dbent.Driver(drv))
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
		SetKind(string(RequestKindRegular)).
		SetStep(0).
		SetStatus("PENDING").
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create request").Cause(err).Err()
	}

	return toResponse(ctx, r)
}

// CreateArchiveRequest creates a closed/archive request that is immediately completed.
//
//encore:api auth method=POST path=/requests/archive
func CreateArchiveRequest(ctx context.Context, req *CreateArchiveRequestRequest) (*RequestResponse, error) {
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}

	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !isAdminRole(string(ad.Role)) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only admins can create archive requests").Err()
	}

	kind := normalizeRequestKind(req.Kind)
	if kind != string(RequestKindClosed) && kind != string(RequestKindArchived) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("kind must be CLOSED or ARCHIVED").Err()
	}
	if strings.TrimSpace(req.Category) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("category is required").Err()
	}
	if len(req.EmployeeIDs) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("employee_ids is required").Err()
	}
	if len(req.Contracts) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("contracts is required").Err()
	}

	actorID, err := getCurrentActorID(ctx)
	if err != nil {
		return nil, err
	}

	employeesByID, dzoIDs, err := loadEmployeesForArchiveRequest(ctx, req.EmployeeIDs)
	if err != nil {
		return nil, err
	}
	contractsByDZO, err := normalizeArchiveContracts(req.Contracts)
	if err != nil {
		return nil, err
	}
	if err := ensureContractsCoverDZOs(dzoIDs, contractsByDZO); err != nil {
		return nil, err
	}

	tx, err := Client.Tx(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}

	now := time.Now()
	requestBuilder := tx.Request.
		Create().
		SetInitiatorID(actorID).
		SetEntityID(uuid.New()).
		SetEntityType("ARCHIVE_REQUEST").
		SetKind(kind).
		SetCategory(strings.TrimSpace(req.Category)).
		SetStep(0).
		SetStatus("COMPLETED").
		SetCompletedAt(now)

	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		requestBuilder = requestBuilder.SetTitle(strings.TrimSpace(*req.Title))
	}

	created, err := requestBuilder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, errs.B().Code(errs.Internal).Msg("failed to create archive request").Cause(err).Err()
	}

	for _, employeeID := range uniqueUUIDs(req.EmployeeIDs) {
		if _, ok := employeesByID[employeeID]; !ok {
			_ = tx.Rollback()
			return nil, errs.B().Code(errs.InvalidArgument).Msg("unknown employee in request").Err()
		}

		if _, err := tx.RequestEmployee.
			Create().
			SetRequestID(created.ID).
			SetEmployeeID(employeeID).
			Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, errs.B().Code(errs.Internal).Msg("failed to save request employees").Cause(err).Err()
		}
	}

	for _, dzoID := range dzoIDs {
		contract := contractsByDZO[dzoID]
		if _, err := tx.RequestDzoContract.
			Create().
			SetRequestID(created.ID).
			SetDzoID(dzoID).
			SetFileName(contract.FileName).
			SetFileURL(contract.FileURL).
			Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, errs.B().Code(errs.Internal).Msg("failed to save request contracts").Cause(err).Err()
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit archive request").Cause(err).Err()
	}

	return toResponse(ctx, created)
}

//encore:api auth method=GET path=/requests
func ListRequests(ctx context.Context) (*ListRequestsResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	rows, err := Client.Request.
		Query().
		Order(dbent.Desc(request.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list requests").Cause(err).Err()
	}

	items := make([]*RequestResponse, 0, len(rows))
	for _, r := range rows {
		if isArchiveOnlyRequest(r.Kind) && !isAdminRole(string(ad.Role)) {
			continue
		}

		resp, err := toResponse(ctx, r)
		if err != nil {
			return nil, err
		}
		items = append(items, resp)
	}

	return &ListRequestsResponse{Items: items}, nil
}

//encore:api auth method=GET path=/requests/:id
func GetRequest(ctx context.Context, id encoreuuid.UUID) (*RequestResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	requestID := uuid.UUID(id)

	r, err := Client.Request.
		Query().
		Where(request.IDEQ(requestID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get request").Cause(err).Err()
	}

	if isArchiveOnlyRequest(r.Kind) && !isAdminRole(string(ad.Role)) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("archive requests are available only to admin").Err()
	}

	return toResponse(ctx, r)
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
		if dbent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get request").Cause(err).Err()
	}

	if existing.Status == "APPROVED" || existing.Status == "CANCELLED" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already finalized").Err()
	}
	if isArchiveOnlyRequest(existing.Kind) || existing.Status == "COMPLETED" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("archive request cannot change step").Err()
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

	return toResponse(ctx, updated)
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
		if dbent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get request").Cause(err).Err()
	}

	if existing.Status == "APPROVED" || existing.Status == "CANCELLED" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already finalized").Err()
	}
	if isArchiveOnlyRequest(existing.Kind) || existing.Status == "COMPLETED" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("archive request status is managed automatically").Err()
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

	return toResponse(ctx, updated)
}

func toResponse(ctx context.Context, r *dbent.Request) (*RequestResponse, error) {
	employeeRows, err := Client.RequestEmployee.
		Query().
		Where(requestemployee.RequestIDEQ(r.ID)).
		Order(dbent.Asc(requestemployee.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request employees").Cause(err).Err()
	}

	contractRows, err := Client.RequestDzoContract.
		Query().
		Where(requestdzocontract.RequestIDEQ(r.ID)).
		Order(dbent.Asc(requestdzocontract.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request contracts").Cause(err).Err()
	}

	resp := &RequestResponse{
		ID:          r.ID,
		InitiatorID: r.InitiatorID,
		EntityID:    r.EntityID,
		EntityType:  r.EntityType,
		Kind:        r.Kind,
		Title:       r.Title,
		Category:    r.Category,
		Step:        r.Step,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		CompletedAt: r.CompletedAt,
		EmployeeIDs: make([]uuid.UUID, 0, len(employeeRows)),
		Contracts:   make([]RequestContractResponse, 0, len(contractRows)),
	}

	for _, row := range employeeRows {
		resp.EmployeeIDs = append(resp.EmployeeIDs, row.EmployeeID)
	}
	for _, row := range contractRows {
		resp.Contracts = append(resp.Contracts, RequestContractResponse{
			DzoID:    row.DzoID,
			FileName: row.FileName,
			FileURL:  row.FileURL,
		})
	}

	return resp, nil
}

func isValidStatus(status string) bool {
	switch status {
	case "PENDING", "APPROVED", "CANCELLED":
		return true
	default:
		return false
	}
}

func normalizeRequestKind(kind string) string {
	return strings.ToUpper(strings.TrimSpace(kind))
}

func isArchiveOnlyRequest(kind string) bool {
	switch normalizeRequestKind(kind) {
	case string(RequestKindClosed), string(RequestKindArchived):
		return true
	default:
		return false
	}
}

func isAdminRole(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case string(authhandler.RoleSA), string(authhandler.RoleADM):
		return true
	default:
		return false
	}
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func loadEmployeesForArchiveRequest(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*dbent.Employee, []uuid.UUID, error) {
	uniqueIDs := uniqueUUIDs(ids)
	rows, err := Client.Employee.
		Query().
		Where(employee.IDIn(uniqueIDs...)).
		All(ctx)
	if err != nil {
		return nil, nil, errs.B().Code(errs.Internal).Msg("failed to load employees").Cause(err).Err()
	}
	if len(rows) != len(uniqueIDs) {
		return nil, nil, errs.B().Code(errs.InvalidArgument).Msg("some employees were not found").Err()
	}

	employeesByID := make(map[uuid.UUID]*dbent.Employee, len(rows))
	dzoSeen := make(map[uuid.UUID]struct{})
	dzoIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		employeesByID[row.ID] = row
		if _, ok := dzoSeen[row.DzoID]; ok {
			continue
		}
		dzoSeen[row.DzoID] = struct{}{}
		dzoIDs = append(dzoIDs, row.DzoID)
	}

	slices.SortFunc(dzoIDs, func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	})

	return employeesByID, dzoIDs, nil
}

func normalizeArchiveContracts(in []ArchiveRequestContractInput) (map[uuid.UUID]ArchiveRequestContractInput, error) {
	contractsByDZO := make(map[uuid.UUID]ArchiveRequestContractInput, len(in))
	for _, contract := range in {
		if contract.DzoID == uuid.Nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("contract dzo_id is required").Err()
		}
		if strings.TrimSpace(contract.FileName) == "" {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("contract file_name is required").Err()
		}
		if strings.TrimSpace(contract.FileURL) == "" {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("contract file_url is required").Err()
		}
		if _, exists := contractsByDZO[contract.DzoID]; exists {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("duplicate contract for one dzo").Err()
		}

		contract.FileName = strings.TrimSpace(contract.FileName)
		contract.FileURL = strings.TrimSpace(contract.FileURL)
		contractsByDZO[contract.DzoID] = contract
	}
	return contractsByDZO, nil
}

func ensureContractsCoverDZOs(required []uuid.UUID, contractsByDZO map[uuid.UUID]ArchiveRequestContractInput) error {
	for _, dzoID := range required {
		if _, ok := contractsByDZO[dzoID]; !ok {
			return errs.B().Code(errs.InvalidArgument).Msg(fmt.Sprintf("contract is required for dzo %s", dzoID)).Err()
		}
	}

	for dzoID := range contractsByDZO {
		if slices.Contains(required, dzoID) {
			continue
		}
		return errs.B().Code(errs.InvalidArgument).Msg(fmt.Sprintf("contract dzo %s is not related to selected employees", dzoID)).Err()
	}
	return nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func getUserRole(ctx context.Context, actorID uuid.UUID) (string, error) {
	u, err := Client.User.
		Query().
		Where(user.IDEQ(actorID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return "", errs.B().Code(errs.NotFound).Msg("actor not found").Err()
		}
		return "", errs.B().Code(errs.Internal).Msg("failed to get actor").Cause(err).Err()
	}

	return strings.ToUpper(strings.TrimSpace(u.Role)), nil
}

func getCurrentActorID(ctx context.Context) (uuid.UUID, error) {
	ad, err := getAuthData()
	if err != nil {
		return uuid.Nil, err
	}

	u, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(ad.KeycloakUserID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
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
