package employees

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"encore.app/auth/authhandler"
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

// CreateEmployee creates a new employee.
//
//encore:api auth method=POST path=/employees/create
func CreateEmployee(ctx context.Context, req *CreateEmployeeRequest) (*GetEmployeeResponse, error) {
	if strings.TrimSpace(req.FullName) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("full_name is required").Err()
	}
	if strings.TrimSpace(req.DzoID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id is required").Err()
	}
	if err := validateEmail(req.Email); err != nil {
		return nil, err
	}

	dzoUID, err := uuid.Parse(req.DzoID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}

	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
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

	if err := checkEmailUnique(ctx, req.Email, nil); err != nil {
		return nil, err
	}

	clientUID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid company_id in token").Err()
	}
	emp, err := insertEmployee(ctx, clientUID, dzoUID, req)
	if err != nil {
		return nil, err
	}

	return &GetEmployeeResponse{Employee: *emp}, nil
}

// ListEmployees returns all active employees, with an optional search filter.
// SA sees all; ADM is scoped to their client; HR is scoped to their DZO.
//
//encore:api auth method=GET path=/employees
func ListEmployees(ctx context.Context, params *ListEmployeesParams) (*ListEmployeesResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	dzoFilter := strings.TrimSpace(params.DzoID)
	clientFilter := ""

	switch ad.Role {
	case authhandler.RoleHR:
		// HR can only see employees in their assigned DZO.
		dzoFilter = ad.DzoID
		clientFilter = ""
	case authhandler.RoleADM:
		// ADM can only see employees within their client; optional DZO sub-filter is allowed.
		clientFilter = ad.CompanyID
	}

	emps, err := queryActiveEmployees(ctx, strings.TrimSpace(params.Search), dzoFilter, clientFilter)
	if err != nil {
		return nil, err
	}

	return &ListEmployeesResponse{
		Employees: emps,
		Total:     len(emps),
	}, nil
}

// GetEmployee returns a single employee by ID.
// ADM is scoped to their client; HR is scoped to their DZO.
//
//encore:api auth method=GET path=/employees/:id
func GetEmployee(ctx context.Context, id string) (*GetEmployeeResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	emp, err := queryEmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if ad.Role == authhandler.RoleHR && emp.DzoID != ad.DzoID {
		return nil, errs.B().Code(errs.NotFound).Msg("employee not found").Err()
	}
	if ad.Role == authhandler.RoleADM && emp.ClientID != ad.CompanyID {
		return nil, errs.B().Code(errs.NotFound).Msg("employee not found").Err()
	}

	return &GetEmployeeResponse{Employee: *emp}, nil
}

// PatchEmployee partially updates an employee. ADM is scoped to their client.
//
//encore:api auth method=PATCH path=/employees/:id
func PatchEmployee(ctx context.Context, id string, req *UpdateEmployeeRequest) (*GetEmployeeResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	emp, err := queryEmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// ADM can only update employees within their own client.
	if ad.Role == authhandler.RoleADM && emp.ClientID != ad.CompanyID {
		return nil, errs.B().Code(errs.NotFound).Msg("employee not found").Err()
	}

	if req.Email != nil {
		if err := validateEmail(*req.Email); err != nil {
			return nil, err
		}
	}
	if req.DzoID != nil {
		dzoUID, err := uuid.Parse(*req.DzoID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
		if err := checkDzoExists(ctx, dzoUID); err != nil {
			return nil, err
		}
	}

	emp, err = patchEmployee(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetEmployeeResponse{Employee: *emp}, nil
}

// DeleteEmployee soft-deletes an employee by setting is_deleted=true.
// ADM is scoped to their client.
//
//encore:api auth method=DELETE path=/employees/:id
func DeleteEmployee(ctx context.Context, id string) (*DeleteEmployeeResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	// ADM can only delete employees within their own client.
	if ad.Role == authhandler.RoleADM {
		emp, err := queryEmployeeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if emp.ClientID != ad.CompanyID {
			return nil, errs.B().Code(errs.NotFound).Msg("employee not found").Err()
		}
	}

	if err := softDeleteEmployee(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteEmployeeResponse{Message: "employee deleted successfully"}, nil
}

// UploadEmployees validates uploaded employees .xlsx file before import.
//
//encore:api auth method=POST path=/employees/upload
func UploadEmployees(ctx context.Context, req *UploadEmployeesRequest) (*UploadResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if err := validateUploadRequest(req.FileName, req.FileData); err != nil {
		return nil, err
	}

	parsedRows, previewRows, validationErrors, totalRows, err := parseAndValidateEmployeeExcel(req.FileData)
	if err != nil {
		return nil, err
	}

	// For ADM, DZO validation is restricted to their client's DZOs.
	clientFilter := ""
	if ad.Role == authhandler.RoleADM {
		clientFilter = ad.CompanyID
	}
	parsedRows, previewRows, validationErrors, err = applyUploadBusinessRules(ctx, parsedRows, previewRows, validationErrors, clientFilter)
	if err != nil {
		return nil, err
	}

	validRows := 0
	validRows = len(parsedRows)

	return &UploadResponse{
		IsValid:     len(validationErrors) == 0,
		TotalRows:   totalRows,
		ValidRows:   validRows,
		InvalidRows: totalRows - validRows,
		Errors:      validationErrors,
		Rows:        previewRows,
	}, nil
}

// ImportEmployees imports employees from uploaded .xlsx file.
//
//encore:api auth method=POST path=/employees/import
func ImportEmployees(ctx context.Context, req *ImportEmployeesRequest) (*ImportResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if err := validateUploadRequest(req.FileName, req.FileData); err != nil {
		return nil, err
	}

	if _, err := uuid.Parse(ad.CompanyID); err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid company_id in token").Err()
	}

	rows, previewRows, validationErrors, _, err := parseAndValidateEmployeeExcel(req.FileData)
	if err != nil {
		return nil, err
	}

	// For ADM, DZO validation is restricted to their client's DZOs.
	clientFilter := ""
	if ad.Role == authhandler.RoleADM {
		clientFilter = ad.CompanyID
	}
	rows, _, validationErrors, err = applyUploadBusinessRules(ctx, rows, previewRows, validationErrors, clientFilter)
	if err != nil {
		return nil, err
	}

	rowsToImport := make([]parsedEmployeeRow, 0)
	rowsToImport, err1 := ImportRows(req, rows, rowsToImport, validationErrors)
	if err1 != nil {
		return nil, err1
	}

	if len(rowsToImport) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("no rows selected for import").Err()
	}

	importedCount := 0
	importedCount, result, err2 := ImportCountedRows(ctx, rowsToImport, importedCount)
	if err2 != nil {
		return result, err2
	}

	return &ImportResponse{
		ImportedCount: importedCount,
		Message:       fmt.Sprintf("imported %d employees", importedCount),
	}, nil
}

// ════ INTERNAL ════

func ImportCountedRows(ctx context.Context, rowsToImport []parsedEmployeeRow, importedCount int) (int, *ImportResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return 0, nil, err
	}

	clientUID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return 0, nil, errs.B().Code(errs.InvalidArgument).Msg("invalid company_id in token").Err()
	}

	// For ADM, only DZOs within their client are accessible.
	clientFilter := ""
	if ad.Role == authhandler.RoleADM {
		clientFilter = ad.CompanyID
	}
	dzoMap, err := getDzoNameToIDMap(ctx, clientFilter)
	if err != nil {
		return 0, nil, errs.B().Code(errs.Internal).Msg("failed to load dzo map").Cause(err).Err()
	}

	builders := make([]*ent.EmployeeCreate, 0, len(rowsToImport))
	for _, row := range rowsToImport {
		dzoId, ok := dzoMap[normalizeHeader(row.DzoName)]
		if !ok {
			continue
		}
		builder := Client.Employee.
			Create().
			SetClientID(clientUID).
			SetDzoID(dzoId).
			SetFullName(row.FullName).
			SetEmail(row.Email)

		if row.Position != nil {
			builder = builder.SetPosition(*row.Position)
		}
		if row.ShortName != nil {
			builder = builder.SetShortName(*row.ShortName)
		}
		if row.Department != nil {
			builder = builder.SetDepartment(*row.Department)
		}
		if row.Direction != nil {
			builder = builder.SetDirection(*row.Direction)
		}
		if row.InternalPhone != nil {
			builder = builder.SetInternalPhone(*row.InternalPhone)
		}
		if row.BirthDate != nil {
			builder = builder.SetBirthDate(*row.BirthDate)
		}
		if row.UserID != nil {
			builder = builder.SetUserID(*row.UserID)
		}

		builders = append(builders, builder)
	}

	if len(builders) == 0 {
		return importedCount, nil, nil
	}

	created, err := Client.Employee.CreateBulk(builders...).Save(ctx)
	if err != nil {
		return 0, nil, errs.B().Code(errs.Internal).Msg("failed to import employees").Cause(err).Err()
	}

	importedCount += len(created)
	return importedCount, nil, nil
}

func insertEmployee(ctx context.Context, clientID, dzoID uuid.UUID, req *CreateEmployeeRequest) (*Employee, error) {
	builder := Client.Employee.
		Create().
		SetClientID(clientID).
		SetDzoID(dzoID).
		SetFullName(req.FullName).
		SetEmail(strings.TrimSpace(req.Email))

	if req.Position != nil {
		builder = builder.SetPosition(*req.Position)
	}
	if req.ShortName != nil {
		builder = builder.SetShortName(*req.ShortName)
	}
	if req.Department != nil {
		builder = builder.SetDepartment(*req.Department)
	}
	if req.Direction != nil {
		builder = builder.SetDirection(*req.Direction)
	}
	if req.InternalPhone != nil {
		builder = builder.SetInternalPhone(*req.InternalPhone)
	}
	if req.BirthDate != nil {
		t, err := time.Parse("2006-01-02", *req.BirthDate)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid birth_date format, expected YYYY-MM-DD").Err()
		}
		builder = builder.SetBirthDate(t)
	}
	if req.UserID != nil {
		uid, err := uuid.Parse(*req.UserID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid user_id format").Err()
		}
		builder = builder.SetUserID(uid)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create employee").Cause(err).Err()
	}

	return entToEmployee(row), nil
}

func queryActiveEmployees(ctx context.Context, search string, dzoID string, clientID string) ([]Employee, error) {
	q := Client.Employee.
		Query().
		WithDzo().
		Where(employee.IsDeletedEQ(false))

	if search != "" {
		q = q.Where(employee.Or(
			employee.FullNameContainsFold(search),
			employee.EmailContainsFold(search),
			employee.DepartmentContainsFold(search),
		))
	}

	if dzoID != "" {
		uid, err := uuid.Parse(dzoID)
		if err == nil {
			q = q.Where(employee.DzoIDEQ(uid))
		}
	}

	if clientID != "" {
		uid, err := uuid.Parse(clientID)
		if err == nil {
			q = q.Where(employee.ClientIDEQ(uid))
		}
	}

	rows, err := q.Order(ent.Asc(employee.FieldFullName)).All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list employees").Cause(err).Err()
	}

	emps := make([]Employee, 0, len(rows))
	for _, row := range rows {
		emps = append(emps, *entToEmployee(row))
	}

	return emps, nil
}

func queryEmployeeByID(ctx context.Context, id string) (*Employee, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row, err := Client.Employee.
		Query().
		Where(employee.IDEQ(uid), employee.IsDeletedEQ(false)).
		WithDzo().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("employee not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get employee").Cause(err).Err()
	}

	return entToEmployee(row), nil
}

func patchEmployee(ctx context.Context, id string, req *UpdateEmployeeRequest) (*Employee, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	builder := Client.Employee.UpdateOneID(uid)

	if req.FullName != nil {
		builder = builder.SetFullName(*req.FullName)
	}
	if req.Email != nil {
		trimmedEmail := strings.TrimSpace(*req.Email)
		empUID := uid
		if err := checkEmailUnique(ctx, trimmedEmail, &empUID); err != nil {
			return nil, err
		}
		builder = builder.SetEmail(trimmedEmail)
	}
	if req.DzoID != nil {
		dzoUID, _ := uuid.Parse(*req.DzoID)
		builder = builder.SetDzoID(dzoUID)
	}
	if req.Position != nil {
		builder = builder.SetPosition(*req.Position)
	}
	if req.ShortName != nil {
		builder = builder.SetShortName(*req.ShortName)
	}
	if req.Department != nil {
		builder = builder.SetDepartment(*req.Department)
	}
	if req.Direction != nil {
		builder = builder.SetDirection(*req.Direction)
	}
	if req.InternalPhone != nil {
		builder = builder.SetInternalPhone(*req.InternalPhone)
	}
	if req.BirthDate != nil {
		t, err := time.Parse("2006-01-02", *req.BirthDate)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid birth_date format, expected YYYY-MM-DD").Err()
		}
		builder = builder.SetBirthDate(t)
	}
	if req.IsActive != nil {
		builder = builder.SetIsActive(*req.IsActive)
	}
	if req.UserID != nil {
		uid2, err := uuid.Parse(*req.UserID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid user_id format").Err()
		}
		builder = builder.SetUserID(uid2)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("employee not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update employee").Cause(err).Err()
	}

	return entToEmployee(row), nil
}

func softDeleteEmployee(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	exists, err := Client.Employee.
		Query().
		Where(
			employee.IDEQ(uid),
			employee.IsDeletedEQ(false),
		).
		Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete employee").Cause(err).Err()
	}
	if !exists {
		return errs.B().Code(errs.NotFound).Msg("employee not found").Err()
	}

	err = Client.Employee.
		UpdateOneID(uid).
		SetIsDeleted(true).
		Exec(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete employee").Cause(err).Err()
	}

	return nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("email is required").Err()
	}
	if strings.Count(email, "@") != 1 {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid email format").Err()
	}
	parts := strings.Split(email, "@")
	if len(parts[0]) == 0 || len(parts[1]) == 0 || !strings.Contains(parts[1], ".") {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid email format").Err()
	}
	return nil
}

func checkEmailUnique(ctx context.Context, email string, excludeID *uuid.UUID) error {
	q := Client.Employee.
		Query().
		Where(
			employee.EmailEQ(email),
			employee.IsDeletedEQ(false),
		)
	if excludeID != nil {
		q = q.Where(employee.IDNEQ(*excludeID))
	}

	exists, err := q.Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to check email uniqueness").Cause(err).Err()
	}
	if exists {
		return errs.B().Code(errs.AlreadyExists).Msg("employee with this email already exists").Err()
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
func getDzoNameToIDMap(ctx context.Context, clientID string) (map[string]uuid.UUID, error) {
	q := Client.DzoOrganization.Query().Where(dzoorganization.IsActiveEQ(true))
	if clientID != "" {
		uid, err := uuid.Parse(clientID)
		if err == nil {
			q = q.Where(dzoorganization.ClientIDEQ(uid))
		}
	}
	dzos, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	dzoMap := make(map[string]uuid.UUID, len(dzos))
	for _, dzo := range dzos {
		name := normalizeHeader(dzo.Name)
		dzoMap[name] = dzo.ID
	}

	return dzoMap, nil
}

func getExistingDzoNames(ctx context.Context, clientID string) (map[string]bool, error) {
	q := Client.DzoOrganization.Query().Where(dzoorganization.IsActiveEQ(true))
	if clientID != "" {
		uid, err := uuid.Parse(clientID)
		if err == nil {
			q = q.Where(dzoorganization.ClientIDEQ(uid))
		}
	}
	names, err := q.Select(dzoorganization.FieldName).Strings(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[normalizeHeader(name)] = true
	}

	return result, nil
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

func validateUploadRequest(fileName string, fileData []byte) error {
	if strings.TrimSpace(fileName) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("file_name is required").Err()
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		return errs.B().Code(errs.InvalidArgument).Msg("only .xlsx files are supported").Err()
	}
	if len(fileData) == 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("file_data is empty").Err()
	}
	return nil
}

func parseAndValidateEmployeeExcel(fileData []byte) ([]parsedEmployeeRow, []UploadEmployeeRow, []string, int, error) {
	f, err := excelize.OpenReader(bytes.NewReader(fileData))
	if err != nil {
		return nil, nil, nil, 0, errs.B().Code(errs.InvalidArgument).Msg("invalid xlsx file").Cause(err).Err()
	}
	defer func() { _ = f.Close() }()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, nil, nil, 0, errs.B().Code(errs.InvalidArgument).Msg("xlsx file has no sheets").Err()
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil, nil, 0, errs.B().Code(errs.InvalidArgument).Msg("failed to read xlsx rows").Cause(err).Err()
	}
	if len(rows) == 0 {
		return nil, nil, nil, 0, errs.B().Code(errs.InvalidArgument).Msg("xlsx file is empty").Err()
	}

	headerIndex, err := buildEmployeeHeaderIndex(rows[0])
	if err != nil {
		return nil, nil, nil, 0, err
	}
	parsedRows := make([]parsedEmployeeRow, 0)
	previewRows := make([]UploadEmployeeRow, 0)
	validationErrors := make([]string, 0)
	totalRows := 0
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if isRowEmpty(row) {
			continue
		}
		totalRows++

		parsed, previewRow, rowErrors := parseEmployeeRow(i+1, row, headerIndex)
		previewRows = append(previewRows, previewRow)
		if len(rowErrors) > 0 {
			validationErrors = append(validationErrors, rowErrors...)
			continue
		}

		parsedRows = append(parsedRows, parsed)
	}

	if len(parsedRows) == 0 && len(validationErrors) == 0 {
		validationErrors = append(validationErrors, "file has no data rows")
	}

	return parsedRows, previewRows, validationErrors, totalRows, nil
}

func applyUploadBusinessRules(ctx context.Context, parsedRows []parsedEmployeeRow, previewRows []UploadEmployeeRow, validationErrors []string, clientID string) ([]parsedEmployeeRow, []UploadEmployeeRow, []string, error) {
	rowIndex := make(map[int]int, len(previewRows))
	for i := range previewRows {
		rowIndex[previewRows[i].RowNumber] = i
	}

	invalidRows := make(map[int]struct{})

	appendRowError := func(rowNumber int, message string) {
		validationErrors = append(validationErrors, message)
		idx, ok := rowIndex[rowNumber]
		if !ok {
			return
		}
		previewRows[idx].IsValid = false
		previewRows[idx].Include = false
		previewRows[idx].Errors = append(previewRows[idx].Errors, message)
		invalidRows[rowNumber] = struct{}{}
	}

	employees, err := queryActiveEmployees(ctx, "", "", "")
	if err != nil {
		return nil, nil, nil, err
	}

	existingEmails := make(map[string]bool)
	for _, e := range employees {
		existingEmails[e.Email] = true
	}

	// Duplicate emails in one uploaded file are invalid for all duplicate rows.
	emailRows := make(map[string][]int)
	for _, row := range previewRows {
		email := strings.ToLower(strings.TrimSpace(row.Email))
		if email == "" {
			continue
		}
		emailRows[email] = append(emailRows[email], row.RowNumber)
	}
	for email, rowNumbers := range emailRows {
		if existingEmails[email] {
			for _, rowNumber := range rowNumbers {
				appendRowError(rowNumber, fmt.Sprintf("row %d: employee with this email already exists", rowNumber))
			}
			continue
		}
		if len(rowNumbers) < 2 {
			continue
		}
		for _, rowNumber := range rowNumbers {
			appendRowError(rowNumber, fmt.Sprintf("row %d: duplicate email in file", rowNumber))
		}
	}

	dzoNames, err := getExistingDzoNames(ctx, clientID)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, row := range previewRows {
		if row.DzoName == "" {
			continue
		}
		normalized := normalizeHeader(row.DzoName)
		if !dzoNames[normalized] {
			appendRowError(row.RowNumber, fmt.Sprintf("row %d: dzo not found", row.RowNumber))
		}
	}

	filteredRows := make([]parsedEmployeeRow, 0, len(parsedRows))
	for _, row := range parsedRows {
		if _, invalid := invalidRows[row.RowNumber]; invalid {
			continue
		}
		filteredRows = append(filteredRows, row)
	}

	return filteredRows, previewRows, validationErrors, nil
}

func buildEmployeeHeaderIndex(headerRow []string) (map[string]int, error) {
	headerIndex := make(map[string]int)
	for i, cell := range headerRow {
		normalized := normalizeHeader(cell)
		if normalized == "" {
			continue
		}
		internalName, ok := employeeExcelHeaderAliases[normalized]
		if !ok {
			continue
		}
		if _, exists := headerIndex[internalName]; !exists {
			headerIndex[internalName] = i
		}
	}

	missing := make([]string, 0)
	for _, required := range employeeExcelRequiredHeaders {
		if _, ok := headerIndex[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg(fmt.Sprintf("missing required columns: %s", strings.Join(missing, ", "))).Err()
	}

	return headerIndex, nil
}

func parseEmployeeRow(rowNumber int, row []string, headerIndex map[string]int) (parsedEmployeeRow, UploadEmployeeRow, []string) {
	validationErrors := make([]string, 0)

	get := func(header string) string {
		idx, ok := headerIndex[header]
		if !ok {
			return ""
		}
		if idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	dzoName := get("dzo_name")
	fullName := get("full_name")
	emailRaw := get("email")
	position := strPtr(get("position"))
	shortName := strPtr(get("short_name"))
	department := strPtr(get("department"))
	direction := strPtr(get("direction"))
	internalPhone := strPtr(get("internal_phone"))
	birthDateRaw := strPtr(get("birth_date"))
	userIDRaw := strPtr(get("user_id"))

	previewRow := UploadEmployeeRow{
		RowNumber:     rowNumber,
		DzoName:       dzoName,
		FullName:      fullName,
		Email:         strings.TrimSpace(emailRaw),
		Position:      position,
		ShortName:     shortName,
		Department:    department,
		Direction:     direction,
		InternalPhone: internalPhone,
		BirthDate:     birthDateRaw,
		UserID:        userIDRaw,
	}

	if dzoName == "" {
		validationErrors = append(validationErrors, fmt.Sprintf("row %d: dzo_name is required", rowNumber))
	}
	if fullName == "" {
		validationErrors = append(validationErrors, fmt.Sprintf("row %d: full_name is required", rowNumber))
	}
	if emailRaw == "" {
		validationErrors = append(validationErrors, fmt.Sprintf("row %d: email is required", rowNumber))
	}

	email := strings.TrimSpace(emailRaw)
	if email != "" {
		if err := validateEmail(email); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("row %d: invalid email format", rowNumber))
		}
	}

	var birthDate *time.Time
	birthDateRawStr := get("birth_date")
	if birthDateRawStr != "" {
		parsedBirthDate, err := time.Parse("2006-01-02", birthDateRawStr)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("row %d: invalid birth_date format, expected YYYY-MM-DD", rowNumber))
		} else {
			birthDate = &parsedBirthDate
		}
	}

	var userID *uuid.UUID
	userIDRawStr := get("user_id")
	if userIDRawStr != "" {
		parsedUserID, err := uuid.Parse(userIDRawStr)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("row %d: invalid user_id format", rowNumber))
		} else {
			userID = &parsedUserID
		}
	}

	if len(validationErrors) > 0 {
		previewRow.IsValid = false
		previewRow.Include = false
		previewRow.Errors = validationErrors
		return parsedEmployeeRow{}, previewRow, validationErrors
	}
	previewRow.IsValid = true
	previewRow.Include = true
	previewRow.Errors = []string{}

	return parsedEmployeeRow{
		RowNumber:     rowNumber,
		DzoName:       dzoName,
		FullName:      fullName,
		Email:         email,
		Position:      position,
		ShortName:     shortName,
		Department:    department,
		Direction:     direction,
		InternalPhone: internalPhone,
		BirthDate:     birthDate,
		UserID:        userID,
	}, previewRow, nil
}

func normalizeHeader(header string) string {
	h := strings.ToLower(strings.TrimSpace(header))
	h = strings.ReplaceAll(h, " ", "_")
	h = strings.ReplaceAll(h, "-", "_")
	return h
}

func isRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// ════ HELPERS ════

func entToEmployee(e *ent.Employee) *Employee {
	emp := &Employee{
		ID:       e.ID.String(),
		ClientID: e.ClientID.String(),
		DzoID:    e.DzoID.String(),
		FullName: e.FullName,
		Email:    e.Email,
		IsActive: e.IsActive,
	}
	// Populate DzoName from eager-loaded edge (available when WithDzo() is used)
	if e.Edges.Dzo != nil {
		emp.DzoName = e.Edges.Dzo.Name
	}
	if e.Position != nil {
		emp.Position = e.Position
	}
	if e.ShortName != nil {
		emp.ShortName = e.ShortName
	}
	if e.Department != nil {
		emp.Department = e.Department
	}
	if e.Direction != nil {
		emp.Direction = e.Direction
	}
	if e.InternalPhone != nil {
		emp.InternalPhone = e.InternalPhone
	}
	if e.BirthDate != nil {
		s := e.BirthDate.Format("2006-01-02")
		emp.BirthDate = &s
	}
	if e.UserID != nil {
		s := e.UserID.String()
		emp.UserID = &s
	}
	return emp
}

func ImportRows(req *ImportEmployeesRequest, rows []parsedEmployeeRow, rowsToImport []parsedEmployeeRow, validationErrors []string) ([]parsedEmployeeRow, error) {
	if len(req.SelectedRows) > 0 {
		selectedMap := make(map[int]struct{})
		rowByNumber := make(map[int]parsedEmployeeRow, len(rows))
		for _, row := range rows {
			rowByNumber[row.RowNumber] = row
		}

		for _, selectedRow := range req.SelectedRows {
			if selectedRow < 1 {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("selected_rows contains invalid row number").Err()
			}
			if _, seen := selectedMap[selectedRow]; seen {
				continue
			}
			selectedMap[selectedRow] = struct{}{}

			row, ok := rowByNumber[selectedRow]
			if !ok {
				return nil, errs.B().Code(errs.InvalidArgument).Msg(fmt.Sprintf("row %d is invalid or does not exist", selectedRow)).Err()
			}
			rowsToImport = append(rowsToImport, row)
		}
	} else {
		if len(validationErrors) > 0 {
			return nil, errs.B().Code(errs.InvalidArgument).Msg(strings.Join(validationErrors, "; ")).Err()
		}
		rowsToImport = rows
	}
	return rowsToImport, nil
}
