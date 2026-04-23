package requests

import (
	"time"

	"github.com/google/uuid"
)

type RequestKind string

const (
	RequestKindRegular  RequestKind = "REGULAR"
	RequestKindClosed   RequestKind = "CLOSED"
	RequestKindArchived RequestKind = "ARCHIVED"
)

type CreateRequestRequest struct {
	EntityID   uuid.UUID `json:"entity_id"`
	EntityType string    `json:"entity_type"`
}

type ArchiveRequestContractInput struct {
	DzoID    uuid.UUID `json:"dzo_id"`
	FileName string    `json:"file_name"`
	FileURL  string    `json:"file_url"`
}

type CreateArchiveRequestRequest struct {
	Kind        string                        `json:"kind"`
	Title       *string                       `json:"title,omitempty"`
	Category    string                        `json:"category"`
	EmployeeIDs []uuid.UUID                   `json:"employee_ids"`
	Contracts   []ArchiveRequestContractInput `json:"contracts"`
}

type UpdateRequestStepRequest struct {
	Step int `json:"step"`
}

type UpdateRequestStatusRequest struct {
	Status string `json:"status"`
}

type RequestContractResponse struct {
	DzoID    uuid.UUID `json:"dzo_id"`
	FileName string    `json:"file_name"`
	FileURL  string    `json:"file_url"`
}

type RequestResponse struct {
	ID          uuid.UUID                 `json:"id"`
	InitiatorID uuid.UUID                 `json:"initiator_id"`
	EntityID    uuid.UUID                 `json:"entity_id"`
	EntityType  string                    `json:"entity_type"`
	Kind        string                    `json:"kind"`
	Title       *string                   `json:"title,omitempty"`
	Category    *string                   `json:"category,omitempty"`
	Step        int                       `json:"step"`
	Status      string                    `json:"status"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	CompletedAt *time.Time                `json:"completed_at,omitempty"`
	EmployeeIDs []uuid.UUID               `json:"employee_ids,omitempty"`
	Contracts   []RequestContractResponse `json:"contracts,omitempty"`
}

type ListRequestsResponse struct {
	Items []*RequestResponse `json:"items"`
}
