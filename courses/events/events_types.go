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
	ID              string      `json:"id"`
	ClientID        string      `json:"client_id"`
	HostID          string      `json:"host_id"`
	HostName        string      `json:"host_name"`
	Title           string      `json:"title"`
	Description     *string     `json:"description"`
	ZoomLink        string      `json:"zoom_link"`
	EventDate       time.Time   `json:"event_date"`
	MaxParticipants int         `json:"max_participants"`
	MaterialsURL    *string     `json:"materials_url"`
	Status          EventStatus `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`

	// Aggregated/expanded fields populated on detail and list responses.
	ParticipantsCount int           `json:"participants_count"`
	AvailableSlots    int           `json:"available_slots"`
	Participants      []Participant `json:"participants,omitempty"`

	// Per-caller fields. Populated for EMP role only — for SA/ADM/HR the
	// concept of self-enrollment doesn't apply and these stay zero/empty.
	IsRegistered bool `json:"is_registered,omitempty"`
}

// Participant is an employee enrolled into an event.
type Participant struct {
	ID               string    `json:"id"`
	EmployeeID       string    `json:"employee_id"`
	FullName         string    `json:"full_name"`
	Email            string    `json:"email"`
	Department       *string   `json:"department,omitempty"`
	Position         *string   `json:"position,omitempty"`
	AttendanceStatus string    `json:"attendance_status"`
	JoinedAt         time.Time `json:"joined_at"`
}

// CreateEventRequest is the request body for creating a new event.
// Newly created events are always saved with status=ACTIVE
// (immediately visible to employees of the same company).
// EmployeeIDs is an optional list of employees the admin pre-enrolls into the event.
// Remaining seats can be taken by other employees via self-enrollment.
type CreateEventRequest struct {
	Title           string    `json:"title"`
	EventDate       time.Time `json:"event_date"`
	HostID          string    `json:"host_id"`
	ZoomLink        string    `json:"zoom_link"`
	MaxParticipants int       `json:"max_participants"`
	Description     *string   `json:"description,omitempty"`
	EmployeeIDs     []string  `json:"employee_ids,omitempty"`
}

// UpdateEventRequest is the request body for updating an event.
// All fields are optional — fields left unset are not changed.
type UpdateEventRequest struct {
	Title           *string      `json:"title,omitempty"`
	EventDate       *time.Time   `json:"event_date,omitempty"`
	HostID          *string      `json:"host_id,omitempty"`
	ZoomLink        *string      `json:"zoom_link,omitempty"`
	MaxParticipants *int         `json:"max_participants,omitempty"`
	Description     *string      `json:"description,omitempty"`
	Status          *EventStatus `json:"status,omitempty"`
	EmployeeIDs     *[]string    `json:"employee_ids,omitempty"`
}

// ListEventsParams are the filters available on GET /events.
// From/To are inclusive date boundaries in RFC3339 or YYYY-MM-DD format.
type ListEventsParams struct {
	From   string `query:"from"`
	To     string `query:"to"`
	Status string `query:"status"`
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

// Host is a potential event host (SA/ADM/HR user).
type Host struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	FullName string `json:"full_name,omitempty"`
}

// ListHostsParams are the query params for GET /events/hosts (lazy-load search).
type ListHostsParams struct {
	Search string `query:"search"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

// ListHostsResponse is the response for GET /events/hosts.
type ListHostsResponse struct {
	Hosts   []Host `json:"hosts"`
	Total   int    `json:"total"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	HasMore bool   `json:"has_more"`
}

// ListAttendanceParams are the query params for GET /events/:id/attendance.
type ListAttendanceParams struct {
	Search string `query:"search"`
}

// ListAttendanceResponse is the response for GET /events/:id/attendance —
// the full participant list with attendance status, optionally filtered
// by full_name substring.
type ListAttendanceResponse struct {
	Participants []Participant `json:"participants"`
	Total        int           `json:"total"`
}

// ListMyRegistrationsParams are the query params for GET /my-event-registrations.
// Filter accepts upcoming|past|all and defaults to upcoming.
// Limit defaults to 4 (sidebar page size); offset defaults to 0.
type ListMyRegistrationsParams struct {
	Filter string `query:"filter"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

// MyRegistration is one row in GET /events/my-registrations — the event the
// caller is enrolled into plus the caller's per-event attendance fields.
type MyRegistration struct {
	Event            Event     `json:"event"`
	AttendanceStatus string    `json:"attendance_status"`
	JoinedAt         time.Time `json:"joined_at"`
}

// ListMyRegistrationsResponse is the response for GET /my-event-registrations.
// Total is the count of all rows matching the filter (before paging).
type ListMyRegistrationsResponse struct {
	Registrations []MyRegistration `json:"registrations"`
	Total         int              `json:"total"`
	Limit         int              `json:"limit"`
	Offset        int              `json:"offset"`
	HasMore       bool             `json:"has_more"`
}

// AttendanceUpdate is one row in the bulk attendance update payload.
// Attended=true → ATTENDED, false → MISSED.
type AttendanceUpdate struct {
	EmployeeID string `json:"employee_id"`
	Attended   bool   `json:"attended"`
}

// UpdateAttendanceRequest is the body for PUT /events/:id/attendance.
type UpdateAttendanceRequest struct {
	Updates []AttendanceUpdate `json:"updates"`
}

// UpdateAttendanceResponse summarises the bulk write.
type UpdateAttendanceResponse struct {
	Updated int `json:"updated"`
}
