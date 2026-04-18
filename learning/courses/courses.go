package courses

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

// CreateCourse creates a new external course.
//
//encore:api auth method=POST path=/courses
func CreateCourse(ctx context.Context, req *CreateCourseRequest) (*GetCourseResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !CanManageCourses(ad.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}
	if req.Format != nil && !req.Format.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid course format").Err()
	}

	row, err := insertCourse(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetCourseResponse{
		Course:  *row,
		Modules: []CourseModule{},
	}, nil
}

// ListCourses returns all active courses.
//
//encore:api auth method=GET path=/courses
func ListCourses(ctx context.Context) (*ListCoursesResponse, error) {
	if _, err := getAuthData(); err != nil {
		return nil, err
	}

	rows, err := queryActiveCourses(ctx)
	if err != nil {
		return nil, err
	}

	return &ListCoursesResponse{
		Courses: rows,
		Total:   len(rows),
	}, nil
}

// GetCourse returns a course and its active modules.
//
//encore:api auth method=GET path=/courses/:id
func GetCourse(ctx context.Context, id string) (*GetCourseResponse, error) {
	if _, err := getAuthData(); err != nil {
		return nil, err
	}

	courseRow, err := queryCourseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	modules, err := queryActiveModulesByCourse(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetCourseResponse{
		Course:  *courseRow,
		Modules: modules,
	}, nil
}

// UpdateCourse partially updates a course.
//
//encore:api auth method=PUT path=/courses/:id
func UpdateCourse(ctx context.Context, id string, req *UpdateCourseRequest) (*GetCourseResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !CanManageCourses(ad.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title cannot be empty").Err()
	}
	if req.Format != nil && !req.Format.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid course format").Err()
	}

	courseRow, err := updateCourse(ctx, id, req)
	if err != nil {
		return nil, err
	}
	modules, err := queryActiveModulesByCourse(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetCourseResponse{
		Course:  *courseRow,
		Modules: modules,
	}, nil
}

// DeleteCourse soft-deletes a course.
//
//encore:api auth method=DELETE path=/courses/:id
func DeleteCourse(ctx context.Context, id string) (*DeleteCourseResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !CanManageCourses(ad.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	if err := softDeleteCourse(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteCourseResponse{Message: "course deleted successfully"}, nil
}

// CreateCourseModule creates a new module inside a course.
//
//encore:api auth method=POST path=/course-modules
func CreateCourseModule(ctx context.Context, req *CreateCourseModuleRequest) (*GetCourseModuleResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !CanManageCourses(ad.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	if strings.TrimSpace(req.CourseID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("course_id is required").Err()
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}

	row, err := insertCourseModule(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetCourseModuleResponse{Module: *row}, nil
}

// ListCourseModules returns active modules for a course.
//
//encore:api auth method=GET path=/courses/:course_id/modules
func ListCourseModules(ctx context.Context, course_id string) (*ListCourseModulesResponse, error) {
	if _, err := getAuthData(); err != nil {
		return nil, err
	}

	rows, err := queryActiveModulesByCourse(ctx, course_id)
	if err != nil {
		return nil, err
	}

	return &ListCourseModulesResponse{
		Modules: rows,
		Total:   len(rows),
	}, nil
}

// GetCourseModule returns a single course module.
//
//encore:api auth method=GET path=/course-modules/:id
func GetCourseModule(ctx context.Context, id string) (*GetCourseModuleResponse, error) {
	if _, err := getAuthData(); err != nil {
		return nil, err
	}

	row, err := queryCourseModuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetCourseModuleResponse{Module: *row}, nil
}

// UpdateCourseModule partially updates a course module.
//
//encore:api auth method=PUT path=/course-modules/:id
func UpdateCourseModule(ctx context.Context, id string, req *UpdateCourseModuleRequest) (*GetCourseModuleResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !CanManageCourses(ad.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title cannot be empty").Err()
	}

	row, err := updateCourseModule(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetCourseModuleResponse{Module: *row}, nil
}

// DeleteCourseModule soft-deletes a course module.
//
//encore:api auth method=DELETE path=/course-modules/:id
func DeleteCourseModule(ctx context.Context, id string) (*DeleteCourseModuleResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if !CanManageCourses(ad.Role) {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
	}

	if err := softDeleteCourseModule(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteCourseModuleResponse{Message: "course module deleted successfully"}, nil
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

func CanManageCourses(role authhandler.UserRole) bool {
	return role == authhandler.RoleSA || role == authhandler.RoleADM || role == authhandler.RoleHR
}

func insertCourse(ctx context.Context, req *CreateCourseRequest) (*Course, error) {
	isExternal := true
	if req.IsExternal != nil {
		isExternal = *req.IsExternal
	}

	row := db.QueryRow(ctx, `
		INSERT INTO courses (id, title, description, format, category, is_external, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW(), NOW())
		RETURNING id, title, description, format, category, is_external, is_active, created_at, updated_at
	`, uuid.New(), strings.TrimSpace(req.Title), nullableString(req.Description), nullableCourseFormat(req.Format), nullableString(req.Category), isExternal)

	course, err := scanCourse(row)
	if err != nil {
		return nil, err
	}
	return course, nil
}

func queryActiveCourses(ctx context.Context) ([]Course, error) {
	rows, err := db.Query(ctx, `
		SELECT id, title, description, format, category, is_external, is_active, created_at, updated_at
		FROM courses
		WHERE is_active = TRUE
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list courses").Cause(err).Err()
	}
	defer rows.Close()

	courses := []Course{}
	for rows.Next() {
		row, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, *row)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list courses").Cause(err).Err()
	}

	return courses, nil
}

func queryCourseByID(ctx context.Context, id string) (*Course, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row := db.QueryRow(ctx, `
		SELECT id, title, description, format, category, is_external, is_active, created_at, updated_at
		FROM courses
		WHERE id = $1
	`, uid)

	course, err := scanCourse(row)
	if err != nil {
		return nil, err
	}
	return course, nil
}

func updateCourse(ctx context.Context, id string, req *UpdateCourseRequest) (*Course, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row := db.QueryRow(ctx, `
		UPDATE courses
		SET
			title = COALESCE($2, title),
			description = CASE
				WHEN $3 THEN NULL
				ELSE COALESCE($4, description)
			END,
			format = CASE
				WHEN $5 THEN NULL
				ELSE COALESCE($6, format)
			END,
			category = CASE
				WHEN $7 THEN NULL
				ELSE COALESCE($8, category)
			END,
			is_external = COALESCE($9, is_external),
			is_active = COALESCE($10, is_active),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, description, format, category, is_external, is_active, created_at, updated_at
	`,
		uid,
		nullableTrimmedString(req.Title),
		shouldClearOptionalString(req.Description),
		nullableTrimmedString(req.Description),
		shouldClearCourseFormat(req.Format),
		nullableCourseFormat(req.Format),
		shouldClearOptionalString(req.Category),
		nullableTrimmedString(req.Category),
		req.IsExternal,
		req.IsActive,
	)

	course, err := scanCourse(row)
	if err != nil {
		return nil, err
	}
	return course, nil
}

func softDeleteCourse(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	result, err := db.Exec(ctx, `
		UPDATE courses
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`, uid)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete course").Cause(err).Err()
	}
	if result.RowsAffected() == 0 {
		return errs.B().Code(errs.NotFound).Msg("course not found").Err()
	}

	return nil
}

func insertCourseModule(ctx context.Context, req *CreateCourseModuleRequest) (*CourseModule, error) {
	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid course_id format").Err()
	}
	if err := ensureActiveCourseExists(ctx, courseID); err != nil {
		return nil, err
	}

	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	row := db.QueryRow(ctx, `
		INSERT INTO course_modules (id, course_id, title, description, sort_order, duration_minutes, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW(), NOW())
		RETURNING id, course_id, title, description, sort_order, duration_minutes, is_active, created_at, updated_at
	`, uuid.New(), courseID, strings.TrimSpace(req.Title), nullableString(req.Description), sortOrder, nullableInt(req.DurationMinutes))

	module, err := scanCourseModule(row)
	if err != nil {
		return nil, err
	}
	return module, nil
}

func queryActiveModulesByCourse(ctx context.Context, courseID string) ([]CourseModule, error) {
	uid, err := uuid.Parse(courseID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid course_id format").Err()
	}

	rows, err := db.Query(ctx, `
		SELECT id, course_id, title, description, sort_order, duration_minutes, is_active, created_at, updated_at
		FROM course_modules
		WHERE course_id = $1 AND is_active = TRUE
		ORDER BY sort_order ASC, created_at ASC
	`, uid)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list course modules").Cause(err).Err()
	}
	defer rows.Close()

	modules := []CourseModule{}
	for rows.Next() {
		row, err := scanCourseModule(rows)
		if err != nil {
			return nil, err
		}
		modules = append(modules, *row)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list course modules").Cause(err).Err()
	}

	return modules, nil
}

func queryCourseModuleByID(ctx context.Context, id string) (*CourseModule, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row := db.QueryRow(ctx, `
		SELECT id, course_id, title, description, sort_order, duration_minutes, is_active, created_at, updated_at
		FROM course_modules
		WHERE id = $1
	`, uid)

	module, err := scanCourseModule(row)
	if err != nil {
		return nil, err
	}
	return module, nil
}

func updateCourseModule(ctx context.Context, id string, req *UpdateCourseModuleRequest) (*CourseModule, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row := db.QueryRow(ctx, `
		UPDATE course_modules
		SET
			title = COALESCE($2, title),
			description = CASE
				WHEN $3 THEN NULL
				ELSE COALESCE($4, description)
			END,
			sort_order = COALESCE($5, sort_order),
			duration_minutes = CASE
				WHEN $6 THEN NULL
				ELSE COALESCE($7, duration_minutes)
			END,
			is_active = COALESCE($8, is_active),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, course_id, title, description, sort_order, duration_minutes, is_active, created_at, updated_at
	`,
		uid,
		nullableTrimmedString(req.Title),
		shouldClearOptionalString(req.Description),
		nullableTrimmedString(req.Description),
		req.SortOrder,
		shouldClearOptionalInt(req.DurationMinutes),
		req.DurationMinutes,
		req.IsActive,
	)

	module, err := scanCourseModule(row)
	if err != nil {
		return nil, err
	}
	return module, nil
}

func softDeleteCourseModule(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	result, err := db.Exec(ctx, `
		UPDATE course_modules
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`, uid)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete course module").Cause(err).Err()
	}
	if result.RowsAffected() == 0 {
		return errs.B().Code(errs.NotFound).Msg("course module not found").Err()
	}

	return nil
}

func ensureActiveCourseExists(ctx context.Context, id uuid.UUID) error {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM courses
			WHERE id = $1 AND is_active = TRUE
		)
	`, id).Scan(&exists)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to validate course").Cause(err).Err()
	}
	if !exists {
		return errs.B().Code(errs.NotFound).Msg("course not found").Err()
	}
	return nil
}

func scanCourse(s rowScanner) (*Course, error) {
	var (
		id          uuid.UUID
		title       string
		description sql.NullString
		format      sql.NullString
		category    sql.NullString
		isExternal  bool
		isActive    bool
		createdAt   sql.NullTime
		updatedAt   sql.NullTime
	)
	if err := s.Scan(&id, &title, &description, &format, &category, &isExternal, &isActive, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.NotFound).Msg("course not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to scan course").Cause(err).Err()
	}

	return &Course{
		ID:          id.String(),
		Title:       title,
		Description: nullableStringValue(description),
		Format:      nullableCourseFormatValue(format),
		Category:    nullableStringValue(category),
		IsExternal:  isExternal,
		IsActive:    isActive,
		CreatedAt:   createdAt.Time,
		UpdatedAt:   updatedAt.Time,
	}, nil
}

func scanCourseModule(s rowScanner) (*CourseModule, error) {
	var (
		id              uuid.UUID
		courseID        uuid.UUID
		title           string
		description     sql.NullString
		sortOrder       int
		durationMinutes sql.NullInt64
		isActive        bool
		createdAt       sql.NullTime
		updatedAt       sql.NullTime
	)
	if err := s.Scan(&id, &courseID, &title, &description, &sortOrder, &durationMinutes, &isActive, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sqldb.ErrNoRows) {
			return nil, errs.B().Code(errs.NotFound).Msg("course module not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to scan course module").Cause(err).Err()
	}

	var duration *int
	if durationMinutes.Valid {
		v := int(durationMinutes.Int64)
		duration = &v
	}

	return &CourseModule{
		ID:              id.String(),
		CourseID:        courseID.String(),
		Title:           title,
		Description:     nullableStringValue(description),
		SortOrder:       sortOrder,
		DurationMinutes: duration,
		IsActive:        isActive,
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
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

func nullableTrimmedString(v *string) interface{} {
	if v == nil {
		return nil
	}
	return strings.TrimSpace(*v)
}

func nullableInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func shouldClearOptionalString(v *string) bool {
	return v != nil && strings.TrimSpace(*v) == ""
}

func shouldClearOptionalInt(v *int) bool {
	return v != nil && *v == 0
}

func nullableCourseFormat(v *CourseFormat) interface{} {
	if v == nil {
		return nil
	}
	if *v == "" {
		return nil
	}
	return string(*v)
}

func shouldClearCourseFormat(v *CourseFormat) bool {
	return v != nil && *v == ""
}

func nullableStringValue(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func nullableCourseFormatValue(v sql.NullString) *CourseFormat {
	if !v.Valid {
		return nil
	}
	format := CourseFormat(v.String)
	return &format
}
