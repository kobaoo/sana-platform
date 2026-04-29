package positions

import "time"

// DzoPositionTitle represents a local DZO-specific position title.
type DzoPositionTitle struct {
	ID                  string    `json:"id"`
	DzoID               string    `json:"dzo_id"`
	ClientID            string    `json:"client_id"`
	GeneralPositionID   *string   `json:"general_position_id"`
	GeneralPositionName *string   `json:"general_position_name"`
	DzoName             *string   `json:"dzo_name"`
	LocalTitle          string    `json:"local_title"`
	IsActive            bool      `json:"is_active"`
	IsDeleted           bool      `json:"is_deleted"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	EmployeesCount      int       `json:"employees_count"`
}

// CreateDzoPositionTitleRequest is the request body for creating a DZO position title.
type CreateDzoPositionTitleRequest struct {
	DzoID             string  `json:"dzo_id"`
	GeneralPositionID *string `json:"general_position_id,omitempty"`
	LocalTitle        string  `json:"local_title"`
}

// UpdateDzoPositionTitleRequest is the request body for partially updating a DZO position title.
type UpdateDzoPositionTitleRequest struct {
	GeneralPositionID *string `json:"general_position_id,omitempty"`
	LocalTitle        *string `json:"local_title,omitempty"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

type ListDzoPositionTitlesRequest struct {
	Search            string `query:"search"`
	DzoID             string `query:"dzo_id"`
	GeneralPositionID string `query:"general_position_id"`
}

// GetDzoPositionTitleResponse is the response for fetching a single DZO position title.
type GetDzoPositionTitleResponse struct {
	PositionTitle DzoPositionTitle `json:"position_title"`
}

// ListDzoPositionTitlesResponse is the response for listing DZO position titles.
type ListDzoPositionTitlesResponse struct {
	PositionTitles []DzoPositionTitle `json:"position_titles"`
	Total          int                `json:"total"`
}

// DeleteDzoPositionTitleResponse is the response for deleting a DZO position title.
type DeleteDzoPositionTitleResponse struct {
	Message        string `json:"message"`
	EmployeesCount int    `json:"employees_count"`
}
