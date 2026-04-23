package certificates

import (
	"time"

	"github.com/google/uuid"
)

// Certificate is the domain model representing a row in the certificates table.
type Certificate struct {
	ID         string     `json:"id"`
	EmployeeID string     `json:"employee_id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	FileURL    *string    `json:"file_url"`
	IssuedDate time.Time  `json:"issued_date"`
	ExpiryDate *time.Time `json:"expiry_date"`
	UploadedBy *string    `json:"uploaded_by,omitempty"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	IsActive   bool       `json:"is_active"`
}

// CreateRequest is the request body for creating a new certificate.
type CreateRequest struct {
	EmployeeID uuid.UUID  `json:"employee_id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	FileURL    *string    `json:"file_url"`
	IssuedDate time.Time  `json:"issued_date"`
	ExpiryDate *time.Time `json:"expiry_date"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
}

// GetCertResponse is the response for fetching a single certificate.
type GetCertResponse struct {
	Certificate Certificate `json:"certificate"`
}

// ListResponse is the response for listing certificates.
type ListResponse struct {
	Certificates []Certificate `json:"certificates"`
	Total        int           `json:"total"`
}

// UpdateRequest is the request body for updating a certificate.
type UpdateRequest struct {
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	FileURL    *string    `json:"file_url"`
	IssuedDate time.Time  `json:"issued_date"`
	ExpiryDate *time.Time `json:"expiry_date"`
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
}

// ListParams holds optional query filters for listing certificates.
type ListParams struct {
	EmployeeID string `query:"employee_id"`
	EntityType string `query:"entity_type"`
}

// ExpiringParams holds query parameters for listing expiring certificates.
type ExpiringParams struct {
	DzoID string `query:"dzo_id"`
	Days  int    `query:"days"`
}

// DeleteResponse is the response for deleting a certificate.
type DeleteResponse struct {
	Message string `json:"message"`
}
