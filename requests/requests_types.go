package requests

import (
	"time"

	"github.com/google/uuid"
)

type CreateRequestRequest struct {
	EntityID   uuid.UUID `json:"entity_id"`
	EntityType string    `json:"entity_type"`
}

type UpdateRequestStepRequest struct {
	Step int `json:"step"`
}

type UpdateRequestStatusRequest struct {
	Status string `json:"status"`
}

type RequestResponse struct {
	ID          uuid.UUID `json:"id"`
	InitiatorID uuid.UUID `json:"initiator_id"`
	EntityID    uuid.UUID `json:"entity_id"`
	EntityType  string    `json:"entity_type"`
	Step        int       `json:"step"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListRequestsResponse struct {
	Items []*RequestResponse `json:"items"`
}

type CreateCertificateRenewalRequest struct {
	EntityID uuid.UUID `json:"entity_id"`
}

type PatchCertificateRenewalStatusRequest struct {
	Status string `json:"status"`
}

type ListCertificateRenewalsParams struct {
	InitiatorID string `query:"initiator_id"`
}
