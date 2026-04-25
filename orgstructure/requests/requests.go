package requests

import (
	"context"
	"database/sql"
	"errors"
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
	"encore.app/db/ent/user"
)

// ════ DATABASE ════

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *dbent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return dbent.NewClient(dbent.Driver(drv))
}

// ════ ENDPOINTS ════

// CreateAdminRequest creates a main request for external learning.
//
//encore:api auth method=POST path=/requests/admin
func CreateAdminRequest(ctx context.Context, req *CreateAdminRequestRequest) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	if strings.TrimSpace(req.TrainingEventID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("training_event_id is required").Err()
	}
	if len(req.EmployeeIDs) == 0 && len(req.DzoIDs) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("employee_ids or dzo_ids must be provided").Err()
	}
	if req.CostMode != nil && !req.CostMode.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid cost_mode").Err()
	}
	if req.CostAmount != nil && *req.CostAmount < 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("cost_amount cannot be negative").Err()
	}

	detail, err := createAdminRequest(ctx, actor, req)
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// ListRequests returns main requests visible to admins.
//
//encore:api auth method=GET path=/requests
func ListRequests(ctx context.Context) (*ListRequestsResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	items, err := queryRequestSummaries(ctx, `
		SELECT
			r.id, r.initiator_id, r.parent_request_id, r.entity_id, r.entity_type, r.request_type, r.kind,
			r.assigned_hr_id, r.target_dzo_id, r.title, r.category, r.format, r.responsible_admin_id,
			r.training_date, r.deadline_at, r.cost_amount, r.cost_mode, r.status, r.created_at, r.updated_at, r.completed_at,
			COALESCE((SELECT COUNT(*) FROM request_participants rp WHERE rp.request_id = r.id), 0) AS employees_count,
			COALESCE((SELECT COUNT(*) FROM requests c WHERE c.parent_request_id = r.id AND c.status = 'APPROVED'), 0) AS approved_children,
			COALESCE((SELECT COUNT(*) FROM requests c WHERE c.parent_request_id = r.id), 0) AS total_children
		FROM requests r
		WHERE r.request_type = 'MAIN'
		ORDER BY r.created_at DESC
	`)
	if err != nil {
		return nil, err
	}

	return &ListRequestsResponse{Items: items}, nil
}

// GetRequest returns request details for admins.
//
//encore:api auth method=GET path=/requests/admin/:id
func GetRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	detail, err := buildRequestDetail(ctx, uuid.UUID(id))
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// SubmitRequest splits a main request into HR subrequests.
//
//encore:api auth method=POST path=/requests/admin/:id/submit
func SubmitRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	detail, err := submitRequest(ctx, actor, uuid.UUID(id))
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// CancelAdminRequest cancels a main request and refunds budget if it was written off.
//
//encore:api auth method=POST path=/requests/admin/:id/cancel
func CancelAdminRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	detail, err := cancelAdminRequest(ctx, actor, uuid.UUID(id))
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// ListHRRequests returns subrequests assigned to the current HR.
//
//encore:api auth method=GET path=/requests/hr
func ListHRRequests(ctx context.Context) (*ListRequestsResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if actor.Role != authhandler.RoleHR {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can access this list").Err()
	}

	items, err := queryRequestSummaries(ctx, `
		SELECT
			r.id, r.initiator_id, r.parent_request_id, r.entity_id, r.entity_type, r.request_type, r.kind,
			r.assigned_hr_id, r.target_dzo_id, r.title, r.category, r.format, r.responsible_admin_id,
			r.training_date, r.deadline_at, r.cost_amount, r.cost_mode, r.status, r.created_at, r.updated_at, r.completed_at,
			COALESCE((SELECT COUNT(*) FROM request_participants rp WHERE rp.request_id = r.id), 0) AS employees_count,
			0 AS approved_children,
			0 AS total_children
		FROM requests r
		WHERE r.request_type = 'SUBREQUEST' AND r.assigned_hr_id = $1
		ORDER BY r.created_at DESC
	`, actor.ID)
	if err != nil {
		return nil, err
	}

	return &ListRequestsResponse{Items: items}, nil
}

// GetHRRequest returns HR request details.
//
//encore:api auth method=GET path=/requests/hr/:id
func GetHRRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if actor.Role != authhandler.RoleHR {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can access this request").Err()
	}

	requestSummary, err := queryRequestSummaryByID(ctx, uuid.UUID(id))
	if err != nil {
		return nil, err
	}
	if requestSummary.RequestType != RequestTypeSubrequest || requestSummary.AssignedHRID == nil || *requestSummary.AssignedHRID != actor.ID.String() {
		return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
	}

	detail, err := buildRequestDetail(ctx, uuid.UUID(id))
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// UpdateHRRequestEmployees replaces the employee set in an HR subrequest.
//
//encore:api auth method=PUT path=/requests/hr/:id/employees
func UpdateHRRequestEmployees(ctx context.Context, id encoreuuid.UUID, req *UpdateHRRequestEmployeesRequest) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if actor.Role != authhandler.RoleHR {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can update employees").Err()
	}

	detail, err := updateHRRequestEmployees(ctx, actor, uuid.UUID(id), req)
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// ApproveHRRequest approves an HR subrequest.
//
//encore:api auth method=POST path=/requests/hr/:id/approve
func ApproveHRRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if actor.Role != authhandler.RoleHR {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can approve requests").Err()
	}

	detail, err := finalizeHRRequest(ctx, actor, uuid.UUID(id), RequestStatusApproved)
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// CancelHRRequest rejects an HR subrequest.
//
//encore:api auth method=POST path=/requests/hr/:id/cancel
func CancelHRRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if actor.Role != authhandler.RoleHR {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can reject requests").Err()
	}

	detail, err := finalizeHRRequest(ctx, actor, uuid.UUID(id), RequestStatusRejected)
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// CreateHRRequest creates a main request from HR to Admin.
//
//encore:api auth method=POST path=/requests/hr-request
func CreateHRRequest(ctx context.Context, req *CreateHRRequestRequest) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if actor.Role != authhandler.RoleHR {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only HR can create request").Err()
	}

	detail, err := createHRRequest(ctx, actor, req)
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// FinalizeAdminRequest finalizes a main request and writes off budget.
//
//encore:api auth method=POST path=/requests/admin/:id/finalize
func FinalizeAdminRequest(ctx context.Context, id encoreuuid.UUID) (*GetRequestResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	detail, err := finalizeAdminRequest(ctx, actor, uuid.UUID(id))
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
}

// budget
//
//encore:api auth method=GET path=/requests/admin/:id/budget-history
func GetRequestBudgetHistory(ctx context.Context, id encoreuuid.UUID) (*GetRequestBudgetHistoryResponse, error) {
	actor, err := resolveCurrentActor(ctx)
	if err != nil {
		return nil, err
	}
	if !canManageAdminRequests(actor.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	rows, err := db.Query(ctx, `
		SELECT operation_type, amount, created_by, reason, created_at
		FROM request_budget_transactions
		WHERE request_id = $1
		ORDER BY created_at ASC
	`, uuid.UUID(id))
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load budget history").Cause(err).Err()
	}
	defer rows.Close()

	items := []BudgetHistoryItem{}

	for rows.Next() {
		var (
			op        string
			amount    float64
			createdBy uuid.UUID
			reason    sql.NullString
			createdAt time.Time
		)

		if err := rows.Scan(&op, &amount, &createdBy, &reason, &createdAt); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan budget history").Cause(err).Err()
		}

		items = append(items, BudgetHistoryItem{
			OperationType: op,
			Amount:        amount,
			CreatedBy:     createdBy.String(),
			Reason:        nullableStringValue(reason),
			CreatedAt:     createdAt,
		})
	}

	return &GetRequestBudgetHistoryResponse{Items: items}, nil
}

// ════ INTERNAL ════

type rowScanner interface {
	Scan(dest ...interface{}) error
}

type actor struct {
	ID    uuid.UUID
	Role  authhandler.UserRole
	DzoID *uuid.UUID
}

type trainingEventRecord struct {
	ID           uuid.UUID
	Title        string
	StartDate    time.Time
	LocationType sql.NullString
}

type employeeRecord struct {
	ID       uuid.UUID
	FullName string
	DzoID    uuid.UUID
	DzoName  string
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func resolveCurrentActor(ctx context.Context) (*actor, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(ctx, `
		SELECT id, role, dzo_id
		FROM users
		WHERE keycloak_user_id = $1 AND is_active = TRUE
	`, ad.KeycloakUserID)

	var (
		id    uuid.UUID
		role  string
		dzoID sql.NullString
	)
	if err := row.Scan(&id, &role, &dzoID); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.NotFound).Msg("actor not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to resolve actor").Cause(err).Err()
	}

	current := &actor{
		ID:   id,
		Role: authhandler.UserRole(strings.ToUpper(strings.TrimSpace(role))),
	}
	if dzoID.Valid {
		value, err := uuid.Parse(dzoID.String)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to parse actor dzo").Cause(err).Err()
		}
		current.DzoID = &value
	}
	return current, nil
}

func canManageAdminRequests(role authhandler.UserRole) bool {
	return role == authhandler.RoleSA || role == authhandler.RoleADM
}

func createAdminRequest(ctx context.Context, actor *actor, req *CreateAdminRequestRequest) (*RequestDetail, error) {
	trainingEvent, err := loadTrainingEvent(ctx, req.TrainingEventID)
	if err != nil {
		return nil, err
	}

	participants, selectedEmployeeIDs, err := resolveSelectedEmployees(ctx, req.EmployeeIDs)
	if err != nil {
		return nil, err
	}
	selectedDZOIDs, err := resolveSelectedDZOs(ctx, req.DzoIDs)
	if err != nil {
		return nil, err
	}

	affectedDZOIDs := collectAffectedDZOs(participants, selectedDZOIDs)
	if len(affectedDZOIDs) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request must target at least one employee or DZO").Err()
	}
	if _, err := resolveHRByDZOs(ctx, affectedDZOIDs); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(trainingEvent.Title)
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		title = strings.TrimSpace(*req.Title)
	}
	var format interface{}
	if req.Format != nil && strings.TrimSpace(*req.Format) != "" {
		format = strings.TrimSpace(*req.Format)
	} else if trainingEvent.LocationType.Valid {
		format = trainingEvent.LocationType.String
	}

	deadlineAt, err := parseOptionalTime(req.DeadlineAt)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	requestID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO requests (
			id, initiator_id, entity_id, entity_type, request_type, kind, title, category, format,
			responsible_admin_id, training_date, deadline_at, cost_amount, cost_mode,
			step, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'TRAINING_EVENT', 'MAIN', 'REGULAR', $4, $5, $6, $7, $8, $9, $10, $11, 0, 'DRAFT', NOW(), NOW())
	`,
		requestID,
		actor.ID,
		trainingEvent.ID,
		title,
		nullableString(req.Category),
		format,
		actor.ID,
		trainingEvent.StartDate,
		deadlineAt,
		nullableFloat(req.CostAmount),
		nullableCostMode(req.CostMode),
	); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create request").Cause(err).Err()
	}

	if err := replaceRequestParticipantsTx(ctx, tx, requestID, selectedEmployeeIDs); err != nil {
		return nil, err
	}
	if err := replaceRequestTargetDZOsTx(ctx, tx, requestID, selectedDZOIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit request").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}

// CreateHRRequest creates a main request from HR to Admin.
func createHRRequest(ctx context.Context, actor *actor, req *CreateHRRequestRequest) (*RequestDetail, error) {
	if actor.DzoID == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("HR must be assigned to DZO").Err()
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}

	selectedEmployeeIDs, err := resolveHREmployeesForRequest(ctx, req.EmployeeIDs, *actor.DzoID, req.AllowInactiveEmployees)
	if err != nil {
		return nil, err
	}

	deadlineAt, err := parseOptionalTime(req.DeadlineAt)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	requestID := uuid.New()

	if _, err := tx.Exec(ctx, `
		INSERT INTO requests (
			id, initiator_id, entity_id, entity_type, request_type,
			target_dzo_id, title, deadline_at,
			step, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'HR_REQUEST', 'MAIN',
			$4, $5, $6,
			0, 'PENDING', NOW(), NOW())
	`,
		requestID,
		actor.ID,
		uuid.Nil,
		*actor.DzoID,
		title,
		deadlineAt,
	); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create HR request").Cause(err).Err()
	}

	if err := replaceRequestParticipantsTx(ctx, tx, requestID, selectedEmployeeIDs); err != nil {
		return nil, err
	}

	if err := replaceRequestTargetDZOsTx(ctx, tx, requestID, []uuid.UUID{*actor.DzoID}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit HR request").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}
func submitRequest(ctx context.Context, actor *actor, requestID uuid.UUID) (*RequestDetail, error) {
	mainRequest, err := queryRequestSummaryByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if mainRequest.RequestType != RequestTypeMain {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("only main requests can be submitted").Err()
	}
	if mainRequest.Status != RequestStatusDraft {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already submitted").Err()
	}
	if mainRequest.InitiatorID != actor.ID.String() && actor.Role != authhandler.RoleSA {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("only the initiator can submit this request").Err()
	}

	mainEmployees, err := queryRequestEmployees(ctx, requestID)
	if err != nil {
		return nil, err
	}
	targetDZOs, err := queryRequestTargetDZOIDs(ctx, requestID)
	if err != nil {
		return nil, err
	}

	groupedEmployeeIDs := make(map[uuid.UUID][]uuid.UUID)
	for _, targetDZO := range targetDZOs {
		groupedEmployeeIDs[targetDZO] = []uuid.UUID{}
	}
	for _, employee := range mainEmployees {
		employeeID, _ := uuid.Parse(employee.ID)
		dzoID, _ := uuid.Parse(employee.DzoID)
		groupedEmployeeIDs[dzoID] = append(groupedEmployeeIDs[dzoID], employeeID)
	}
	if len(groupedEmployeeIDs) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request has no employees or DZO targets to split").Err()
	}

	hrByDZO, err := resolveHRByDZOs(ctx, mapsKeys(groupedEmployeeIDs))
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	//budget reserve
	if err := reserveBudgetTx(ctx, tx, requestID, actor.ID); err != nil {
		return nil, err
	}
	for _, dzoID := range mapsKeys(groupedEmployeeIDs) {
		hrID, ok := hrByDZO[dzoID]
		if !ok {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("hr not found for one of selected DZO").Err()
		}

		childID := uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO requests (
				id, initiator_id, parent_request_id, entity_id, entity_type, request_type, kind, assigned_hr_id,
				target_dzo_id, title, category, format, responsible_admin_id, training_date,
				deadline_at, cost_amount, cost_mode, step, status, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, 'SUBREQUEST', 'REGULAR', $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, 0, 'PENDING', NOW(), NOW())
		`,
			childID,
			actor.ID,
			requestID,
			parseUUIDOrNil(mainRequest.TrainingEventID),
			mainRequest.EntityType,
			hrID,
			dzoID,
			mainRequest.Title,
			nullableString(mainRequest.Category),
			nullableString(mainRequest.Format),
			parseUUIDOrNilPtr(mainRequest.ResponsibleAdminID),
			mainRequest.TrainingDate,
			mainRequest.DeadlineAt,
			nullableFloat(mainRequest.CostAmount),
			nullableCostMode(mainRequest.CostMode),
		); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to create subrequest").Cause(err).Err()
		}

		if err := replaceRequestParticipantsTx(ctx, tx, childID, groupedEmployeeIDs[dzoID]); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET status = 'IN_PROGRESS', updated_at = NOW()
		WHERE id = $1
	`, requestID); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update parent request").Cause(err).Err()
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit request split").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}

func updateHRRequestEmployees(ctx context.Context, actor *actor, requestID uuid.UUID, req *UpdateHRRequestEmployeesRequest) (*RequestDetail, error) {
	requestSummary, err := queryRequestSummaryByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if requestSummary.RequestType != RequestTypeSubrequest || requestSummary.AssignedHRID == nil || *requestSummary.AssignedHRID != actor.ID.String() {
		return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
	}
	if requestSummary.Status != RequestStatusPending {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already finalized").Err()
	}
	if requestSummary.TargetDzoID == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("subrequest is missing target DZO").Err()
	}

	targetDZO, err := uuid.Parse(*requestSummary.TargetDzoID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to parse target DZO").Cause(err).Err()
	}
	employeeRecords, employeeIDs, err := resolveSelectedEmployees(ctx, req.EmployeeIDs)
	if err != nil {
		return nil, err
	}
	for _, employee := range employeeRecords {
		if employee.DzoID != targetDZO {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("all employees must belong to the HR DZO").Err()
		}
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	if err := replaceRequestParticipantsTx(ctx, tx, requestID, employeeIDs); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET updated_at = NOW()
		WHERE id = $1
	`, requestID); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to touch request").Cause(err).Err()
	}
	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit employee update").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}

func finalizeHRRequest(ctx context.Context, actor *actor, requestID uuid.UUID, status RequestStatus) (*RequestDetail, error) {
	requestSummary, err := queryRequestSummaryByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if requestSummary.RequestType != RequestTypeSubrequest || requestSummary.AssignedHRID == nil || *requestSummary.AssignedHRID != actor.ID.String() {
		return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
	}
	if requestSummary.Status != RequestStatusPending {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already finalized").Err()
	}

	if status == RequestStatusApproved {
		count, err := queryParticipantCount(ctx, requestID)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("cannot approve request without employees").Err()
		}
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, requestID, string(status)); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to finalize request").Cause(err).Err()
	}

	if requestSummary.ParentRequestID != nil {
		parentID, err := uuid.Parse(*requestSummary.ParentRequestID)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to parse parent request id").Cause(err).Err()
		}
		if err := syncParentRequestStatusTx(ctx, tx, parentID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit request finalization").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}

func buildRequestDetail(ctx context.Context, requestID uuid.UUID) (*RequestDetail, error) {
	summary, err := queryRequestSummaryByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	employees, err := queryRequestEmployees(ctx, requestID)
	if err != nil {
		return nil, err
	}
	targetDZOs, err := queryRequestTargetDZOs(ctx, requestID)
	if err != nil {
		return nil, err
	}

	children := []RequestSummary{}
	if summary.RequestType == RequestTypeMain {
		children, err = queryRequestSummaries(ctx, `
			SELECT
				r.id, r.initiator_id, r.parent_request_id, r.entity_id, r.entity_type, r.request_type, r.kind,
				r.assigned_hr_id, r.target_dzo_id, r.title, r.category, r.format, r.responsible_admin_id,
				r.training_date, r.deadline_at, r.cost_amount, r.cost_mode, r.status, r.created_at, r.updated_at, r.completed_at,
				COALESCE((SELECT COUNT(*) FROM request_participants rp WHERE rp.request_id = r.id), 0) AS employees_count,
				0 AS approved_children,
				0 AS total_children
			FROM requests r
			WHERE r.parent_request_id = $1
			ORDER BY r.created_at ASC
		`, requestID)
		if err != nil {
			return nil, err
		}
	}

	contracts, err := queryRequestDzoContracts(ctx, requestID)
	if err != nil {
		return nil, err
	}

	return &RequestDetail{
		Request:       *summary,
		Employees:     employees,
		TargetDZOs:    targetDZOs,
		Contracts:     contracts,
		ChildRequests: children,
	}, nil
}

func queryRequestSummaryByID(ctx context.Context, requestID uuid.UUID) (*RequestSummary, error) {
	row := db.QueryRow(ctx, `
		SELECT
			r.id, r.initiator_id, r.parent_request_id, r.entity_id, r.entity_type, r.request_type, r.kind,
			r.assigned_hr_id, r.target_dzo_id, r.title, r.category, r.format, r.responsible_admin_id,
			r.training_date, r.deadline_at, r.cost_amount, r.cost_mode, r.status, r.created_at, r.updated_at, r.completed_at,
			COALESCE((SELECT COUNT(*) FROM request_participants rp WHERE rp.request_id = r.id), 0) AS employees_count,
			COALESCE((SELECT COUNT(*) FROM requests c WHERE c.parent_request_id = r.id AND c.status = 'APPROVED'), 0) AS approved_children,
			COALESCE((SELECT COUNT(*) FROM requests c WHERE c.parent_request_id = r.id), 0) AS total_children
		FROM requests r
		WHERE r.id = $1
	`, requestID)

	summary, err := scanRequestSummary(row)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func queryRequestSummaries(ctx context.Context, query string, args ...interface{}) ([]RequestSummary, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to query requests").Cause(err).Err()
	}
	defer rows.Close()

	items := []RequestSummary{}
	for rows.Next() {
		item, err := scanRequestSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to read requests").Cause(err).Err()
	}
	return items, nil
}

func queryRequestEmployees(ctx context.Context, requestID uuid.UUID) ([]RequestEmployee, error) {
	rows, err := db.Query(ctx, `
		SELECT e.id, e.full_name, e.dzo_id, d.name
		FROM request_participants rp
		JOIN employees e ON e.id = rp.employee_id
		JOIN dzo_organizations d ON d.id = e.dzo_id
		WHERE rp.request_id = $1
		ORDER BY e.full_name ASC
	`, requestID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request employees").Cause(err).Err()
	}
	defer rows.Close()

	employees := []RequestEmployee{}
	for rows.Next() {
		var (
			id       uuid.UUID
			fullName string
			dzoID    uuid.UUID
			dzoName  string
		)
		if err := rows.Scan(&id, &fullName, &dzoID, &dzoName); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan request employee").Cause(err).Err()
		}
		employees = append(employees, RequestEmployee{
			ID:       id.String(),
			FullName: fullName,
			DzoID:    dzoID.String(),
			DzoName:  dzoName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request employees").Cause(err).Err()
	}

	return employees, nil
}

func queryRequestTargetDZOs(ctx context.Context, requestID uuid.UUID) ([]RequestTargetDZO, error) {
	rows, err := db.Query(ctx, `
		SELECT d.id, d.name
		FROM request_target_dzos rtd
		JOIN dzo_organizations d ON d.id = rtd.dzo_id
		WHERE rtd.request_id = $1
		ORDER BY d.name ASC
	`, requestID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request target DZO").Cause(err).Err()
	}
	defer rows.Close()

	items := []RequestTargetDZO{}
	for rows.Next() {
		var (
			id   uuid.UUID
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan request target DZO").Cause(err).Err()
		}
		items = append(items, RequestTargetDZO{ID: id.String(), Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request target DZO").Cause(err).Err()
	}

	return items, nil
}

func queryRequestDzoContracts(ctx context.Context, requestID uuid.UUID) ([]RequestContractResponse, error) {
	rows, err := db.Query(ctx, `
		SELECT dzo_id, file_name, file_url
		FROM request_dzo_contracts
		WHERE request_id = $1
		ORDER BY created_at ASC
	`, requestID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request contracts").Cause(err).Err()
	}
	defer rows.Close()

	items := []RequestContractResponse{}
	for rows.Next() {
		var (
			dzoID    uuid.UUID
			fileName string
			fileURL  string
		)
		if err := rows.Scan(&dzoID, &fileName, &fileURL); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan request contract").Cause(err).Err()
		}
		items = append(items, RequestContractResponse{
			DzoID:    dzoID,
			FileName: fileName,
			FileURL:  fileURL,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request contracts").Cause(err).Err()
	}

	return items, nil
}
func queryRequestTargetDZOIDs(ctx context.Context, requestID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, `
		SELECT dzo_id
		FROM request_target_dzos
		WHERE request_id = $1
	`, requestID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request target DZO").Cause(err).Err()
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan request target DZO").Cause(err).Err()
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load request target DZO").Cause(err).Err()
	}
	return ids, nil
}

func loadTrainingEvent(ctx context.Context, rawID string) (*trainingEventRecord, error) {
	trainingID, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid training_event_id format").Err()
	}

	row := db.QueryRow(ctx, `
		SELECT id, title, start_date, location_type
		FROM training_events
		WHERE id = $1
	`, trainingID)

	var event trainingEventRecord
	if err := row.Scan(&event.ID, &event.Title, &event.StartDate, &event.LocationType); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.NotFound).Msg("training event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to load training event").Cause(err).Err()
	}
	return &event, nil
}

func resolveSelectedEmployees(ctx context.Context, rawIDs []string) ([]employeeRecord, []uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{}, len(rawIDs))
	employees := make([]employeeRecord, 0, len(rawIDs))
	ids := make([]uuid.UUID, 0, len(rawIDs))

	for _, rawID := range rawIDs {
		employeeID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, nil, errs.B().Code(errs.InvalidArgument).Msg("invalid employee_id format").Err()
		}
		if _, ok := seen[employeeID]; ok {
			continue
		}
		seen[employeeID] = struct{}{}

		row := db.QueryRow(ctx, `
			SELECT e.id, e.full_name, e.dzo_id, d.name
			FROM employees e
			JOIN dzo_organizations d ON d.id = e.dzo_id
			WHERE e.id = $1 AND e.is_active = TRUE AND e.is_deleted = FALSE
		`, employeeID)

		var employee employeeRecord
		if err := row.Scan(&employee.ID, &employee.FullName, &employee.DzoID, &employee.DzoName); err != nil {
			if errors.Is(err, sqldb.ErrNoRows) {
				return nil, nil, errs.B().Code(errs.InvalidArgument).Msg("employee not found").Err()
			}
			return nil, nil, errs.B().Code(errs.Internal).Msg("failed to load employee").Cause(err).Err()
		}

		employees = append(employees, employee)
		ids = append(ids, employeeID)
	}

	return employees, ids, nil
}

// helper проверки сотрудников (CreateHRRequest creates a main request from HR to Admin)
func resolveHREmployeesForRequest(ctx context.Context, rawIDs []string, hrDzoID uuid.UUID, allowInactive bool) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{}, len(rawIDs))
	ids := make([]uuid.UUID, 0, len(rawIDs))

	for _, rawID := range rawIDs {
		employeeID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid employee_id format").Err()
		}
		if _, ok := seen[employeeID]; ok {
			continue
		}
		seen[employeeID] = struct{}{}

		var (
			dzoID     uuid.UUID
			isActive  bool
			isDeleted bool
		)

		if err := db.QueryRow(ctx, `
			SELECT dzo_id, is_active, is_deleted
			FROM employees
			WHERE id = $1
		`, employeeID).Scan(&dzoID, &isActive, &isDeleted); err != nil {
			if errors.Is(err, sqldb.ErrNoRows) {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("employee not found").Err()
			}
			return nil, errs.B().Code(errs.Internal).Msg("failed to load employee").Cause(err).Err()
		}

		if dzoID != hrDzoID {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("employee does not belong to HR DZO").Err()
		}

		if (!isActive || isDeleted) && !allowInactive {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("employee is inactive").Err()
		}

		ids = append(ids, employeeID)
	}

	return ids, nil
}
func resolveSelectedDZOs(ctx context.Context, rawIDs []string) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{}, len(rawIDs))
	dzoIDs := make([]uuid.UUID, 0, len(rawIDs))

	for _, rawID := range rawIDs {
		dzoID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
		if _, ok := seen[dzoID]; ok {
			continue
		}
		seen[dzoID] = struct{}{}

		var exists bool
		if err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM dzo_organizations
				WHERE id = $1 AND is_active = TRUE
			)
		`, dzoID).Scan(&exists); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to validate dzo").Cause(err).Err()
		}
		if !exists {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo not found").Err()
		}

		dzoIDs = append(dzoIDs, dzoID)
	}

	return dzoIDs, nil
}

func resolveHRByDZOs(ctx context.Context, dzoIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	result := make(map[uuid.UUID]uuid.UUID, len(dzoIDs))
	for _, dzoID := range dzoIDs {
		row := db.QueryRow(ctx, `
			SELECT id
			FROM users
			WHERE role = 'HR' AND dzo_id = $1 AND is_active = TRUE
			ORDER BY created_at ASC
			LIMIT 1
		`, dzoID)

		var hrID uuid.UUID
		if err := row.Scan(&hrID); err != nil {
			if errors.Is(err, sqldb.ErrNoRows) {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("hr not found for selected dzo").Err()
			}
			return nil, errs.B().Code(errs.Internal).Msg("failed to resolve hr").Cause(err).Err()
		}
		result[dzoID] = hrID
	}
	return result, nil
}

func replaceRequestParticipantsTx(ctx context.Context, tx *sqldb.Tx, requestID uuid.UUID, employeeIDs []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM request_participants
		WHERE request_id = $1
	`, requestID); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to replace request participants").Cause(err).Err()
	}

	for _, employeeID := range employeeIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO request_participants (id, request_id, employee_id, created_at)
			VALUES ($1, $2, $3, NOW())
		`, uuid.New(), requestID, employeeID); err != nil {
			return errs.B().Code(errs.Internal).Msg("failed to save request participant").Cause(err).Err()
		}
	}

	return nil
}

func replaceRequestTargetDZOsTx(ctx context.Context, tx *sqldb.Tx, requestID uuid.UUID, dzoIDs []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM request_target_dzos
		WHERE request_id = $1
	`, requestID); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to replace request target DZO").Cause(err).Err()
	}

	for _, dzoID := range dzoIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO request_target_dzos (id, request_id, dzo_id, created_at)
			VALUES ($1, $2, $3, NOW())
		`, uuid.New(), requestID, dzoID); err != nil {
			return errs.B().Code(errs.Internal).Msg("failed to save request target DZO").Cause(err).Err()
		}
	}

	return nil
}

func syncParentRequestStatusTx(ctx context.Context, tx *sqldb.Tx, parentID uuid.UUID) error {
	row := tx.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_children,
			COUNT(*) FILTER (WHERE status = 'PENDING') AS pending_children,
			COUNT(*) FILTER (WHERE status = 'APPROVED') AS approved_children
		FROM requests
		WHERE parent_request_id = $1
	`, parentID)

	var totalChildren, pendingChildren, approvedChildren int
	if err := row.Scan(&totalChildren, &pendingChildren, &approvedChildren); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to aggregate child requests").Cause(err).Err()
	}
	if totalChildren == 0 {
		return nil
	}

	status := RequestStatusInProgress
	switch {
	case pendingChildren > 0:
		status = RequestStatusInProgress
	case approvedChildren > 0:
		status = RequestStatusApproved
	default:
		status = RequestStatusRejected
	}

	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, parentID, string(status)); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to sync parent request").Cause(err).Err()
	}
	return nil
}

func queryParticipantCount(ctx context.Context, requestID uuid.UUID) (int, error) {
	var count int
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM request_participants
		WHERE request_id = $1
	`, requestID).Scan(&count); err != nil {
		return 0, errs.B().Code(errs.Internal).Msg("failed to count request participants").Cause(err).Err()
	}
	return count, nil
}

func finalizeAdminRequest(ctx context.Context, actor *actor, requestID uuid.UUID) (*RequestDetail, error) {
	requestSummary, err := queryRequestSummaryByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if requestSummary.RequestType != RequestTypeMain {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("only main requests can be finalized").Err()
	}

	if requestSummary.Status == RequestStatusRejected {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("cancelled request cannot be finalized").Err()
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	var pendingChildren int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM requests
		WHERE parent_request_id = $1 AND status = 'PENDING'
	`, requestID).Scan(&pendingChildren); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to check pending subrequests").Cause(err).Err()
	}

	if pendingChildren > 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("not all HR subrequests are finalized").Err()
	}

	hasReserve, err := budgetOperationExistsTx(ctx, tx, requestID, "RESERVE")
	if err != nil {
		return nil, err
	}

	if !hasReserve {
		if err := reserveBudgetTx(ctx, tx, requestID, actor.ID); err != nil {
			return nil, err
		}
	}

	if err := writeOffBudgetTx(ctx, tx, requestID, actor.ID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET status = 'IN_PROGRESS',
			responsible_admin_id = $2,
			updated_at = NOW()
		WHERE id = $1
	`, requestID, actor.ID); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to finalize request").Cause(err).Err()
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit admin finalization").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}

// cancel admin
func cancelAdminRequest(ctx context.Context, actor *actor, requestID uuid.UUID) (*RequestDetail, error) {
	requestSummary, err := queryRequestSummaryByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if requestSummary.RequestType != RequestTypeMain {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("only main requests can be cancelled by admin").Err()
	}

	if requestSummary.Status == RequestStatusRejected {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request is already cancelled").Err()
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET status = 'REJECTED', updated_at = NOW()
		WHERE id = $1
	`, requestID); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to cancel request").Cause(err).Err()
	}

	if _, err := tx.Exec(ctx, `
		UPDATE requests
		SET status = 'REJECTED', updated_at = NOW()
		WHERE parent_request_id = $1 AND status = 'PENDING'
	`, requestID); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to cancel child requests").Cause(err).Err()
	}

	//refund
	if err := refundBudgetTx(ctx, tx, requestID, actor.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit request cancellation").Cause(err).Err()
	}

	return buildRequestDetail(ctx, requestID)
}

func scanRequestSummary(s rowScanner) (*RequestSummary, error) {
	var (
		id                 uuid.UUID
		initiatorID        uuid.UUID
		parentRequestID    sql.NullString
		entityID           uuid.UUID
		entityType         string
		requestType        string
		kind               string
		assignedHRID       sql.NullString
		targetDzoID        sql.NullString
		title              sql.NullString
		category           sql.NullString
		format             sql.NullString
		responsibleAdminID sql.NullString
		trainingDate       sql.NullTime
		deadlineAt         sql.NullTime
		costAmount         sql.NullFloat64
		costMode           sql.NullString
		status             string
		createdAt          time.Time
		updatedAt          time.Time
		completedAt        sql.NullTime
		employeesCount     int
		approvedChildren   int
		totalChildren      int
	)

	if err := s.Scan(
		&id,
		&initiatorID,
		&parentRequestID,
		&entityID,
		&entityType,
		&requestType,
		&kind,
		&assignedHRID,
		&targetDzoID,
		&title,
		&category,
		&format,
		&responsibleAdminID,
		&trainingDate,
		&deadlineAt,
		&costAmount,
		&costMode,
		&status,
		&createdAt,
		&updatedAt,
		&completedAt,
		&employeesCount,
		&approvedChildren,
		&totalChildren,
	); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.NotFound).Msg("request not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to scan request").Cause(err).Err()
	}

	return &RequestSummary{
		ID:                 id.String(),
		InitiatorID:        initiatorID.String(),
		ParentRequestID:    nullableStringValue(parentRequestID),
		TrainingEventID:    entityID.String(),
		EntityType:         entityType,
		RequestType:        RequestType(requestType),
		Kind:               RequestKind(kind),
		Status:             RequestStatus(status),
		AssignedHRID:       nullableStringValue(assignedHRID),
		TargetDzoID:        nullableStringValue(targetDzoID),
		Title:              title.String,
		Category:           nullableStringValue(category),
		Format:             nullableStringValue(format),
		ResponsibleAdminID: nullableStringValue(responsibleAdminID),
		TrainingDate:       nullableTimeValue(trainingDate),
		DeadlineAt:         nullableTimeValue(deadlineAt),
		CostAmount:         nullableFloatValue(costAmount),
		CostMode:           nullableCostModeValue(costMode),
		EmployeesCount:     employeesCount,
		ApprovedChildren:   approvedChildren,
		TotalChildren:      totalChildren,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		CompletedAt:        nullableTimeValue(completedAt),
	}, nil
}

func collectAffectedDZOs(participants []employeeRecord, dzoIDs []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	result := make([]uuid.UUID, 0, len(participants)+len(dzoIDs))
	for _, dzoID := range dzoIDs {
		if _, ok := seen[dzoID]; ok {
			continue
		}
		seen[dzoID] = struct{}{}
		result = append(result, dzoID)
	}
	for _, participant := range participants {
		if _, ok := seen[participant.DzoID]; ok {
			continue
		}
		seen[participant.DzoID] = struct{}{}
		result = append(result, participant.DzoID)
	}
	return result
}

func mapsKeys[T comparable, V any](m map[T]V) []T {
	keys := make([]T, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func parseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("deadline_at must be RFC3339").Err()
	}
	return &parsed, nil
}

func nullableString(v *string) interface{} {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableFloat(v *float64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullableCostMode(v *CostMode) interface{} {
	if v == nil {
		return nil
	}
	return string(*v)
}

func nullableStringValue(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}

func nullableTimeValue(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	value := v.Time
	return &value
}

func nullableFloatValue(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	value := v.Float64
	return &value
}

func nullableCostModeValue(v sql.NullString) *CostMode {
	if !v.Valid {
		return nil
	}
	value := CostMode(v.String)
	return &value
}

func parseUUIDOrNil(raw string) uuid.UUID {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil
	}
	return id
}

func parseUUIDOrNilPtr(raw *string) interface{} {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return nil
	}
	return id
}

// ════ ARCHIVE FLOW ════

// CreateArchiveRequest creates a closed/archive request that is immediately completed.
//
//encore:api auth method=POST path=/requests/archive
func CreateArchiveRequest(ctx context.Context, req *CreateArchiveRequestRequest) (*GetRequestResponse, error) {
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
		SetRequestType(string(RequestTypeMain)).
		SetKind(kind).
		SetCategory(strings.TrimSpace(req.Category)).
		SetStep(0).
		SetStatus(string(RequestStatusCompleted)).
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

		if _, err := tx.RequestParticipant.
			Create().
			SetRequestID(created.ID).
			SetEmployeeID(employeeID).
			Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, errs.B().Code(errs.Internal).Msg("failed to save request participants").Cause(err).Err()
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

	detail, err := buildRequestDetail(ctx, created.ID)
	if err != nil {
		return nil, err
	}

	return &GetRequestResponse{Detail: *detail}, nil
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

func normalizeRequestKind(kind string) string {
	return strings.ToUpper(strings.TrimSpace(kind))
}

func isAdminRole(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case string(authhandler.RoleSA), string(authhandler.RoleADM):
		return true
	default:
		return false
	}
}

func getRequestBudgetSource(ctx context.Context, requestID uuid.UUID) (uuid.UUID, float64, error) {
	row := db.QueryRow(ctx, `
		SELECT te.supplier_contract_id, r.cost_amount
		FROM requests r
		JOIN training_events te ON te.id = r.entity_id
		WHERE r.id = $1
	`, requestID)

	var (
		contractID uuid.UUID
		costAmount sql.NullFloat64
	)

	if err := row.Scan(&contractID, &costAmount); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return uuid.Nil, 0, errs.B().Code(errs.NotFound).Msg("request budget source not found").Err()
		}
		return uuid.Nil, 0, errs.B().Code(errs.Internal).Msg("failed to load request budget source").Cause(err).Err()
	}

	if !costAmount.Valid || costAmount.Float64 <= 0 {
		return contractID, 0, nil
	}

	return contractID, costAmount.Float64, nil
}

func reserveBudgetTx(ctx context.Context, tx *sqldb.Tx, requestID uuid.UUID, actorID uuid.UUID) error {
	contractID, amount, err := getRequestBudgetSource(ctx, requestID)
	if err != nil {
		return err
	}
	if amount <= 0 {
		return nil
	}

	var remaining float64
	if err := tx.QueryRow(ctx, `
		SELECT remaining_amount
		FROM contract_suppliers
		WHERE id = $1
		FOR UPDATE
	`, contractID).Scan(&remaining); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to lock contract budget").Cause(err).Err()
	}

	if remaining < amount {
		return errs.B().Code(errs.InvalidArgument).Msg("not enough budget").Err()
	}

	if _, err := tx.Exec(ctx, `
		UPDATE contract_suppliers
		SET remaining_amount = remaining_amount - $2, updated_at = NOW()
		WHERE id = $1
	`, contractID, amount); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to reserve budget").Cause(err).Err()
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO request_budget_transactions (
			id, request_id, contract_id, amount, operation_type, created_by, reason, created_at
		)
		VALUES ($1, $2, $3, $4, 'RESERVE', $5, 'Budget reserved on request submit', NOW())
	`, uuid.New(), requestID, contractID, amount, actorID); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to log budget reserve").Cause(err).Err()
	}

	return nil
}

func writeOffBudgetTx(ctx context.Context, tx *sqldb.Tx, requestID uuid.UUID, actorID uuid.UUID) error {
	contractID, amount, err := getRequestBudgetSource(ctx, requestID)
	if err != nil {
		return err
	}
	if amount <= 0 {
		return nil
	}

	var alreadyWrittenOff bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM request_budget_transactions
			WHERE request_id = $1 AND operation_type = 'WRITE_OFF'
		)
	`, requestID).Scan(&alreadyWrittenOff); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to check budget write-off").Cause(err).Err()
	}
	if alreadyWrittenOff {
		return nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO request_budget_transactions (
			id, request_id, contract_id, amount, operation_type, created_by, reason, created_at
		)
		VALUES ($1, $2, $3, $4, 'WRITE_OFF', $5, 'Budget written off on final approve', NOW())
	`, uuid.New(), requestID, contractID, amount, actorID); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to log budget write-off").Cause(err).Err()
	}

	return nil
}

func refundBudgetTx(ctx context.Context, tx *sqldb.Tx, requestID uuid.UUID, actorID uuid.UUID) error {
	contractID, amount, err := getRequestBudgetSource(ctx, requestID)
	if err != nil {
		return err
	}
	if amount <= 0 {
		return nil
	}

	var wasWriteOff bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM request_budget_transactions
			WHERE request_id = $1 AND operation_type = 'WRITE_OFF'
		)
	`, requestID).Scan(&wasWriteOff); err != nil {
		return err
	}

	var wasReserve bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM request_budget_transactions
			WHERE request_id = $1 AND operation_type = 'RESERVE'
		)
	`, requestID).Scan(&wasReserve); err != nil {
		return err
	}

	// 1. если был write-off → REFUND
	if wasWriteOff {
		_, err := tx.Exec(ctx, `
			UPDATE contract_suppliers
			SET remaining_amount = remaining_amount + $2, updated_at = NOW()
			WHERE id = $1
		`, contractID, amount)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
	INSERT INTO request_budget_transactions (
		id, request_id, contract_id, amount, operation_type, created_by, reason, created_at
	)
	VALUES ($1, $2, $3, $4, 'REFUND', $5, 'Budget refunded on request cancel after write-off', NOW())
`, uuid.New(), requestID, contractID, amount, actorID)
		return err
	}

	// 2. если был reserve → RELEASE
	if wasReserve {
		_, err := tx.Exec(ctx, `
			UPDATE contract_suppliers
			SET remaining_amount = remaining_amount + $2, updated_at = NOW()
			WHERE id = $1
		`, contractID, amount)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
	INSERT INTO request_budget_transactions (
		id, request_id, contract_id, amount, operation_type, created_by, reason, created_at
	)
	VALUES ($1, $2, $3, $4, 'RELEASE', $5, 'Budget reserve released on request cancel', NOW())
`, uuid.New(), requestID, contractID, amount, actorID)
		return err
	}

	return nil
}
func budgetOperationExistsTx(ctx context.Context, tx *sqldb.Tx, requestID uuid.UUID, operationType string) (bool, error) {
	var exists bool

	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM request_budget_transactions
			WHERE request_id = $1 AND operation_type = $2
		)
	`, requestID, operationType).Scan(&exists); err != nil {
		return false, errs.B().Code(errs.Internal).Msg("failed to check budget operation").Cause(err).Err()
	}

	return exists, nil
}
