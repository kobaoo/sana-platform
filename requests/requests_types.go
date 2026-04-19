package requests

import (
	"time"

	"github.com/google/uuid"
)

type CreateRequestRequest struct {
	InitiatorID uuid.UUID `json:"initiator_id"`
	EntityID    uuid.UUID `json:"entity_id"`
	EntityType  string    `json:"entity_type"`
}

type UpdateRequestStepRequest struct {
	ActorID uuid.UUID `json:"actor_id"`
	Step    int       `json:"step"`
}

type UpdateRequestStatusRequest struct {
	ActorID uuid.UUID `json:"actor_id"`
	Status  string    `json:"status"`
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
