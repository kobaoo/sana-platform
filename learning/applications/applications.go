package applications

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"github.com/google/uuid"

	"encore.app/auth/authhandler"
)

// ════ DATABASE ════

var db = sqldb.Named("lms")

// ════ ENDPOINTS ════

// CreateApplication creates a new training application.
//
//encore:api auth method=POST path=/applications
func CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanCreateApplication(caller.Role, req.Kind) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	if !req.Kind.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid application kind").Err()
	}

	row, err := insertApplication(ctx, caller, req)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *row}, nil
}

// ListApplications returns applications visible to the current caller.
//
//encore:api auth method=GET path=/applications
func ListApplications(ctx context.Context) (*ListApplicationsResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !CanListApplications(caller.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	rows, err := queryApplicationsForCaller(ctx, caller)
	if err != nil {
		return nil, err
	}

	return &ListApplicationsResponse{
		Applications: rows,
		Total:        len(rows),
	}, nil
}

// GetApplication returns a single application.
//
//encore:api auth method=GET path=/applications/:id
func GetApplication(ctx context.Context, id string) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}

	row, err := queryApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanAccessApplication(caller, &row.Application) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("application is outside your scope").Err()
	}

	return row, nil
}

// UpdateApplication updates a draft application.
//
//encore:api auth method=PUT path=/applications/:id
func UpdateApplication(ctx context.Context, id string, req *UpdateApplicationRequest) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}

	row, err := updateApplication(ctx, caller, id, req)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *row}, nil
}

// SubmitApplication moves a draft application to submitted.
//
//encore:api auth method=POST path=/applications/:id/submit
func SubmitApplication(ctx context.Context, id string) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}

	row, err := transitionApplication(ctx, caller, id, ApplicationStatusSubmitted)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *row}, nil
}

// StartApplication moves an application into processing.
//
//encore:api auth method=POST path=/applications/:id/start
func StartApplication(ctx context.Context, id string) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}

	row, err := transitionApplication(ctx, caller, id, ApplicationStatusInProcess)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *row}, nil
}

// CompleteApplication completes an application.
//
//encore:api auth method=POST path=/applications/:id/complete
func CompleteApplication(ctx context.Context, id string) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}

	row, err := transitionApplication(ctx, caller, id, ApplicationStatusCompleted)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *row}, nil
}

// CancelApplication cancels an application.
//
//encore:api auth method=POST path=/applications/:id/cancel
func CancelApplication(ctx context.Context, id string) (*GetApplicationResponse, error) {
	caller, err := resolveCaller(ctx)
	if err != nil {
		return nil, err
	}

	row, err := transitionApplication(ctx, caller, id, ApplicationStatusCancelled)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *row}, nil
}

// ════ INTERNAL ════

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func resolveCaller(ctx context.Context) (*Caller, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	caller := &Caller{Role: ad.Role}
	if strings.TrimSpace(ad.DzoID) != "" {
		dzoID := strings.TrimSpace(ad.DzoID)
		caller.DzoID = &dzoID
	}

	row := db.QueryRow(ctx, `
		SELECT id, dzo_id
		FROM users
		WHERE keycloak_user_id = $1
	`, ad.KeycloakUserID)

	var (
		userID uuid.UUID
		dzoID  sql.NullString
	)
	if err := row.Scan(&userID, &dzoID); err != nil {
		if !errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.Internal).Msg("failed to resolve caller").Cause(err).Err()
		}
		return caller, nil
	}

	caller.UserID = userID.String()
	if dzoID.Valid {
		value := dzoID.String
		caller.DzoID = &value
	}
	return caller, nil
}

func CanListApplications(role authhandler.UserRole) bool {
	return role == authhandler.RoleSA || role == authhandler.RoleADM || role == authhandler.RoleHR
}

func CanCreateApplication(role authhandler.UserRole, kind ApplicationKind) bool {
	if kind == "" {
		kind = ApplicationKindRegular
	}

	switch role {
	case authhandler.RoleSA, authhandler.RoleADM:
		return true
	case authhandler.RoleHR:
		return kind == ApplicationKindRegular
	default:
		return false
	}
}

func CanAccessApplication(caller *Caller, app *Application) bool {
	if caller.Role == authhandler.RoleSA {
		return true
	}
	if caller.UserID != "" && app.CreatedByUserID != nil && *app.CreatedByUserID == caller.UserID {
		return true
	}
	if caller.DzoID != nil && app.DzoID != nil {
		return *caller.DzoID == *app.DzoID
	}
	return false
}

func CanEditApplication(caller *Caller, app *Application) bool {
	if !CanAccessApplication(caller, app) {
		return false
	}
	if caller.Role != authhandler.RoleSA && caller.Role != authhandler.RoleADM && caller.Role != authhandler.RoleHR {
		return false
	}
	return app.Kind == ApplicationKindRegular && app.Status == ApplicationStatusDraft
}

func CanTransitionApplication(caller *Caller, app *Application, next ApplicationStatus) bool {
	if !CanAccessApplication(caller, app) {
		return false
	}

	switch next {
	case ApplicationStatusSubmitted:
		return caller.Role == authhandler.RoleSA || caller.Role == authhandler.RoleADM || caller.Role == authhandler.RoleHR
	case ApplicationStatusInProcess, ApplicationStatusCompleted:
		return caller.Role == authhandler.RoleSA || caller.Role == authhandler.RoleADM
	case ApplicationStatusCancelled:
		return caller.Role == authhandler.RoleSA || caller.Role == authhandler.RoleADM || caller.Role == authhandler.RoleHR
	default:
		return false
	}
}

func initialStatusForKind(kind ApplicationKind) ApplicationStatus {
	if kind == ApplicationKindClosed || kind == ApplicationKindArchived {
		return ApplicationStatusCompleted
	}
	return ApplicationStatusDraft
}

func validateStatusTransition(kind ApplicationKind, current, next ApplicationStatus) error {
	if isAllowedStatusTransition(kind, current, next) {
		return nil
	}
	if kind == ApplicationKindClosed || kind == ApplicationKindArchived {
		return errs.B().Code(errs.InvalidArgument).Msg("closed and archived applications are final").Err()
	}
	return errs.B().Code(errs.InvalidArgument).Msg("invalid application status transition").Err()
}

func isAllowedStatusTransition(kind ApplicationKind, current, next ApplicationStatus) bool {
	if kind == ApplicationKindClosed || kind == ApplicationKindArchived {
		return false
	}

	switch current {
	case ApplicationStatusDraft:
		return next == ApplicationStatusSubmitted || next == ApplicationStatusCancelled
	case ApplicationStatusSubmitted:
		return next == ApplicationStatusInProcess || next == ApplicationStatusCompleted || next == ApplicationStatusCancelled
	case ApplicationStatusInProcess:
		return next == ApplicationStatusCompleted || next == ApplicationStatusCancelled
	default:
		return false
	}
}

func insertApplication(ctx context.Context, caller *Caller, req *CreateApplicationRequest) (*Application, error) {
	kind := req.Kind
	if kind == "" {
		kind = ApplicationKindRegular
	}

	courseID, courseName, err := normalizeCourseSelection(ctx, req.CourseID, req.RequestedCourseName)
	if err != nil {
		return nil, err
	}
	employeeIDs, err := validateEmployeeIDs(ctx, req.EmployeeIDs)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to start transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	applicationID := uuid.New()
	row := tx.QueryRow(ctx, `
		INSERT INTO applications (
			id, kind, status, dzo_id, created_by_user_id, course_id,
			requested_course_name, expense_category, comment, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, NOW(), NOW())
		RETURNING id, kind, status, dzo_id, created_by_user_id, course_id,
			requested_course_name, expense_category, comment, is_active, created_at, updated_at
	`,
		applicationID,
		string(kind),
		string(initialStatusForKind(kind)),
		nullableUUID(caller.DzoID),
		nullableUUID(pointerOrNil(caller.UserID)),
		courseID,
		courseName,
		nullableString(req.ExpenseCategory),
		nullableString(req.Comment),
	)

	app, err := scanApplication(row)
	if err != nil {
		return nil, err
	}
	if err := replaceParticipantsTx(ctx, tx, applicationID, employeeIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit transaction").Cause(err).Err()
	}

	app.EmployeeIDs = employeeIDs
	return app, nil
}

func queryApplicationsForCaller(ctx context.Context, caller *Caller) ([]Application, error) {
	query := `
		SELECT id, kind, status, dzo_id, created_by_user_id, course_id,
			requested_course_name, expense_category, comment, is_active, created_at, updated_at
		FROM applications
		WHERE is_active = TRUE
	`
	args := []interface{}{}

	switch caller.Role {
	case authhandler.RoleSA:
	case authhandler.RoleADM, authhandler.RoleHR:
		if caller.DzoID != nil {
			query += " AND dzo_id = $1"
			args = append(args, nullableUUID(caller.DzoID))
		} else if caller.UserID != "" {
			query += " AND created_by_user_id = $1"
			args = append(args, nullableUUID(pointerOrNil(caller.UserID)))
		} else {
			return []Application{}, nil
		}
	default:
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list applications").Cause(err).Err()
	}
	defer rows.Close()

	apps := []Application{}
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		app.EmployeeIDs, err = loadParticipantIDs(ctx, app.ID)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *app)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list applications").Cause(err).Err()
	}

	return apps, nil
}

func queryApplicationByID(ctx context.Context, id string) (*GetApplicationResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row := db.QueryRow(ctx, `
		SELECT id, kind, status, dzo_id, created_by_user_id, course_id,
			requested_course_name, expense_category, comment, is_active, created_at, updated_at
		FROM applications
		WHERE id = $1
	`, uid)

	app, err := scanApplication(row)
	if err != nil {
		return nil, err
	}
	app.EmployeeIDs, err = loadParticipantIDs(ctx, app.ID)
	if err != nil {
		return nil, err
	}

	return &GetApplicationResponse{Application: *app}, nil
}

func updateApplication(ctx context.Context, caller *Caller, id string, req *UpdateApplicationRequest) (*Application, error) {
	current, err := queryApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanEditApplication(caller, &current.Application) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("application cannot be edited").Err()
	}

	courseID, courseName, err := normalizeCourseSelectionForUpdate(ctx, req.CourseID, req.RequestedCourseName, &current.Application)
	if err != nil {
		return nil, err
	}

	var employeeIDs []string
	if req.EmployeeIDs != nil {
		employeeIDs, err = validateEmployeeIDs(ctx, *req.EmployeeIDs)
		if err != nil {
			return nil, err
		}
	}

	uid, _ := uuid.Parse(id)
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to start transaction").Cause(err).Err()
	}
	defer tx.Rollback()

	row := tx.QueryRow(ctx, `
		UPDATE applications
		SET
			course_id = $2,
			requested_course_name = $3,
			expense_category = CASE
				WHEN $4 THEN NULL
				ELSE COALESCE($5, expense_category)
			END,
			comment = CASE
				WHEN $6 THEN NULL
				ELSE COALESCE($7, comment)
			END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, kind, status, dzo_id, created_by_user_id, course_id,
			requested_course_name, expense_category, comment, is_active, created_at, updated_at
	`,
		uid,
		courseID,
		courseName,
		shouldClearOptionalString(req.ExpenseCategory),
		nullableString(req.ExpenseCategory),
		shouldClearOptionalString(req.Comment),
		nullableString(req.Comment),
	)

	app, err := scanApplication(row)
	if err != nil {
		return nil, err
	}

	if req.EmployeeIDs != nil {
		if err := replaceParticipantsTx(ctx, tx, uid, employeeIDs); err != nil {
			return nil, err
		}
		app.EmployeeIDs = employeeIDs
	} else {
		app.EmployeeIDs = current.Application.EmployeeIDs
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to commit transaction").Cause(err).Err()
	}

	return app, nil
}

func transitionApplication(ctx context.Context, caller *Caller, id string, next ApplicationStatus) (*Application, error) {
	current, err := queryApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanTransitionApplication(caller, &current.Application, next) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("status transition is not allowed").Err()
	}
	if err := validateStatusTransition(current.Application.Kind, current.Application.Status, next); err != nil {
		return nil, err
	}
	if next == ApplicationStatusSubmitted {
		if _, err := validateEmployeeIDs(ctx, current.Application.EmployeeIDs); err != nil {
			return nil, err
		}
	}

	uid, _ := uuid.Parse(id)
	row := db.QueryRow(ctx, `
		UPDATE applications
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, kind, status, dzo_id, created_by_user_id, course_id,
			requested_course_name, expense_category, comment, is_active, created_at, updated_at
	`, uid, string(next))

	app, err := scanApplication(row)
	if err != nil {
		return nil, err
	}
	app.EmployeeIDs = current.Application.EmployeeIDs
	return app, nil
}

func normalizeCourseSelection(ctx context.Context, courseIDStr *string, requestedCourseName string) (*uuid.UUID, string, error) {
	trimmedName := strings.TrimSpace(requestedCourseName)
	if courseIDStr == nil || strings.TrimSpace(*courseIDStr) == "" {
		if trimmedName == "" {
			return nil, "", errs.B().Code(errs.InvalidArgument).Msg("requested_course_name is required").Err()
		}
		return nil, trimmedName, nil
	}

	courseID, err := uuid.Parse(strings.TrimSpace(*courseIDStr))
	if err != nil {
		return nil, "", errs.B().Code(errs.InvalidArgument).Msg("invalid course_id format").Err()
	}

	var title string
	err = db.QueryRow(ctx, `
		SELECT title
		FROM courses
		WHERE id = $1 AND is_active = TRUE
	`, courseID).Scan(&title)
	if err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, "", errs.B().Code(errs.NotFound).Msg("course not found").Err()
		}
		return nil, "", errs.B().Code(errs.Internal).Msg("failed to validate course").Cause(err).Err()
	}

	if trimmedName == "" {
		trimmedName = title
	}
	return &courseID, trimmedName, nil
}

func normalizeCourseSelectionForUpdate(ctx context.Context, courseIDStr *string, requestedCourseName *string, current *Application) (*uuid.UUID, string, error) {
	name := current.RequestedCourseName
	if requestedCourseName != nil {
		name = strings.TrimSpace(*requestedCourseName)
	}

	if courseIDStr == nil {
		if name == "" {
			return nil, "", errs.B().Code(errs.InvalidArgument).Msg("requested_course_name is required").Err()
		}
		if current.CourseID == nil {
			return nil, name, nil
		}
		existingID, err := uuid.Parse(*current.CourseID)
		if err != nil {
			return nil, "", errs.B().Code(errs.InvalidArgument).Msg("invalid course_id format").Err()
		}
		return &existingID, name, nil
	}

	if strings.TrimSpace(*courseIDStr) == "" {
		if name == "" {
			return nil, "", errs.B().Code(errs.InvalidArgument).Msg("requested_course_name is required").Err()
		}
		return nil, name, nil
	}

	return normalizeCourseSelection(ctx, courseIDStr, name)
}

func validateEmployeeIDs(ctx context.Context, employeeIDs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(employeeIDs))
	validIDs := []string{}

	for _, rawID := range employeeIDs {
		trimmed := strings.TrimSpace(rawID)
		uid, err := uuid.Parse(trimmed)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid employee_id format").Err()
		}
		if _, ok := seen[uid.String()]; ok {
			continue
		}

		var exists bool
		err = db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM users
				WHERE id = $1 AND is_active = TRUE
			)
		`, uid).Scan(&exists)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to validate employees").Cause(err).Err()
		}
		if !exists {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("all employees must exist and be active").Err()
		}

		seen[uid.String()] = struct{}{}
		validIDs = append(validIDs, uid.String())
	}

	return validIDs, nil
}

func replaceParticipantsTx(ctx context.Context, tx *sqldb.Tx, applicationID uuid.UUID, employeeIDs []string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM application_participants
		WHERE application_id = $1
	`, applicationID); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to replace application participants").Cause(err).Err()
	}

	for _, employeeID := range employeeIDs {
		uid, err := uuid.Parse(employeeID)
		if err != nil {
			return errs.B().Code(errs.InvalidArgument).Msg("invalid employee_id format").Err()
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO application_participants (id, application_id, user_id, created_at)
			VALUES ($1, $2, $3, NOW())
		`, uuid.New(), applicationID, uid); err != nil {
			return errs.B().Code(errs.Internal).Msg("failed to save application participants").Cause(err).Err()
		}
	}

	return nil
}

func loadParticipantIDs(ctx context.Context, applicationID string) ([]string, error) {
	uid, err := uuid.Parse(applicationID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	rows, err := db.Query(ctx, `
		SELECT user_id
		FROM application_participants
		WHERE application_id = $1
		ORDER BY created_at ASC
	`, uid)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load application participants").Cause(err).Err()
	}
	defer rows.Close()

	employeeIDs := []string{}
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to scan application participant").Cause(err).Err()
		}
		employeeIDs = append(employeeIDs, userID.String())
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to load application participants").Cause(err).Err()
	}

	return employeeIDs, nil
}

func scanApplication(s rowScanner) (*Application, error) {
	var (
		id                  uuid.UUID
		kind                string
		status              string
		dzoID               sql.NullString
		createdByUserID     sql.NullString
		courseID            sql.NullString
		requestedCourseName string
		expenseCategory     sql.NullString
		comment             sql.NullString
		isActive            bool
		createdAt           sql.NullTime
		updatedAt           sql.NullTime
	)
	if err := s.Scan(
		&id,
		&kind,
		&status,
		&dzoID,
		&createdByUserID,
		&courseID,
		&requestedCourseName,
		&expenseCategory,
		&comment,
		&isActive,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.NotFound).Msg("application not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to scan application").Cause(err).Err()
	}

	return &Application{
		ID:                  id.String(),
		Kind:                ApplicationKind(kind),
		Status:              ApplicationStatus(status),
		DzoID:               nullableStringValue(dzoID),
		CreatedByUserID:     nullableStringValue(createdByUserID),
		CourseID:            nullableStringValue(courseID),
		RequestedCourseName: requestedCourseName,
		ExpenseCategory:     nullableStringValue(expenseCategory),
		Comment:             nullableStringValue(comment),
		EmployeeIDs:         []string{},
		IsActive:            isActive,
		CreatedAt:           createdAt.Time,
		UpdatedAt:           updatedAt.Time,
	}, nil
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

func nullableUUID(v *string) interface{} {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	uid, err := uuid.Parse(strings.TrimSpace(*v))
	if err != nil {
		return nil
	}
	return uid
}

func nullableStringValue(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}

func pointerOrNil(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func shouldClearOptionalString(v *string) bool {
	return v != nil && strings.TrimSpace(*v) == ""
}
