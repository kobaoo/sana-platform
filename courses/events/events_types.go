package events

import "time"

// EventStatus represents the lifecycle status of an event.
type EventStatus string

const (
	StatusActive    EventStatus = "ACTIVE"
	StatusCompleted EventStatus = "COMPLETED"
	StatusCancelled EventStatus = "CANCELLED"
)

func (s EventStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

// Event is the domain model representing a row in the events table.
type Event struct {
	ID           string      `json:"id"`
	ClientID     string      `json:"client_id"`
	HostID       string      `json:"host_id"`
	Title        string      `json:"title"`
	Description  *string     `json:"description"`
	ZoomLink     *string     `json:"zoom_link"`
	EventDate    time.Time   `json:"event_date"`
	MaterialsURL *string     `json:"materials_url"`
	Status       EventStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// CreateEventRequest is the request body for creating a new event.
// Newly created events are always saved with status=ACTIVE
// (immediately visible to employees of the same company).
type CreateEventRequest struct {
	Title       string    `json:"title"`
	EventDate   time.Time `json:"event_date"`
	HostID      string    `json:"host_id"`
	ZoomLink    *string   `json:"zoom_link,omitempty"`
	Description *string   `json:"description,omitempty"`
}

// UpdateEventRequest is the request body for partially updating an event.
type UpdateEventRequest struct {
	Title       *string      `json:"title,omitempty"`
	EventDate   *time.Time   `json:"event_date,omitempty"`
	HostID      *string      `json:"host_id,omitempty"`
	ZoomLink    *string      `json:"zoom_link,omitempty"`
	Description *string      `json:"description,omitempty"`
	Status      *EventStatus `json:"status,omitempty"`
}

// SetMaterialsRequest is the request body for attaching a materials link
// (Google Drive / OneDrive / Dropbox folder URL).
type SetMaterialsRequest struct {
	MaterialsURL string `json:"materials_url"`
}

// GetEventResponse is the response for fetching a single event.
type GetEventResponse struct {
	Event Event `json:"event"`
}

// ListEventsResponse is the response for listing events.
type ListEventsResponse struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
}

// DeleteEventResponse is the response for deleting an event.
type DeleteEventResponse struct {
	Message string `json:"message"`
}
