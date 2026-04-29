package positions

import "time"

// GeneralPosition represents a global/common position title.
type GeneralPosition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsDeleted   bool      `json:"is_deleted"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateGeneralPositionRequest is the request body for creating a general position.
type CreateGeneralPositionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type ListGeneralPositionsRequest struct {
	Search string `query:"search"`
}

// UpdateGeneralPositionRequest is the request body for partially updating a general position.
type UpdateGeneralPositionRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// GetGeneralPositionResponse is the response for fetching a single general position.
type GetGeneralPositionResponse struct {
	GeneralPosition GeneralPosition `json:"general_position"`
}

// ListGeneralPositionsResponse is the response for listing general positions.
type ListGeneralPositionsResponse struct {
	GeneralPositions []GeneralPosition `json:"general_positions"`
	Total            int               `json:"total"`
}

// DeleteGeneralPositionResponse is the response for deleting a general position.
type DeleteGeneralPositionResponse struct {
	Message string `json:"message"`
}
