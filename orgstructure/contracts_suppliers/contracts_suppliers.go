package contractssuppliers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"bytes"
	"encoding/csv"
	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/contractsupplier"
	"encore.app/db/ent/supplier"
	csh "encore.app/orgstructure/contracts_suppliers_history"
	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/storage/objects"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"strconv"
)

// ════ DATABASE ════

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()

	// contractFiles stores uploaded contract documents (pdf/png/jpeg).
	// Keyed by "<contract_id>/<file_name>".
	contractFiles = objects.NewBucket("contract-files", objects.BucketConfig{})
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

var requirePermission = authhandler.RequirePermission

// ════ ENDPOINTS ════

// CreateContract creates a new supplier contract.
//
//encore:api auth method=POST path=/suppliers/:supplierID/contracts
func CreateContract(ctx context.Context, supplierID string, req *CreateContractRequest) (*GetContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}

	// TODO: validate supplier exists once suppliers module is merged into dev.

	row, err := insertContract(ctx, supplierID, req)
	if err != nil {
		return nil, err
	}

	newContract := entToContract(row)

	// Audit failure does not fail the request — the contract already exists.
	if auditErr := csh.InsertAuditRecord(ctx, newContract.ID, csh.OpCreate, nil, csh.EntToContract(row)); auditErr != nil {
		rlog.Error("contracts-suppliers: failed to write audit record",
			"contract_id", newContract.ID, "err", auditErr)
	}

	return &GetContractResponse{Contract: *newContract}, nil
}

// ListContracts returns a paginated, filtered list of supplier contracts.
//
//encore:api auth method=GET path=/contracts-suppliers
func ListContracts(ctx context.Context, filter *ListContractsFilter) (*ListContractsResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	page, limit := applyFilterDefaults(filter.Page, filter.Limit)

	rows, total, err := queryContractsFiltered(ctx, filter, page, limit)
	if err != nil {
		return nil, err
	}

	contracts := make([]ContractSupplier, 0, len(rows))
	for _, r := range rows {
		contracts = append(contracts, *entToContract(r))
	}

	return &ListContractsResponse{
		Contracts: contracts,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

// GetContract returns a single supplier contract by ID.
//
//encore:api auth method=GET path=/contracts-suppliers/id/:id
func GetContract(ctx context.Context, id string) (*GetContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	row, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetContractResponse{Contract: *entToContract(row)}, nil
}

// UpdateContract patches a supplier contract.
//
//encore:api auth method=PATCH path=/contracts-suppliers/id/:id
func UpdateContract(ctx context.Context, id string, req *UpdateContractRequest) (*GetContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	if err := validateUpdateRequest(req); err != nil {
		return nil, err
	}

	rowBefore, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rowBefore.IsActive {
		return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
	}

	rowAfter, err := updateContract(ctx, rowBefore, req)
	if err != nil {
		return nil, err
	}

	if auditErr := csh.InsertAuditRecord(ctx, id, csh.OpUpdate,
		csh.EntToContract(rowBefore), csh.EntToContract(rowAfter)); auditErr != nil {
		rlog.Error("contracts-suppliers: failed to write audit record",
			"contract_id", id, "err", auditErr)
	}

	return &GetContractResponse{Contract: *entToContract(rowAfter)}, nil
}

// DeleteContract soft-deletes a supplier contract (sets is_active=false).
//
//encore:api auth method=DELETE path=/contracts-suppliers/id/:id
func DeleteContract(ctx context.Context, id string) (*DeleteContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	rowBefore, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rowBefore.IsActive {
		return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
	}

	rowAfter, err := softDeleteContract(ctx, rowBefore)
	if err != nil {
		return nil, err
	}

	if auditErr := csh.InsertAuditRecord(ctx, id, csh.OpDelete,
		csh.EntToContract(rowBefore), csh.EntToContract(rowAfter)); auditErr != nil {
		rlog.Error("contracts-suppliers: failed to write audit record",
			"contract_id", id, "err", auditErr)
	}

	return &DeleteContractResponse{Message: "contract deleted"}, nil
}

// AddAmendment records an amendment (доп. соглашение) on the contract.
// Only one amendment is allowed per contract; a second attempt returns 409.
// Recomputes total_with_amendment and remaining_amount.
//
//encore:api auth method=POST path=/contracts-suppliers/id/:id/amendment
func AddAmendment(ctx context.Context, id string, req *AmendmentRequest) (*GetContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	if err := validateAmendmentRequest(req); err != nil {
		return nil, err
	}

	rowBefore, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rowBefore.IsActive {
		return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
	}
	if rowBefore.AmendmentNumber != nil {
		return nil, errs.B().Code(errs.AlreadyExists).Msg("amendment already exists for this contract").Err()
	}

	rowAfter, err := applyAmendment(ctx, rowBefore, req)
	if err != nil {
		return nil, err
	}

	if auditErr := csh.InsertAuditRecord(ctx, id, csh.OpUpdate,
		csh.EntToContract(rowBefore), csh.EntToContract(rowAfter)); auditErr != nil {
		rlog.Error("contracts-suppliers: failed to write audit record",
			"contract_id", id, "err", auditErr)
	}

	return &GetContractResponse{Contract: *entToContract(rowAfter)}, nil
}

// UploadFile attaches a contract document (pdf/jpg/png) to the contract.
// Replaces any previously uploaded file. Max size 25 MB.
//
//encore:api auth method=POST path=/contracts-suppliers/id/:id/upload-file
func UploadFile(ctx context.Context, id string, req *UploadFileRequest) (*GetContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	if err := validateUploadFileRequest(req); err != nil {
		return nil, err
	}

	rowBefore, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rowBefore.IsActive {
		return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
	}

	mimeType := http.DetectContentType(req.FileData)
	if !isAllowedMimeType(mimeType) {
		return nil, errs.B().Code(errs.InvalidArgument).
			Msgf("unsupported file type %q; allowed: pdf, png, jpeg", mimeType).Err()
	}

	newKey := buildFileKey(id, req.FileName)
	if err := uploadFileToBucket(ctx, newKey, req.FileData); err != nil {
		return nil, err
	}

	// If replacing a different key, remove the old object (best-effort).
	if rowBefore.FileKey != nil && *rowBefore.FileKey != newKey {
		if rmErr := contractFiles.Remove(ctx, *rowBefore.FileKey); rmErr != nil {
			rlog.Error("contracts-suppliers: failed to remove old file",
				"contract_id", id, "key", *rowBefore.FileKey, "err", rmErr)
		}
	}

	rowAfter, err := updateContractFileFields(ctx, rowBefore, newKey, req.FileName, int64(len(req.FileData)), mimeType)
	if err != nil {
		// Roll back the bucket upload to avoid orphaned objects.
		if rmErr := contractFiles.Remove(ctx, newKey); rmErr != nil {
			rlog.Error("contracts-suppliers: failed to remove orphaned file",
				"contract_id", id, "key", newKey, "err", rmErr)
		}
		return nil, err
	}

	if auditErr := csh.InsertAuditRecord(ctx, id, csh.OpUpdate,
		csh.EntToContract(rowBefore), csh.EntToContract(rowAfter)); auditErr != nil {
		rlog.Error("contracts-suppliers: failed to write audit record",
			"contract_id", id, "err", auditErr)
	}

	return &GetContractResponse{Contract: *entToContract(rowAfter)}, nil
}

// GetFileURL returns a short-lived signed URL to download the contract's file.
//
//encore:api auth method=GET path=/contracts-suppliers/id/:id/file-url
func GetFileURL(ctx context.Context, id string) (*FileURLResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}

	row, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !row.IsActive {
		return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
	}
	if row.FileKey == nil {
		return nil, errs.B().Code(errs.NotFound).Msg("no file uploaded for this contract").Err()
	}

	signed, err := contractFiles.SignedDownloadURL(ctx, *row.FileKey, objects.WithTTL(signedURLTTL))
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to generate signed url").Cause(err).Err()
	}

	resp := &FileURLResponse{
		URL:       signed.URL,
		ExpiresAt: time.Now().Add(signedURLTTL),
	}
	if row.FileName != nil {
		resp.FileName = *row.FileName
	}
	if row.FileMimeType != nil {
		resp.MimeType = *row.FileMimeType
	}
	return resp, nil
}

// ImportContracts imports selected rows from file.
//
//encore:api auth method=POST path=/contracts-suppliers/import
func ImportContracts(ctx context.Context, req *ImportContractsRequest) (*ImportResponse, error) {
	ad, err := requirePermission()
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	clientID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid company_id in token").Err()
	}

	if err := validateUploadFileRequest(&UploadFileRequest{
		FileName: req.FileName,
		FileData: req.FileData,
	}); err != nil {
		return nil, err
	}

	rows, previewRows, _, _, err := parseAndValidateContractFile(req.FileData, req.FileName)
	if err != nil {
		return nil, err
	}
	rows, previewRows, _, err = applyContractBusinessRules(ctx, clientID, rows, previewRows, []string{})
	if err != nil {
		return nil, err
	}

	selectedMap := make(map[int]struct{}, len(req.SelectedRows))
	for _, rowNumber := range req.SelectedRows {
		selectedMap[rowNumber] = struct{}{}
	}

	parsedByRow := make(map[int]parsedContractRow, len(rows))
	for _, row := range rows {
		parsedByRow[row.RowNumber] = row
	}

	imported := 0
	failed := 0
	errorsList := []string{}

	for _, previewRow := range previewRows {
		// если есть выбор строк — фильтруем
		if len(selectedMap) > 0 {
			if _, ok := selectedMap[previewRow.RowNumber]; !ok {
				continue
			}
		}

		if !previewRow.IsValid {
			failed++
			if len(previewRow.Errors) == 0 {
				errorsList = append(errorsList, fmt.Sprintf("row %d: invalid row", previewRow.RowNumber))
			} else {
				for _, rowError := range previewRow.Errors {
					errorsList = append(errorsList, fmt.Sprintf("row %d: %s", previewRow.RowNumber, rowError))
				}
			}
			continue
		}

		row, ok := parsedByRow[previewRow.RowNumber]
		if !ok {
			failed++
			errorsList = append(errorsList, fmt.Sprintf("row %d: row not found in parsed data", previewRow.RowNumber))
			continue
		}

		reqCopy := row.Request
		if err := validateCreateRequest(&reqCopy); err != nil {
			failed++
			errorsList = append(errorsList, fmt.Sprintf("row %d: %s", previewRow.RowNumber, err.Error()))
			continue
		}

		_, err := insertContract(ctx, row.SupplierID, &reqCopy)
		if err != nil {
			failed++
			errorsList = append(errorsList, fmt.Sprintf("row %d: %s", previewRow.RowNumber, err.Error()))
			continue
		}

		imported++
	}

	if len(selectedMap) > 0 && imported == 0 && failed == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("no rows selected for import").Err()
	}

	return &ImportResponse{
		Imported: imported,
		Failed:   failed,
		Errors:   errorsList,
	}, nil
}

// UploadContracts parses and validates file, returns preview.
//
//encore:api auth method=POST path=/contracts-suppliers/upload
func UploadContracts(ctx context.Context, req *UploadContractsRequest) (*UploadContractsResponse, error) {
	ad, err := requirePermission()
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	clientID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid company_id in token").Err()
	}

	if err := validateUploadFileRequest(&UploadFileRequest{
		FileName: req.FileName,
		FileData: req.FileData,
	}); err != nil {
		return nil, err
	}

	rows, previewRows, validationErrors, totalRows, err := parseAndValidateContractFile(req.FileData, req.FileName)
	if err != nil {
		return nil, err
	}
	rows, previewRows, validationErrors, err = applyContractBusinessRules(ctx, clientID, rows, previewRows, validationErrors)
	if err != nil {
		return nil, err
	}

	validCount := 0
	invalidCount := 0

	for _, row := range previewRows {
		if row.IsValid {
			validCount++
		} else {
			invalidCount++
		}
	}

	return &UploadContractsResponse{
		IsValid:     invalidCount == 0,
		TotalRows:   totalRows,
		ValidRows:   validCount,
		InvalidRows: invalidCount,
		Errors:      validationErrors,
		Rows:        previewRows,
	}, nil
}

// ════ INTERNAL ════

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
	defaultContractCurrency = "KZT"

	// expiringSoonWindow is the threshold for marking a contract EXPIRING_SOON.
	expiringSoonWindow = 30 * 24 * time.Hour
)

var contractRequiredHeaders = []string{
	"supplier_id",
	"contract_number",
	"vat_flag",
	"signed_date",
	"amount",
}

var contractHeaderAliases = map[string]string{
	"supplier_id":                   "supplier_id",
	"supplierid":                    "supplier_id",
	"contract_number":               "contract_number",
	"contractnumber":                "contract_number",
	"vat_flag":                      "vat_flag",
	"vat":                           "vat_flag",
	"percent_nds":                   "vat_flag",
	"signed_date":                   "signed_date",
	"signeddate":                    "signed_date",
	"amount":                        "amount",
	"amount_currency":               "amount_currency",
	"amountcurrency":                "amount_currency",
	"currency":                      "currency",
	"balance_at_year_end":           "balance_at_year_end",
	"balanceatyearend":              "balance_at_year_end",
	"end_date":                      "end_date",
	"enddate":                       "end_date",
	"номер_договора":                "contract_number",
	"процент_ндс":                   "vat_flag",
	"дата_договора":                 "signed_date",
	"сумма":                         "amount",
	"сумма_в_иностранной_валюте":    "amount_currency",
	"валюта":                        "currency",
	"остаток_на_конец_года":         "balance_at_year_end",
	"дата_окончания":                "end_date",
}

var allowedContractCurrencies = map[string]struct{}{
	"KZT": {},
	"USD": {},
	"EUR": {},
}

type parsedContractRow struct {
	RowNumber  int
	SupplierID string
	Request    CreateContractRequest
}

// computeStatus derives the lifecycle status from end_date.
// Contracts without end_date are treated as ACTIVE.
func computeStatus(now time.Time, endDate *time.Time) ContractStatus {
	if endDate == nil {
		return StatusActive
	}
	if !now.Before(*endDate) {
		return StatusExpired
	}
	if endDate.Sub(now) <= expiringSoonWindow {
		return StatusExpiringSoon
	}
	return StatusActive
}

// applyFilterDefaults normalizes page and limit: page >= 1, limit in [1, 100].
func applyFilterDefaults(page, limit int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if limit < 1 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func queryContractsFiltered(ctx context.Context, filter *ListContractsFilter, page, limit int) ([]*ent.ContractSupplier, int, error) {
	q := Client.ContractSupplier.Query()

	if !filter.IncludeInactive {
		q = q.Where(contractsupplier.IsActive(true))
	}

	if filter.SupplierID != "" {
		sid, err := uuid.Parse(filter.SupplierID)
		if err != nil {
			return nil, 0, errs.B().Code(errs.InvalidArgument).Msg("invalid supplier_id format").Err()
		}
		q = q.Where(contractsupplier.SupplierIDEQ(sid))
	}

	if search := strings.TrimSpace(filter.Search); search != "" {
		q = q.Where(contractsupplier.ContractNumberContainsFold(search))
	}

	if status := strings.TrimSpace(filter.Status); status != "" {
		s := ContractStatus(status)
		if !s.IsValid() {
			return nil, 0, errs.B().Code(errs.InvalidArgument).Msg("invalid status; allowed: ACTIVE, EXPIRED, EXPIRING_SOON").Err()
		}
		now := time.Now()
		switch s {
		case StatusExpired:
			q = q.Where(contractsupplier.EndDateLTE(now))
		case StatusExpiringSoon:
			q = q.Where(
				contractsupplier.EndDateGT(now),
				contractsupplier.EndDateLTE(now.Add(expiringSoonWindow)),
			)
		case StatusActive:
			q = q.Where(contractsupplier.Or(
				contractsupplier.EndDateIsNil(),
				contractsupplier.EndDateGT(now.Add(expiringSoonWindow)),
			))
		}
	}

	if !filter.ExpiryDateFrom.IsZero() {
		q = q.Where(contractsupplier.EndDateGTE(filter.ExpiryDateFrom))
	}
	if !filter.ExpiryDateTo.IsZero() {
		q = q.Where(contractsupplier.EndDateLTE(filter.ExpiryDateTo))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, errs.B().Code(errs.Internal).Msg("failed to count contracts").Cause(err).Err()
	}

	rows, err := q.
		Order(ent.Desc(contractsupplier.FieldCreatedAt)).
		Limit(limit).
		Offset((page - 1) * limit).
		All(ctx)
	if err != nil {
		return nil, 0, errs.B().Code(errs.Internal).Msg("failed to list contracts").Cause(err).Err()
	}

	return rows, total, nil
}

func validateUpdateRequest(req *UpdateContractRequest) error {
	if req == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if req.ContractNumber == nil &&
		req.VatFlag == nil &&
		req.SignedDate == nil &&
		req.EndDate == nil &&
		req.Amount == nil &&
		req.AmountCurrency == nil &&
		req.Currency == nil &&
		req.BalanceAtYearEnd == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("no fields to update").Err()
	}
	if req.ContractNumber != nil && strings.TrimSpace(*req.ContractNumber) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("contract_number cannot be empty").Err()
	}
	if req.SignedDate != nil && req.SignedDate.IsZero() {
		return errs.B().Code(errs.InvalidArgument).Msg("signed_date cannot be zero").Err()
	}
	if req.EndDate != nil && req.EndDate.IsZero() {
		return errs.B().Code(errs.InvalidArgument).Msg("end_date cannot be zero").Err()
	}
	if req.VatFlag != nil && (*req.VatFlag < 0 || *req.VatFlag > 100) {
		return errs.B().Code(errs.InvalidArgument).Msg("vat_flag must be between 0 and 100").Err()
	}
	if req.Amount != nil && *req.Amount < 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("amount must be >= 0").Err()
	}
	if req.AmountCurrency != nil && *req.AmountCurrency < 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("amount_currency must be >= 0").Err()
	}
	if req.Currency != nil {
		currency := normalizeContractCurrencyValue(req.Currency)
		if !isAllowedContractCurrency(currency) {
			return errs.B().Code(errs.InvalidArgument).Msg("currency must be one of KZT, USD, EUR").Err()
		}
	}

	return nil
}

func updateContract(ctx context.Context, row *ent.ContractSupplier, req *UpdateContractRequest) (*ent.ContractSupplier, error) {
	if req.Amount != nil && row.AmendmentAmount != nil {
		return nil, errs.B().
			Code(errs.FailedPrecondition).
			Msg("cannot update amount after amendment exists; use amendment flow").
			Err()
	}

	upd := Client.ContractSupplier.UpdateOne(row)

	if req.ContractNumber != nil {
		upd.SetContractNumber(strings.TrimSpace(*req.ContractNumber))
	}
	if req.VatFlag != nil {
		upd.SetVatFlag(*req.VatFlag)
	}
	if req.SignedDate != nil {
		upd.SetSignedDate(*req.SignedDate)
	}
	if req.EndDate != nil {
		upd.SetEndDate(*req.EndDate)
	}
	if req.Amount != nil {
		upd.SetAmount(*req.Amount)

		upd.SetTotalWithAmendment(*req.Amount)
		upd.SetRemainingAmount(*req.Amount)
	}
	if req.AmountCurrency != nil {
		upd.SetAmountCurrency(*req.AmountCurrency)
	}
	if req.Currency != nil {
		normalizedCurrency := normalizeContractCurrencyValue(req.Currency)
		upd.SetCurrency(normalizedCurrency)
	}
	if req.BalanceAtYearEnd != nil {
		upd.SetBalanceAtYearEnd(*req.BalanceAtYearEnd)
	}

	updated, err := upd.Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update contract").Cause(err).Err()
	}
	return updated, nil
}

const (
	maxUploadSize = 25 * 1024 * 1024
	signedURLTTL  = 15 * time.Minute
)

var allowedMimeTypes = map[string]struct{}{
	"application/pdf": {},
	"image/png":       {},
	"image/jpeg":      {},
}

func isAllowedMimeType(mime string) bool {
	_, ok := allowedMimeTypes[mime]
	return ok
}

func validateUploadFileRequest(req *UploadFileRequest) error {
	if req == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if strings.TrimSpace(req.FileName) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("file_name is required").Err()
	}
	if len(req.FileData) == 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("file_data is required").Err()
	}
	if len(req.FileData) > maxUploadSize {
		return errs.B().Code(errs.InvalidArgument).Msg("file_data exceeds 25 MB limit").Err()
	}
	return nil
}

// buildFileKey produces a deterministic bucket key: "<contract_id>/<basename>".
// Strips any directory components from the user-supplied file name.
func buildFileKey(contractID, fileName string) string {
	return contractID + "/" + filepath.Base(strings.TrimSpace(fileName))
}

func uploadFileToBucket(ctx context.Context, key string, data []byte) error {
	w := contractFiles.Upload(ctx, key)
	if _, err := w.Write(data); err != nil {
		w.Abort(err)
		return errs.B().Code(errs.Internal).Msg("failed to upload file").Cause(err).Err()
	}
	if err := w.Close(); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to finalize upload").Cause(err).Err()
	}
	return nil
}

func updateContractFileFields(ctx context.Context, row *ent.ContractSupplier, key, name string, size int64, mime string) (*ent.ContractSupplier, error) {
	updated, err := Client.ContractSupplier.UpdateOne(row).
		SetFileKey(key).
		SetFileName(name).
		SetFileSize(size).
		SetFileMimeType(mime).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to save file metadata").Cause(err).Err()
	}
	return updated, nil
}

func validateAmendmentRequest(req *AmendmentRequest) error {
	if req == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if strings.TrimSpace(req.AmendmentNumber) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("amendment_number is required").Err()
	}
	if req.AmendmentDate.IsZero() {
		return errs.B().Code(errs.InvalidArgument).Msg("amendment_date is required").Err()
	}
	// TODO: revisit once business confirms whether negative amendments (scope reduction) are needed.
	if req.AmendmentAmount <= 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("amendment_amount must be > 0").Err()
	}
	return nil
}

func applyAmendment(ctx context.Context, row *ent.ContractSupplier, req *AmendmentRequest) (*ent.ContractSupplier, error) {
	updated, err := Client.ContractSupplier.UpdateOne(row).
		SetAmendmentNumber(strings.TrimSpace(req.AmendmentNumber)).
		SetAmendmentDate(req.AmendmentDate).
		SetAmendmentAmount(req.AmendmentAmount).
		SetTotalWithAmendment(row.Amount + req.AmendmentAmount).
		SetRemainingAmount(row.RemainingAmount + req.AmendmentAmount).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to apply amendment").Cause(err).Err()
	}
	return updated, nil
}

func softDeleteContract(ctx context.Context, row *ent.ContractSupplier) (*ent.ContractSupplier, error) {
	updated, err := Client.ContractSupplier.
		UpdateOne(row).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to delete contract").Cause(err).Err()
	}
	return updated, nil
}

func queryContractByID(ctx context.Context, id string) (*ent.ContractSupplier, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid contract id format").Err()
	}

	row, err := Client.ContractSupplier.
		Query().
		Where(contractsupplier.IDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get contract").Cause(err).Err()
	}
	return row, nil
}

func validateCreateRequest(req *CreateContractRequest) error {
	if req == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if strings.TrimSpace(req.ContractNumber) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("contract_number is required").Err()
	}
	if req.Amount < 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("amount must be >= 0").Err()
	}
	if req.AmountCurrency != nil && *req.AmountCurrency < 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("amount_currency must be >= 0").Err()
	}
	if req.VatFlag < 0 || req.VatFlag > 100 {
		return errs.B().Code(errs.InvalidArgument).Msg("vat_flag must be between 0 and 100").Err()
	}
	if req.SignedDate.IsZero() {
		return errs.B().Code(errs.InvalidArgument).Msg("signed_date is required").Err()
	}
	currency := normalizeContractCurrencyValue(req.Currency)
	if !isAllowedContractCurrency(currency) {
		return errs.B().Code(errs.InvalidArgument).Msg("currency must be one of KZT, USD, EUR").Err()
	}
	if currency != defaultContractCurrency && req.AmountCurrency == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("amount_currency is required for USD/EUR contracts").Err()
	}
	if req.EndDate != nil && !req.EndDate.After(req.SignedDate) {
		return errs.B().Code(errs.InvalidArgument).Msg("end_date must be after signed_date").Err()
	}
	return nil
}

func insertContract(ctx context.Context, supplierID string, req *CreateContractRequest) (*ent.ContractSupplier, error) {
	sid, err := uuid.Parse(supplierID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid supplier_id format").Err()
	}

	normalizedCurrency := normalizeContractCurrencyValue(req.Currency)
	normalizedReq := *req
	normalizedReq.Currency = strPtr(normalizedCurrency)

	row, err := Client.ContractSupplier.Create().
		SetSupplierID(sid).
		SetContractNumber(strings.TrimSpace(normalizedReq.ContractNumber)).
		SetVatFlag(normalizedReq.VatFlag).
		SetSignedDate(normalizedReq.SignedDate).
		SetNillableEndDate(normalizedReq.EndDate).
		SetAmount(normalizedReq.Amount).
		SetNillableAmountCurrency(normalizedReq.AmountCurrency).
		SetNillableCurrency(normalizedReq.Currency).
		SetNillableBalanceAtYearEnd(normalizedReq.BalanceAtYearEnd).
		SetTotalWithAmendment(normalizedReq.Amount).
		SetRemainingAmount(normalizedReq.Amount).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to create contract").Cause(err).Err()
	}
	return row, nil
}

func entToContract(e *ent.ContractSupplier) *ContractSupplier {
	return &ContractSupplier{
		ID:                 e.ID.String(),
		SupplierID:         e.SupplierID.String(),
		ContractNumber:     e.ContractNumber,
		VatFlag:            e.VatFlag,
		SignedDate:         e.SignedDate,
		EndDate:            e.EndDate,
		Status:             computeStatus(time.Now(), e.EndDate),
		Amount:             e.Amount,
		AmountCurrency:     e.AmountCurrency,
		Currency:           e.Currency,
		BalanceAtYearEnd:   e.BalanceAtYearEnd,
		AmendmentNumber:    e.AmendmentNumber,
		AmendmentDate:      e.AmendmentDate,
		AmendmentAmount:    e.AmendmentAmount,
		TotalWithAmendment: e.TotalWithAmendment,
		RemainingAmount:    e.RemainingAmount,
		FileKey:            e.FileKey,
		FileName:           e.FileName,
		FileSize:           e.FileSize,
		FileMimeType:       e.FileMimeType,
		IsActive:           e.IsActive,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

func parseAndValidateContractFile(fileData []byte, fileName string) ([]parsedContractRow, []UploadContractRow, []string, int, error) {
	ext := strings.ToLower(filepath.Ext(fileName))

	var rawRows [][]string
	var err error

	switch ext {
	case ".csv":
		rawRows, err = parseContractCSV(fileData)
	case ".xlsx":
		rawRows, err = parseContractXLSX(fileData)
	default:
		return nil, nil, nil, 0, errs.B().Code(errs.InvalidArgument).Msg("only .csv and .xlsx are supported").Err()
	}
	if err != nil {
		return nil, nil, nil, 0, errs.B().Code(errs.InvalidArgument).Msg("failed to parse file").Cause(err).Err()
	}
	if len(rawRows) < 2 {
		return nil, []UploadContractRow{}, []string{"file is empty or has only headers"}, 0, nil
	}

	headerIndex, globalErr := buildContractHeaderIndex(rawRows[0])
	if globalErr != "" {
		return nil, []UploadContractRow{}, []string{globalErr}, 0, nil
	}

	parsedRows := []parsedContractRow{}
	previewRows := []UploadContractRow{}
	validationErrors := []string{}
	totalRows := 0

	for i, row := range rawRows[1:] {
		if isContractRowEmpty(row) {
			continue
		}

		totalRows++
		rowNumber := i + 1

		parsed, previewRow, rowErrors := parseContractRow(rowNumber, row, headerIndex)
		previewRow.Errors = rowErrors
		previewRow.IsValid = len(rowErrors) == 0
		previewRow.Include = previewRow.IsValid
		previewRows = append(previewRows, previewRow)

		if len(rowErrors) > 0 {
			for _, rowError := range rowErrors {
				validationErrors = append(validationErrors, fmt.Sprintf("row %d: %s", rowNumber, rowError))
			}
			continue
		}

		parsedRows = append(parsedRows, parsed)
	}

	if totalRows == 0 {
		return nil, previewRows, []string{"file has no data rows"}, 0, nil
	}

	return parsedRows, previewRows, validationErrors, totalRows, nil
}

func buildContractHeaderIndex(headers []string) (map[string]int, string) {
	index := map[string]int{}
	for i, header := range headers {
		normalized := normalizeContractHeader(header)
		internalName, ok := contractHeaderAliases[normalized]
		if !ok {
			continue
		}
		if _, exists := index[internalName]; !exists {
			index[internalName] = i
		}
	}

	missing := []string{}
	for _, required := range contractRequiredHeaders {
		if _, ok := index[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Sprintf("missing required columns: %s", strings.Join(missing, ", "))
	}

	return index, ""
}

func parseContractRow(rowNumber int, row []string, headerIndex map[string]int) (parsedContractRow, UploadContractRow, []string) {
	get := func(header string) string {
		idx, ok := headerIndex[header]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	rowErrors := []string{}
	previewRow := UploadContractRow{RowNumber: rowNumber}

	supplierID := get("supplier_id")
	if supplierID == "" {
		rowErrors = append(rowErrors, "supplier_id is required")
	}

	contractNumber := get("contract_number")
	if contractNumber == "" {
		rowErrors = append(rowErrors, "contract_number is required")
	}

	var vatFlag *int
	vatRaw := get("vat_flag")
	if vatRaw == "" {
		rowErrors = append(rowErrors, "vat_flag is required")
	} else {
		parsedVat, err := strconv.Atoi(vatRaw)
		if err != nil {
			rowErrors = append(rowErrors, "vat_flag must be an integer percentage")
		} else {
			vatFlag = &parsedVat
		}
	}

	var signedDate *time.Time
	signedDateRaw := get("signed_date")
	if signedDateRaw == "" {
		rowErrors = append(rowErrors, "signed_date is required")
	} else {
		parsedSignedDate, err := time.Parse("2006-01-02", signedDateRaw)
		if err != nil {
			rowErrors = append(rowErrors, "signed_date must be in YYYY-MM-DD format")
		} else {
			signedDate = &parsedSignedDate
		}
	}

	var endDate *time.Time
	if endDateRaw := get("end_date"); endDateRaw != "" {
		parsedEndDate, err := time.Parse("2006-01-02", endDateRaw)
		if err != nil {
			rowErrors = append(rowErrors, "end_date must be in YYYY-MM-DD format")
		} else {
			endDate = &parsedEndDate
		}
	}

	var amount *float64
	amountRaw := get("amount")
	if amountRaw == "" {
		rowErrors = append(rowErrors, "amount is required")
	} else {
		parsedAmount, err := strconv.ParseFloat(amountRaw, 64)
		if err != nil {
			rowErrors = append(rowErrors, "amount must be a number")
		} else {
			amount = &parsedAmount
		}
	}

	var amountCurrency *float64
	if amountCurrencyRaw := get("amount_currency"); amountCurrencyRaw != "" {
		parsedAmountCurrency, err := strconv.ParseFloat(amountCurrencyRaw, 64)
		if err != nil {
			rowErrors = append(rowErrors, "amount_currency must be a number")
		} else {
			amountCurrency = &parsedAmountCurrency
		}
	}

	currencyValue := normalizeContractCurrencyValue(strPtr(get("currency")))
	currency := strPtr(currencyValue)

	var balanceAtYearEnd *float64
	if balanceRaw := get("balance_at_year_end"); balanceRaw != "" {
		parsedBalance, err := strconv.ParseFloat(balanceRaw, 64)
		if err != nil {
			rowErrors = append(rowErrors, "balance_at_year_end must be a number")
		} else {
			balanceAtYearEnd = &parsedBalance
		}
	}

	previewRow.SupplierID = supplierID
	previewRow.ContractNumber = contractNumber
	previewRow.VatFlag = vatFlag
	previewRow.SignedDate = signedDate
	previewRow.EndDate = endDate
	previewRow.Amount = amount
	previewRow.AmountCurrency = amountCurrency
	previewRow.Currency = currency
	previewRow.BalanceAtYearEnd = balanceAtYearEnd

	if len(rowErrors) > 0 {
		return parsedContractRow{}, previewRow, rowErrors
	}

	req := CreateContractRequest{
		ContractNumber:   contractNumber,
		VatFlag:          *vatFlag,
		SignedDate:       *signedDate,
		EndDate:          endDate,
		Amount:           *amount,
		AmountCurrency:   amountCurrency,
		Currency:         currency,
		BalanceAtYearEnd: balanceAtYearEnd,
	}
	if err := validateCreateRequest(&req); err != nil {
		return parsedContractRow{}, previewRow, []string{err.Error()}
	}

	return parsedContractRow{
		RowNumber:  rowNumber,
		SupplierID: supplierID,
		Request:    req,
	}, previewRow, nil
}

func applyContractBusinessRules(ctx context.Context, clientID uuid.UUID, parsedRows []parsedContractRow, previewRows []UploadContractRow, validationErrors []string) ([]parsedContractRow, []UploadContractRow, []string, error) {
	rowIndex := make(map[int]int, len(previewRows))
	for i := range previewRows {
		rowIndex[previewRows[i].RowNumber] = i
	}

	appendRowErrors := func(rowNumber int, rowErrors ...string) {
		if len(rowErrors) == 0 {
			return
		}
		for _, rowError := range rowErrors {
			validationErrors = append(validationErrors, fmt.Sprintf("row %d: %s", rowNumber, rowError))
		}
		idx, ok := rowIndex[rowNumber]
		if !ok {
			return
		}
		previewRows[idx].IsValid = false
		previewRows[idx].Include = false
		previewRows[idx].Errors = append(previewRows[idx].Errors, rowErrors...)
	}

	supplierIDs := []uuid.UUID{}
	for _, row := range parsedRows {
		parsedSupplierID, err := uuid.Parse(row.SupplierID)
		if err != nil {
			appendRowErrors(row.RowNumber, "supplier_id must be a valid UUID")
			continue
		}
		supplierIDs = append(supplierIDs, parsedSupplierID)
	}

	existingSuppliers := map[string]struct{}{}
	if len(supplierIDs) > 0 {
		rows, err := Client.Supplier.
			Query().
			Where(
				supplier.ClientIDEQ(clientID),
				supplier.IDIn(supplierIDs...),
				supplier.IsActiveEQ(true),
			).
			All(ctx)
		if err != nil {
			return nil, nil, nil, errs.B().Code(errs.Internal).Msg("failed to check suppliers for import").Cause(err).Err()
		}
		for _, row := range rows {
			existingSuppliers[row.ID.String()] = struct{}{}
		}
	}

	seenInFile := map[string]int{}
	for _, row := range parsedRows {
		if _, ok := existingSuppliers[row.SupplierID]; !ok {
			appendRowErrors(row.RowNumber, "supplier not found")
		}

		key := buildContractDuplicateKey(row.SupplierID, row.Request.ContractNumber)
		if firstRow, seen := seenInFile[key]; seen {
			appendRowErrors(firstRow, "duplicate contract in file")
			appendRowErrors(row.RowNumber, "duplicate contract in file")
		} else {
			seenInFile[key] = row.RowNumber
		}
	}

	if len(supplierIDs) > 0 {
		existingContracts, err := Client.ContractSupplier.
			Query().
			Where(
				contractsupplier.SupplierIDIn(supplierIDs...),
				contractsupplier.IsActiveEQ(true),
			).
			All(ctx)
		if err != nil {
			return nil, nil, nil, errs.B().Code(errs.Internal).Msg("failed to check existing contracts").Cause(err).Err()
		}

		existingKeys := map[string]struct{}{}
		for _, row := range existingContracts {
			existingKeys[buildContractDuplicateKey(row.SupplierID.String(), row.ContractNumber)] = struct{}{}
		}

		for _, row := range parsedRows {
			key := buildContractDuplicateKey(row.SupplierID, row.Request.ContractNumber)
			if _, exists := existingKeys[key]; exists {
				appendRowErrors(row.RowNumber, "contract already exists in database")
			}
		}
	}

	validRows := []parsedContractRow{}
	for _, row := range parsedRows {
		idx, ok := rowIndex[row.RowNumber]
		if !ok {
			continue
		}
		if previewRows[idx].IsValid {
			validRows = append(validRows, row)
		}
	}

	return validRows, previewRows, validationErrors, nil
}

func parseContractCSV(data []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	return reader.ReadAll()
}

func parseContractXLSX(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	for i, row := range rows {
		for j, cell := range row {
			rows[i][j] = normalizeContractExcelValue(cell)
		}
	}

	return rows, nil
}

func normalizeContractHeader(header string) string {
	h := strings.ToLower(strings.TrimSpace(header))
	h = strings.ReplaceAll(h, " ", "_")
	h = strings.ReplaceAll(h, "-", "_")
	return h
}

func normalizeContractExcelValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "eE") {
		f, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
	}
	return value
}

func normalizeContractCurrencyValue(currency *string) string {
	if currency == nil {
		return defaultContractCurrency
	}
	normalized := strings.ToUpper(strings.TrimSpace(*currency))
	if normalized == "" {
		return defaultContractCurrency
	}
	return normalized
}

func isAllowedContractCurrency(currency string) bool {
	_, ok := allowedContractCurrencies[currency]
	return ok
}

func isContractRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func buildContractDuplicateKey(supplierID, contractNumber string) string {
	return strings.ToLower(strings.TrimSpace(supplierID)) + "|" + strings.ToLower(strings.TrimSpace(contractNumber))
}

func strPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
