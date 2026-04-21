package contractssuppliers

import (
	"time"
)

// ContractStatus is the computed lifecycle status of a contract.
// It is not stored in the DB — derived from signed_date + contract duration.
type ContractStatus string

const (
	StatusActive        ContractStatus = "ACTIVE"
	StatusExpired       ContractStatus = "EXPIRED"
	StatusExpiringSoon  ContractStatus = "EXPIRING_SOON"
)

func (s ContractStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusExpired, StatusExpiringSoon:
		return true
	}
	return false
}

// ContractSupplier is the domain model for a row in contract_suppliers.
//
// TODO: extend with calculation fields once team confirms scope —
// cost_per_person, head_count, local_content_share_pct, contract_category,
// actual_spent, uso_lektor_cost (see calc-logic spreadsheet).
type ContractSupplier struct {
	ID                 string     `json:"id"`
	SupplierID         string     `json:"supplier_id"`
	ContractNumber     string     `json:"contract_number"`
	VatFlag            bool       `json:"vat_flag"`
	SignedDate         time.Time  `json:"signed_date"`
	Amount             float64    `json:"amount"`
	AmountCurrency     *float64   `json:"amount_currency,omitempty"`
	Currency           *string    `json:"currency,omitempty"`
	BalanceAtYearEnd   *float64   `json:"balance_at_year_end,omitempty"`
	AmendmentNumber    *string    `json:"amendment_number,omitempty"`
	AmendmentDate      *time.Time `json:"amendment_date,omitempty"`
	AmendmentAmount    *float64   `json:"amendment_amount,omitempty"`
	TotalWithAmendment float64    `json:"total_with_amendment"`
	RemainingAmount    float64    `json:"remaining_amount"`
	IsActive           bool       `json:"is_active"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// ════ REQUESTS ════

// CreateContractRequest is the body for POST /suppliers/:id/contracts.
// supplier_id is taken from the URL path, not the body.
type CreateContractRequest struct {
	ContractNumber   string     `json:"contract_number"`
	VatFlag          bool       `json:"vat_flag"`
	SignedDate       time.Time  `json:"signed_date"`
	Amount           float64    `json:"amount"`
	AmountCurrency   *float64   `json:"amount_currency,omitempty"`
	Currency         *string    `json:"currency,omitempty"`
	BalanceAtYearEnd *float64   `json:"balance_at_year_end,omitempty"`
}

// UpdateContractRequest is the body for PATCH /contracts-suppliers/id/:id.
// All fields are optional — only set fields are updated.
//
// amount is NOT editable here — use POST /amendment to change the contract
// sum via a formal amendment. That keeps derived fields (total_with_amendment,
// remaining_amount) consistent with their audit trail.
type UpdateContractRequest struct {
	ContractNumber   *string    `json:"contract_number,omitempty"`
	VatFlag          *bool      `json:"vat_flag,omitempty"`
	SignedDate       *time.Time `json:"signed_date,omitempty"`
	AmountCurrency   *float64   `json:"amount_currency,omitempty"`
	Currency         *string    `json:"currency,omitempty"`
	BalanceAtYearEnd *float64   `json:"balance_at_year_end,omitempty"`
}

// AmendmentRequest is the body for POST /contracts-suppliers/id/:id/amendment.
type AmendmentRequest struct {
	AmendmentNumber string    `json:"amendment_number"`
	AmendmentDate   time.Time `json:"amendment_date"`
	AmendmentAmount float64   `json:"amendment_amount"`
}

// SpendRequest is the body for POST /contracts-suppliers/id/:id/spend.
// Decreases remaining_amount by the given amount.
type SpendRequest struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason,omitempty"`
}

// ListContractsFilter holds query-string filters for GET /contracts-suppliers.
// Encore binds query params from the struct fields. Encore only allows
// non-pointer built-in types in query params, so "not provided" is detected
// by zero-value: empty string, zero time, zero int, false bool.
//
// include_inactive=false by default — only active contracts are returned.
// Pass include_inactive=true to see soft-deleted rows as well.
type ListContractsFilter struct {
	Status          string    `query:"status"`
	SupplierID      string    `query:"supplier_id"`
	Search          string    `query:"search"`
	ExpiryDateFrom  time.Time `query:"expiry_date_from"`
	ExpiryDateTo    time.Time `query:"expiry_date_to"`
	Page            int       `query:"page"`
	Limit           int       `query:"limit"`
	IncludeInactive bool      `query:"include_inactive"`
}

// ════ RESPONSES ════

// GetContractResponse is the response for fetching a single contract.
type GetContractResponse struct {
	Contract ContractSupplier `json:"contract"`
}

// ListContractsResponse is the paginated response for listing contracts.
type ListContractsResponse struct {
	Contracts []ContractSupplier `json:"contracts"`
	Total     int                `json:"total"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
}

// DeleteContractResponse confirms a soft-delete.
type DeleteContractResponse struct {
	Message string `json:"message"`
}

// MessageResponse is a generic message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// ImportResponse summarises the result of POST /contracts-suppliers/import.
type ImportResponse struct {
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}
