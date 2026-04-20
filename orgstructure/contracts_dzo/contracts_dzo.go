package contracts_dzo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

const contractColumns = `
	id,
	dzo_id,
	contract_number,
	category,
	signed_date,
	expiry_date,
	amount_with_vat::float8,
	amendment_number,
	amendment_date,
	amendment_amount::float8,
	total_amount::float8,
	spent_amount::float8,
	remaining_amount::float8,
	is_active,
	created_at,
	updated_at
`

// ════ DATABASE ════

var db = sqldb.Named("lms")

// ════ ENDPOINTS ════

// CreateContractDZO creates a new DZO contract.
//
//encore:api public method=POST path=/contracts-dzo
func CreateContractDZO(ctx context.Context, req *CreateContractDZORequest) (*GetContractDZOResponse, error) {
	if strings.TrimSpace(req.DZOID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id is required").Err()
	}
	if _, err := uuid.Parse(req.DZOID); err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
	}
	if strings.TrimSpace(req.ContractNumber) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("contract_number is required").Err()
	}
	if strings.TrimSpace(req.Category) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("category is required").Err()
	}
	if _, err := parseDate(req.SignedDate); err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("signed_date must be in YYYY-MM-DD format").Err()
	}
	if _, err := parseNullableDate(req.ExpiryDate); err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("expiry_date must be in YYYY-MM-DD format").Err()
	}
	if req.AmountWithVAT <= 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("amount_with_vat must be greater than 0").Err()
	}

	contract, err := insertContract(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GetContractDZOResponse{Contract: *contract}, nil
}

// ListContractsDZO lists contracts with optional filters.
//
//encore:api public method=GET path=/contracts-dzo
func ListContractsDZO(ctx context.Context, req *ListContractsDZORequest) (*ListContractsDZOResponse, error) {
	contracts, err := queryContracts(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ListContractsDZOResponse{
		Contracts: contracts,
		Total:     len(contracts),
	}, nil
}

// GetContractDZO returns a contract by ID.
//
//encore:api public method=GET path=/contracts-dzo/:id
func GetContractDZO(ctx context.Context, id string) (*GetContractDZOResponse, error) {
	contract, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetContractDZOResponse{Contract: *contract}, nil
}

// UpdateContractDZO partially updates a contract.
//
//encore:api public method=PATCH path=/contracts-dzo/:id
func UpdateContractDZO(ctx context.Context, id string, req *UpdateContractDZORequest) (*GetContractDZOResponse, error) {
	if req.DZOID != nil {
		if strings.TrimSpace(*req.DZOID) == "" {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id must not be empty").Err()
		}
		if _, err := uuid.Parse(*req.DZOID); err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
	}
	if req.ContractNumber != nil && strings.TrimSpace(*req.ContractNumber) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("contract_number must not be empty").Err()
	}
	if req.Category != nil && strings.TrimSpace(*req.Category) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("category must not be empty").Err()
	}
	if req.SignedDate != nil {
		if _, err := parseDate(*req.SignedDate); err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("signed_date must be in YYYY-MM-DD format").Err()
		}
	}
	if req.ExpiryDate != nil {
		if _, err := parseNullableDate(req.ExpiryDate); err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("expiry_date must be in YYYY-MM-DD format").Err()
		}
	}
	if req.AmountWithVAT != nil && *req.AmountWithVAT <= 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("amount_with_vat must be greater than 0").Err()
	}

	contract, err := updateContract(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetContractDZOResponse{Contract: *contract}, nil
}

// DeleteContractDZO soft-deletes a contract.
//
//encore:api public method=DELETE path=/contracts-dzo/:id
func DeleteContractDZO(ctx context.Context, id string) (*DeleteContractDZOResponse, error) {
	if err := softDeleteContract(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteContractDZOResponse{Message: "contract deleted successfully"}, nil
}

// AddContractDZOAmendment applies amendment data and recalculates totals.
//
//encore:api public method=POST path=/contracts-dzo/:id/amendment
func AddContractDZOAmendment(ctx context.Context, id string, req *AddAmendmentRequest) (*GetContractDZOResponse, error) {
	if req.AmendmentAmount < 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("amendment_amount must be greater or equal to 0").Err()
	}
	if req.AmendmentDate != nil {
		if _, err := parseNullableDate(req.AmendmentDate); err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("amendment_date must be in YYYY-MM-DD format").Err()
		}
	}

	contract, err := applyAmendment(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return &GetContractDZOResponse{Contract: *contract}, nil
}

// GetContractDZOAnalytics returns budget analytics for a contract.
//
//encore:api public method=GET path=/contracts-dzo/:id/analytics
func GetContractDZOAnalytics(ctx context.Context, id string) (*ContractDZOAnalyticsResponse, error) {
	analytics, err := queryContractAnalytics(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ContractDZOAnalyticsResponse{Analytics: *analytics}, nil
}

// SpendContractDZOBudget spends contract budget and updates remaining amount.
//
//encore:api public method=POST path=/contracts-dzo/:id/spend
func SpendContractDZOBudget(ctx context.Context, id string, req *SpendContractBudgetRequest) (*GetContractDZOResponse, error) {
	if req.Amount <= 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("amount must be greater than 0").Err()
	}

	contract, err := spendContractBudget(ctx, id, req.Amount)
	if err != nil {
		return nil, err
	}

	return &GetContractDZOResponse{Contract: *contract}, nil
}

// ════ INTERNAL ════

func insertContract(ctx context.Context, req *CreateContractDZORequest) (*ContractDZO, error) {
	dzoID, _ := uuid.Parse(req.DZOID)
	signedDate, _ := parseDate(req.SignedDate)
	expiryDate, _ := parseNullableDate(req.ExpiryDate)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := fmt.Sprintf(`
		INSERT INTO contracts_dzo (
			dzo_id,
			contract_number,
			category,
			signed_date,
			expiry_date,
			amount_with_vat,
			amendment_amount,
			total_amount,
			spent_amount,
			remaining_amount,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $6, 0, $6, $7)
		RETURNING %s
	`, contractColumns)

	row := db.QueryRow(ctx, query,
		dzoID,
		req.ContractNumber,
		req.Category,
		signedDate,
		expiryDate,
		req.AmountWithVAT,
		isActive,
	)

	contract, err := scanContract(row)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id does not exist").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to create contract").Cause(err).Err()
	}

	return contract, nil
}

func queryContracts(ctx context.Context, req *ListContractsDZORequest) ([]ContractDZO, error) {
	if req == nil {
		req = &ListContractsDZORequest{}
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM contracts_dzo
		WHERE 1=1
	`, contractColumns)

	args := make([]interface{}, 0, 3)
	argPos := 1

	if req.IsActive != "" {
		isActive, err := strconv.ParseBool(req.IsActive)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("is_active must be true or false").Err()
		}
		query += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, isActive)
		argPos++
	}

	if req.DZOID != "" {
		dzoID, err := uuid.Parse(req.DZOID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
		query += fmt.Sprintf(" AND dzo_id = $%d", argPos)
		args = append(args, dzoID)
		argPos++
	}

	if req.RemainingAmountLT != "" {
		remainingAmountLT, err := strconv.ParseFloat(req.RemainingAmountLT, 64)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("remaining_amount_lt must be a number").Err()
		}
		query += fmt.Sprintf(" AND remaining_amount < $%d", argPos)
		args = append(args, remainingAmountLT)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list contracts").Cause(err).Err()
	}
	defer rows.Close()

	contracts := []ContractDZO{}
	for rows.Next() {
		contract, scanErr := scanContract(rows)
		if scanErr != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to read contracts").Cause(scanErr).Err()
		}
		contracts = append(contracts, *contract)
	}

	if err = rows.Err(); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to iterate contracts").Cause(err).Err()
	}

	return contracts, nil
}

func queryContractByID(ctx context.Context, id string) (*ContractDZO, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM contracts_dzo
		WHERE id = $1
	`, contractColumns)

	row := db.QueryRow(ctx, query, uid)
	contract, err := scanContract(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errs.Code(err) == errs.NotFound {
			return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get contract").Cause(err).Err()
	}

	return contract, nil
}

func updateContract(ctx context.Context, id string, req *UpdateContractDZORequest) (*ContractDZO, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	setClauses := make([]string, 0, 8)
	args := make([]interface{}, 0, 9)
	args = append(args, uid)
	argPos := 2

	if req.DZOID != nil {
		dzoID, parseErr := uuid.Parse(*req.DZOID)
		if parseErr != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid dzo_id format").Err()
		}
		setClauses = append(setClauses, fmt.Sprintf("dzo_id = $%d", argPos))
		args = append(args, dzoID)
		argPos++
	}
	if req.ContractNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("contract_number = $%d", argPos))
		args = append(args, strings.TrimSpace(*req.ContractNumber))
		argPos++
	}
	if req.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argPos))
		args = append(args, strings.TrimSpace(*req.Category))
		argPos++
	}
	if req.SignedDate != nil {
		signedDate, _ := parseDate(*req.SignedDate)
		setClauses = append(setClauses, fmt.Sprintf("signed_date = $%d", argPos))
		args = append(args, signedDate)
		argPos++
	}
	if req.ExpiryDate != nil {
		expiryDate, _ := parseNullableDate(req.ExpiryDate)
		setClauses = append(setClauses, fmt.Sprintf("expiry_date = $%d", argPos))
		args = append(args, expiryDate)
		argPos++
	}
	if req.AmountWithVAT != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount_with_vat = $%d", argPos))
		args = append(args, *req.AmountWithVAT)
		argPos++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	if len(setClauses) == 0 {
		return queryContractByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE contracts_dzo
		SET %s
		WHERE id = $1
		RETURNING id
	`, strings.Join(setClauses, ", "))

	var updatedID uuid.UUID
	err = db.QueryRow(ctx, query, args...).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errs.Code(err) == errs.NotFound {
			return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
		}
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("dzo_id does not exist").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update contract").Cause(err).Err()
	}

	row := db.QueryRow(ctx, fmt.Sprintf(`
		UPDATE contracts_dzo
		SET
			total_amount = amount_with_vat + COALESCE(amendment_amount, 0),
			remaining_amount = (amount_with_vat + COALESCE(amendment_amount, 0)) - spent_amount,
			updated_at = NOW()
		WHERE id = $1
		RETURNING %s
	`, contractColumns), updatedID)

	contract, err := scanContract(row)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to update contract").Cause(err).Err()
	}

	return contract, nil
}

func softDeleteContract(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	result, err := db.Exec(ctx, `
		UPDATE contracts_dzo
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`, uid)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete contract").Cause(err).Err()
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errs.B().Code(errs.NotFound).Msg("contract not found").Err()
	}

	return nil
}

func applyAmendment(ctx context.Context, id string, req *AddAmendmentRequest) (*ContractDZO, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	var amendmentNumber sql.NullString
	if req.AmendmentNumber != nil {
		trimmed := strings.TrimSpace(*req.AmendmentNumber)
		if trimmed != "" {
			amendmentNumber = sql.NullString{String: trimmed, Valid: true}
		}
	}

	amendmentDate, _ := parseNullableDate(req.AmendmentDate)

	query := fmt.Sprintf(`
		UPDATE contracts_dzo
		SET
			amendment_number = $2,
			amendment_date = $3,
			amendment_amount = $4,
			total_amount = amount_with_vat + $4,
			remaining_amount = (amount_with_vat + $4) - spent_amount,
			updated_at = NOW()
		WHERE id = $1
		RETURNING %s
	`, contractColumns)

	row := db.QueryRow(ctx, query, uid, amendmentNumber, amendmentDate, req.AmendmentAmount)
	contract, err := scanContract(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errs.B().Code(errs.NotFound).Msg("contract not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to apply amendment").Cause(err).Err()
	}

	return contract, nil
}

func spendContractBudget(ctx context.Context, id string, amount float64) (*ContractDZO, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	query := fmt.Sprintf(`
		UPDATE contracts_dzo
		SET
			spent_amount = spent_amount + $2,
			remaining_amount = total_amount - (spent_amount + $2),
			updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE AND remaining_amount >= $2
		RETURNING %s
	`, contractColumns)

	row := db.QueryRow(ctx, query, uid, amount)
	contract, err := scanContract(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errs.Code(err) == errs.NotFound {
			existing, findErr := queryContractByID(ctx, id)
			if findErr != nil {
				return nil, findErr
			}
			if !existing.IsActive {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("cannot spend from inactive contract").Err()
			}
			if existing.RemainingAmount < amount {
				return nil, errs.B().Code(errs.InvalidArgument).Msg("insufficient remaining amount").Err()
			}
			return nil, errs.B().Code(errs.Internal).Msg("failed to spend contract budget").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to spend contract budget").Cause(err).Err()
	}

	return contract, nil
}

func queryContractAnalytics(ctx context.Context, id string) (*ContractDZOAnalytics, error) {
	contract, err := queryContractByID(ctx, id)
	if err != nil {
		return nil, err
	}

	utilization := 0.0
	if contract.TotalAmount > 0 {
		utilization = (contract.SpentAmount / contract.TotalAmount) * 100
	}

	return &ContractDZOAnalytics{
		ContractID:         contract.ID,
		TotalAmount:        contract.TotalAmount,
		SpentAmount:        contract.SpentAmount,
		RemainingAmount:    contract.RemainingAmount,
		UtilizationPercent: utilization,
		IsActive:           contract.IsActive,
	}, nil
}

func scanContract(scanner interface {
	Scan(dest ...interface{}) error
}) (*ContractDZO, error) {
	var (
		id              uuid.UUID
		dzoID           uuid.UUID
		signedDate      time.Time
		expiryDate      sql.NullTime
		amountWithVAT   float64
		amendmentNumber sql.NullString
		amendmentDate   sql.NullTime
		amendmentAmount sql.NullFloat64
		totalAmount     float64
		spentAmount     float64
		remainingAmount float64
		isActive        bool
		createdAt       time.Time
		updatedAt       time.Time
		contractNumber  string
		category        string
	)

	err := scanner.Scan(
		&id,
		&dzoID,
		&contractNumber,
		&category,
		&signedDate,
		&expiryDate,
		&amountWithVAT,
		&amendmentNumber,
		&amendmentDate,
		&amendmentAmount,
		&totalAmount,
		&spentAmount,
		&remainingAmount,
		&isActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	var expiryDateValue *string
	if expiryDate.Valid {
		s := expiryDate.Time.Format(dateLayout)
		expiryDateValue = &s
	}

	var amendmentNumberValue *string
	if amendmentNumber.Valid {
		s := amendmentNumber.String
		amendmentNumberValue = &s
	}

	var amendmentDateValue *string
	if amendmentDate.Valid {
		s := amendmentDate.Time.Format(dateLayout)
		amendmentDateValue = &s
	}

	var amendmentAmountValue *float64
	if amendmentAmount.Valid {
		v := amendmentAmount.Float64
		amendmentAmountValue = &v
	}

	return &ContractDZO{
		ID:              id.String(),
		DZOID:           dzoID.String(),
		ContractNumber:  contractNumber,
		Category:        category,
		SignedDate:      signedDate.Format(dateLayout),
		ExpiryDate:      expiryDateValue,
		AmountWithVAT:   amountWithVAT,
		AmendmentNumber: amendmentNumberValue,
		AmendmentDate:   amendmentDateValue,
		AmendmentAmount: amendmentAmountValue,
		TotalAmount:     totalAmount,
		SpentAmount:     spentAmount,
		RemainingAmount: remainingAmount,
		IsActive:        isActive,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

func parseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, strings.TrimSpace(value))
}

func parseNullableDate(value *string) (sql.NullTime, error) {
	if value == nil {
		return sql.NullTime{}, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return sql.NullTime{}, nil
	}

	tm, err := time.Parse(dateLayout, trimmed)
	if err != nil {
		return sql.NullTime{}, err
	}

	return sql.NullTime{Time: tm, Valid: true}, nil
}
