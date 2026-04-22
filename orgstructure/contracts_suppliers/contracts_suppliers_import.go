package contractssuppliers

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"encore.app/auth/authhandler"
	"encore.app/db/ent/supplier"
	csh "encore.app/orgstructure/contracts_suppliers_history"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// ImportContracts bulk-imports contracts from a .csv or .xlsx file.
// The file's header row names the columns (case-insensitive). Missing optional
// columns are allowed; missing required columns fail the whole import.
// Rows that fail validation are skipped; both imported count and per-row
// errors are returned.
//
//encore:api auth method=POST path=/contracts-suppliers/import
func ImportContracts(ctx context.Context, req *ImportContractsRequest) (*ImportResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}
	if err := validateImportRequest(req); err != nil {
		return nil, err
	}

	ud, ok := auth.Data().(*authhandler.AuthData)
	if !ok || ud.CompanyID == "" {
		return nil, errs.B().Code(errs.PermissionDenied).Msg("missing company in token").Err()
	}
	clientUID, err := uuid.Parse(ud.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("invalid company id in token").Err()
	}

	rows, err := parseImportFile(req.FileName, req.FileData)
	if err != nil {
		return nil, err
	}

	converted, rowErrors := validateAndConvertRows(rows)

	supplierByBin, supplierByName, err := buildSupplierLookup(ctx, clientUID, converted)
	if err != nil {
		return nil, err
	}

	imported := 0
	for _, row := range converted {
		supID, ok := resolveSupplierID(row, supplierByBin, supplierByName)
		if !ok {
			rowErrors = append(rowErrors, fmt.Sprintf(
				"row %d: supplier not found (bin_or_iin=%q, name=%q)",
				row.rowNum, row.supplierBinOrIin, row.supplierName))
			continue
		}
		if err := insertImportedContract(ctx, clientUID, supID, row); err != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: %v", row.rowNum, err))
			continue
		}
		imported++
	}

	return &ImportResponse{
		Imported: imported,
		Failed:   len(rows) - imported,
		Errors:   rowErrors,
	}, nil
}

// ════ REQUEST VALIDATION ════

const (
	maxImportFileSize = 25 * 1024 * 1024
	extXLSX           = ".xlsx"
	extCSV            = ".csv"
)

func validateImportRequest(req *ImportContractsRequest) error {
	if req == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("request body is required").Err()
	}
	if strings.TrimSpace(req.FileName) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("file_name is required").Err()
	}
	if len(req.FileData) == 0 {
		return errs.B().Code(errs.InvalidArgument).Msg("file_data is required").Err()
	}
	if len(req.FileData) > maxImportFileSize {
		return errs.B().Code(errs.InvalidArgument).Msg("file_data exceeds 25 MB limit").Err()
	}
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if ext != extXLSX && ext != extCSV {
		return errs.B().Code(errs.InvalidArgument).Msgf("unsupported file extension %q; allowed: .xlsx, .csv", ext).Err()
	}
	return nil
}

// ════ COLUMN HEADERS ════

const (
	colContractNumber     = "contract_number"
	colSupplierBinOrIin   = "supplier_bin_or_iin"
	colSupplierName       = "supplier_name"
	colVatFlag            = "vat_flag"
	colSignedDate         = "signed_date"
	colEndDate            = "end_date"
	colAmount             = "amount"
	colAmountCurrency     = "amount_currency"
	colCurrency           = "currency"
	colBalanceAtYearEnd   = "balance_at_year_end"
	colAmendmentNumber    = "amendment_number"
	colAmendmentDate      = "amendment_date"
	colAmendmentAmount    = "amendment_amount"
	colTotalWithAmendment = "total_with_amendment"
	colRemainingAmount    = "remaining_amount"
)

// requiredColumns lists headers that must appear in the file.
// supplier_bin_or_iin OR supplier_name is required — checked per-row, not in headers.
var requiredColumns = []string{
	colContractNumber,
	colVatFlag,
	colSignedDate,
	colAmount,
	colTotalWithAmendment,
	colRemainingAmount,
}

// parsedContractRow holds raw string cells for one file row.
// An empty string means "cell absent or blank"; type conversion happens later.
type parsedContractRow struct {
	rowNum int
	cells  map[string]string
}

func (r parsedContractRow) get(col string) string {
	return strings.TrimSpace(r.cells[col])
}

// ════ PARSING ════

func parseImportFile(fileName string, data []byte) ([]parsedContractRow, error) {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case extCSV:
		return parseContractsCSV(data)
	case extXLSX:
		return parseContractsXLSX(data)
	default:
		return nil, errs.B().Code(errs.InvalidArgument).Msg("unsupported file type").Err()
	}
}

func parseContractsCSV(data []byte) ([]parsedContractRow, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("failed to read CSV header").Cause(err).Err()
	}
	headerIdx, err := indexHeader(header)
	if err != nil {
		return nil, err
	}

	var rows []parsedContractRow
	rowNum := 1 // header is row 1; data starts at 2
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msgf("CSV parse error on row %d", rowNum+1).Cause(err).Err()
		}
		rowNum++
		if isEmptyRecord(record) {
			continue
		}
		rows = append(rows, buildParsedRow(rowNum, record, headerIdx))
	}
	return rows, nil
}

func parseContractsXLSX(data []byte) ([]parsedContractRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("failed to open xlsx").Cause(err).Err()
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("xlsx has no sheets").Err()
	}

	raw, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("failed to read xlsx rows").Cause(err).Err()
	}
	if len(raw) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("xlsx is empty").Err()
	}

	headerIdx, err := indexHeader(raw[0])
	if err != nil {
		return nil, err
	}

	var rows []parsedContractRow
	for i := 1; i < len(raw); i++ {
		if isEmptyRecord(raw[i]) {
			continue
		}
		rows = append(rows, buildParsedRow(i+1, raw[i], headerIdx))
	}
	return rows, nil
}

func indexHeader(header []string) (map[string]int, error) {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		name := normalizeColumnName(h)
		if name == "" {
			continue
		}
		idx[name] = i
	}
	for _, col := range requiredColumns {
		if _, ok := idx[col]; !ok {
			return nil, errs.B().Code(errs.InvalidArgument).
				Msgf("required column %q is missing from header", col).Err()
		}
	}
	if _, hasBin := idx[colSupplierBinOrIin]; !hasBin {
		if _, hasName := idx[colSupplierName]; !hasName {
			return nil, errs.B().Code(errs.InvalidArgument).
				Msgf("at least one of %q or %q must be present in header", colSupplierBinOrIin, colSupplierName).Err()
		}
	}
	return idx, nil
}

func normalizeColumnName(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isEmptyRecord(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func buildParsedRow(rowNum int, record []string, headerIdx map[string]int) parsedContractRow {
	cells := make(map[string]string, len(headerIdx))
	for col, i := range headerIdx {
		if i < len(record) {
			cells[col] = record[i]
		}
	}
	return parsedContractRow{rowNum: rowNum, cells: cells}
}

// ════ ROW VALIDATION + CONVERSION ════

// convertedContractRow is a typed row ready for DB insert.
type convertedContractRow struct {
	rowNum             int
	supplierBinOrIin   string
	supplierName       string
	contractNumber     string
	vatFlag            bool
	signedDate         time.Time
	endDate            *time.Time
	amount             float64
	amountCurrency     *float64
	currency           *string
	balanceAtYearEnd   *float64
	amendmentNumber    *string
	amendmentDate      *time.Time
	amendmentAmount    *float64
	totalWithAmendment float64
	remainingAmount    float64
}

func validateAndConvertRows(rows []parsedContractRow) ([]convertedContractRow, []string) {
	var out []convertedContractRow
	var errors []string
	for _, r := range rows {
		c, err := convertRow(r)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: %s", r.rowNum, err.Error()))
			continue
		}
		out = append(out, c)
	}
	return out, errors
}

func convertRow(r parsedContractRow) (convertedContractRow, error) {
	var c convertedContractRow
	c.rowNum = r.rowNum
	c.supplierBinOrIin = r.get(colSupplierBinOrIin)
	c.supplierName = r.get(colSupplierName)
	if c.supplierBinOrIin == "" && c.supplierName == "" {
		return c, fmt.Errorf("supplier_bin_or_iin or supplier_name is required")
	}

	c.contractNumber = r.get(colContractNumber)
	if c.contractNumber == "" {
		return c, fmt.Errorf("contract_number is required")
	}

	vat, err := parseBoolCell(r.get(colVatFlag))
	if err != nil {
		return c, fmt.Errorf("vat_flag: %w", err)
	}
	c.vatFlag = vat

	signed, err := parseDateCell(r.get(colSignedDate))
	if err != nil {
		return c, fmt.Errorf("signed_date: %w", err)
	}
	if signed == nil {
		return c, fmt.Errorf("signed_date is required")
	}
	c.signedDate = *signed

	c.endDate, err = parseDateCell(r.get(colEndDate))
	if err != nil {
		return c, fmt.Errorf("end_date: %w", err)
	}
	if c.endDate != nil && !c.endDate.After(c.signedDate) {
		return c, fmt.Errorf("end_date must be after signed_date")
	}

	amount, err := parseRequiredFloat(r.get(colAmount))
	if err != nil {
		return c, fmt.Errorf("amount: %w", err)
	}
	c.amount = amount
	if c.amount < 0 {
		return c, fmt.Errorf("amount must be >= 0")
	}

	total, err := parseRequiredFloat(r.get(colTotalWithAmendment))
	if err != nil {
		return c, fmt.Errorf("total_with_amendment: %w", err)
	}
	c.totalWithAmendment = total

	remaining, err := parseRequiredFloat(r.get(colRemainingAmount))
	if err != nil {
		return c, fmt.Errorf("remaining_amount: %w", err)
	}
	c.remainingAmount = remaining

	c.amountCurrency, err = parseOptionalFloat(r.get(colAmountCurrency))
	if err != nil {
		return c, fmt.Errorf("amount_currency: %w", err)
	}

	if cur := r.get(colCurrency); cur != "" {
		c.currency = &cur
	}

	c.balanceAtYearEnd, err = parseOptionalFloat(r.get(colBalanceAtYearEnd))
	if err != nil {
		return c, fmt.Errorf("balance_at_year_end: %w", err)
	}

	if an := r.get(colAmendmentNumber); an != "" {
		c.amendmentNumber = &an
	}

	c.amendmentDate, err = parseDateCell(r.get(colAmendmentDate))
	if err != nil {
		return c, fmt.Errorf("amendment_date: %w", err)
	}

	c.amendmentAmount, err = parseOptionalFloat(r.get(colAmendmentAmount))
	if err != nil {
		return c, fmt.Errorf("amendment_amount: %w", err)
	}

	return c, nil
}

// ════ CELL PARSERS ════

func parseBoolCell(s string) (bool, error) {
	if s == "" {
		return false, fmt.Errorf("required")
	}
	switch strings.ToLower(s) {
	case "true", "1", "yes", "да", "с ндс":
		return true, nil
	case "false", "0", "no", "нет", "без ндс":
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value %q", s)
}

// parseDateCell accepts several common formats. Empty cell → nil, nil.
// Excelize returns dates as pre-formatted strings when the sheet has a date format;
// otherwise as a numeric string (Excel serial). We handle both.
func parseDateCell(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"02.01.2006",
		"02/01/2006",
		"01/02/2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	if serial, err := strconv.ParseFloat(s, 64); err == nil {
		t, err := excelize.ExcelDateToTime(serial, false)
		if err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid date %q", s)
}

func parseRequiredFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("required")
	}
	return parseFloat(s)
}

func parseOptionalFloat(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := parseFloat(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// parseFloat tolerates spaces (thousand separators in pasted data) and comma decimals.
func parseFloat(s string) (float64, error) {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.Replace(s, ",", ".", 1)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	return v, nil
}

// ════ SUPPLIER RESOLUTION ════

// buildSupplierLookup queries all suppliers referenced by the rows in one shot
// and builds bin→id and name-lowercased→id maps scoped to the caller's client.
func buildSupplierLookup(ctx context.Context, clientID uuid.UUID, rows []convertedContractRow) (map[string]uuid.UUID, map[string]uuid.UUID, error) {
	binSet := make(map[string]struct{})
	nameSet := make(map[string]struct{})
	for _, r := range rows {
		if r.supplierBinOrIin != "" {
			binSet[r.supplierBinOrIin] = struct{}{}
		}
		if r.supplierName != "" {
			nameSet[r.supplierName] = struct{}{}
		}
	}

	bins := setToSlice(binSet)
	names := setToSlice(nameSet)

	q := Client.Supplier.Query().Where(supplier.ClientIDEQ(clientID), supplier.IsActive(true))
	switch {
	case len(bins) > 0 && len(names) > 0:
		q = q.Where(supplier.Or(supplier.BinOrIinIn(bins...), supplier.NameIn(names...)))
	case len(bins) > 0:
		q = q.Where(supplier.BinOrIinIn(bins...))
	case len(names) > 0:
		q = q.Where(supplier.NameIn(names...))
	default:
		return map[string]uuid.UUID{}, map[string]uuid.UUID{}, nil
	}

	found, err := q.All(ctx)
	if err != nil {
		return nil, nil, errs.B().Code(errs.Internal).Msg("failed to load suppliers").Cause(err).Err()
	}

	byBin := make(map[string]uuid.UUID, len(found))
	byName := make(map[string]uuid.UUID, len(found))
	for _, s := range found {
		if s.BinOrIin != nil && *s.BinOrIin != "" {
			byBin[*s.BinOrIin] = s.ID
		}
		byName[strings.ToLower(s.Name)] = s.ID
	}
	return byBin, byName, nil
}

func resolveSupplierID(row convertedContractRow, byBin, byName map[string]uuid.UUID) (uuid.UUID, bool) {
	if row.supplierBinOrIin != "" {
		if id, ok := byBin[row.supplierBinOrIin]; ok {
			return id, true
		}
	}
	if row.supplierName != "" {
		if id, ok := byName[strings.ToLower(row.supplierName)]; ok {
			return id, true
		}
	}
	return uuid.Nil, false
}

func setToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// ════ INSERT ════

func insertImportedContract(ctx context.Context, clientID, supplierID uuid.UUID, row convertedContractRow) error {
	builder := Client.ContractSupplier.Create().
		SetSupplierID(supplierID).
		SetContractNumber(row.contractNumber).
		SetVatFlag(row.vatFlag).
		SetSignedDate(row.signedDate).
		SetNillableEndDate(row.endDate).
		SetAmount(row.amount).
		SetNillableAmountCurrency(row.amountCurrency).
		SetNillableCurrency(row.currency).
		SetNillableBalanceAtYearEnd(row.balanceAtYearEnd).
		SetNillableAmendmentNumber(row.amendmentNumber).
		SetNillableAmendmentDate(row.amendmentDate).
		SetNillableAmendmentAmount(row.amendmentAmount).
		SetTotalWithAmendment(row.totalWithAmendment).
		SetRemainingAmount(row.remainingAmount).
		SetIsActive(true)

	inserted, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert: %w", err)
	}

	if auditErr := csh.InsertAuditRecord(ctx, inserted.ID.String(), csh.OpCreate, nil, csh.EntToContract(inserted)); auditErr != nil {
		rlog.Error("contracts-suppliers: failed to write audit record on import",
			"contract_id", inserted.ID.String(), "err", auditErr)
	}
	return nil
}
