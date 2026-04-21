package contractssuppliers

import (
	"context"
	"strings"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/contractsupplier"
	csh "encore.app/orgstructure/contracts_suppliers_history"
	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
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

// Spend decreases the contract's remaining budget.
//
//encore:api auth method=POST path=/contracts-suppliers/id/:id/spend
func Spend(ctx context.Context, id string, req *SpendRequest) (*GetContractResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}
	return nil, errs.B().Code(errs.Unimplemented).Msg("Spend not implemented").Err()
}

// UploadFile attaches a contract document (pdf/jpg/png).
//
//encore:api auth method=POST path=/contracts-suppliers/id/:id/upload-file
func UploadFile(ctx context.Context, id string) (*MessageResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}
	return nil, errs.B().Code(errs.Unimplemented).Msg("UploadFile not implemented").Err()
}

// ImportContracts bulk-imports contracts from CSV/XLSX.
//
//encore:api auth method=POST path=/contracts-suppliers/import
func ImportContracts(ctx context.Context) (*ImportResponse, error) {
	if _, err := requirePermission(); err != nil {
		return nil, err
	}
	return nil, errs.B().Code(errs.Unimplemented).Msg("ImportContracts not implemented").Err()
}

// ════ INTERNAL ════

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

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
	if req.ContractNumber == nil && req.VatFlag == nil && req.SignedDate == nil &&
		req.AmountCurrency == nil && req.Currency == nil && req.BalanceAtYearEnd == nil {
		return errs.B().Code(errs.InvalidArgument).Msg("no fields to update").Err()
	}
	if req.ContractNumber != nil && strings.TrimSpace(*req.ContractNumber) == "" {
		return errs.B().Code(errs.InvalidArgument).Msg("contract_number cannot be empty").Err()
	}
	if req.SignedDate != nil && req.SignedDate.IsZero() {
		return errs.B().Code(errs.InvalidArgument).Msg("signed_date cannot be zero").Err()
	}
	return nil
}

func updateContract(ctx context.Context, row *ent.ContractSupplier, req *UpdateContractRequest) (*ent.ContractSupplier, error) {
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
	if req.AmountCurrency != nil {
		upd.SetAmountCurrency(*req.AmountCurrency)
	}
	if req.Currency != nil {
		upd.SetCurrency(*req.Currency)
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
	if req.SignedDate.IsZero() {
		return errs.B().Code(errs.InvalidArgument).Msg("signed_date is required").Err()
	}
	return nil
}

func insertContract(ctx context.Context, supplierID string, req *CreateContractRequest) (*ent.ContractSupplier, error) {
	sid, err := uuid.Parse(supplierID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid supplier_id format").Err()
	}

	row, err := Client.ContractSupplier.Create().
		SetSupplierID(sid).
		SetContractNumber(strings.TrimSpace(req.ContractNumber)).
		SetVatFlag(req.VatFlag).
		SetSignedDate(req.SignedDate).
		SetAmount(req.Amount).
		SetNillableAmountCurrency(req.AmountCurrency).
		SetNillableCurrency(req.Currency).
		SetNillableBalanceAtYearEnd(req.BalanceAtYearEnd).
		SetTotalWithAmendment(req.Amount).
		SetRemainingAmount(req.Amount).
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
		Amount:             e.Amount,
		AmountCurrency:     e.AmountCurrency,
		Currency:           e.Currency,
		BalanceAtYearEnd:   e.BalanceAtYearEnd,
		AmendmentNumber:    e.AmendmentNumber,
		AmendmentDate:      e.AmendmentDate,
		AmendmentAmount:    e.AmendmentAmount,
		TotalWithAmendment: e.TotalWithAmendment,
		RemainingAmount:    e.RemainingAmount,
		IsActive:           e.IsActive,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}
