package contracts_dzo

import "time"

// ContractDZO is the domain model for contracts_dzo.
type ContractDZO struct {
	ID              string    `json:"id"`
	DZOID           string    `json:"dzo_id"`
	ContractNumber  string    `json:"contract_number"`
	Category        string    `json:"category"`
	SignedDate      string    `json:"signed_date"`
	ExpiryDate      *string   `json:"expiry_date"`
	AmountWithVAT   float64   `json:"amount_with_vat"`
	AmendmentNumber *string   `json:"amendment_number"`
	AmendmentDate   *string   `json:"amendment_date"`
	AmendmentAmount *float64  `json:"amendment_amount"`
	TotalAmount     float64   `json:"total_amount"`
	SpentAmount     float64   `json:"spent_amount"`
	RemainingAmount float64   `json:"remaining_amount"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateContractDZORequest is the request for creating a contract.
type CreateContractDZORequest struct {
	DZOID          string  `json:"dzo_id"`
	ContractNumber string  `json:"contract_number"`
	Category       string  `json:"category"`
	SignedDate     string  `json:"signed_date"`
	ExpiryDate     *string `json:"expiry_date,omitempty"`
	AmountWithVAT  float64 `json:"amount_with_vat"`
	IsActive       *bool   `json:"is_active,omitempty"`
}

// UpdateContractDZORequest is the PATCH request for a contract.
type UpdateContractDZORequest struct {
	DZOID          *string  `json:"dzo_id,omitempty"`
	ContractNumber *string  `json:"contract_number,omitempty"`
	Category       *string  `json:"category,omitempty"`
	SignedDate     *string  `json:"signed_date,omitempty"`
	ExpiryDate     *string  `json:"expiry_date,omitempty"`
	AmountWithVAT  *float64 `json:"amount_with_vat,omitempty"`
	IsActive       *bool    `json:"is_active,omitempty"`
}

// AddAmendmentRequest is the request for adding/updating amendment data.
type AddAmendmentRequest struct {
	AmendmentNumber *string `json:"amendment_number,omitempty"`
	AmendmentDate   *string `json:"amendment_date,omitempty"`
	AmendmentAmount float64 `json:"amendment_amount"`
}

// SpendContractBudgetRequest is the request for spending contract budget.
type SpendContractBudgetRequest struct {
	Amount float64 `json:"amount"`
}

// ListContractsDZORequest is the request for filtering contracts.
type ListContractsDZORequest struct {
	IsActive          string `query:"is_active" json:"is_active,omitempty"`
	DZOID             string `query:"dzo_id" json:"dzo_id,omitempty"`
	RemainingAmountLT string `query:"remaining_amount_lt" json:"remaining_amount_lt,omitempty"`
}

// ContractDZOAnalytics is analytics for a contract.
type ContractDZOAnalytics struct {
	ContractID         string  `json:"contract_id"`
	TotalAmount        float64 `json:"total_amount"`
	SpentAmount        float64 `json:"spent_amount"`
	RemainingAmount    float64 `json:"remaining_amount"`
	UtilizationPercent float64 `json:"utilization_percent"`
	IsActive           bool    `json:"is_active"`
}

// GetContractDZOResponse is the response for a single contract.
type GetContractDZOResponse struct {
	Contract ContractDZO `json:"contract"`
}

// ListContractsDZOResponse is the response for listing contracts.
type ListContractsDZOResponse struct {
	Contracts []ContractDZO `json:"contracts"`
	Total     int           `json:"total"`
}

// DeleteContractDZOResponse is the response for deleting a contract.
type DeleteContractDZOResponse struct {
	Message string `json:"message"`
}

// ContractDZOAnalyticsResponse is the response for contract analytics.
type ContractDZOAnalyticsResponse struct {
	Analytics ContractDZOAnalytics `json:"analytics"`
}
