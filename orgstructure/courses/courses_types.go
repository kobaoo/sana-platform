package courses

import (
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID          uuid.UUID   `json:"id"`
	ClientID    uuid.UUID   `json:"client_id"`
	Title       string      `json:"title"`
	CategoryIDs []uuid.UUID `json:"category_ids"`
	Description *string     `json:"description,omitempty"`
	Lecturer    *string     `json:"lecturer,omitempty"`
	ScormURL    string      `json:"scorm_url"`
	IsActive    bool        `json:"is_active"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type CreateCourseRequest struct {
	Title       string       `json:"title"`
	CategoryIDs []uuid.UUID  `json:"category_ids"`
	Description *string      `json:"description,omitempty"`
	Lecturer    *string      `json:"lecturer,omitempty"`
	ScormURL    string       `json:"scorm_url"`
}

type UpdateCourseRequest struct {
	Title       *string      `json:"title,omitempty"`
	CategoryIDs *[]uuid.UUID `json:"category_ids,omitempty"`
	Description *string      `json:"description,omitempty"`
	Lecturer    *string      `json:"lecturer,omitempty"`
	ScormURL    *string      `json:"scorm_url,omitempty"`
	IsActive    *bool        `json:"is_active,omitempty"`
}

type GetCourseResponse struct {
	Course *Course `json:"course"`
}

type ListCoursesResponse struct {
	Courses []*Course `json:"courses"`
}

type UploadSCORMRequest struct {
	FileName string `json:"file_name"`
	FileData []byte `json:"file_data"`
}

type UploadSCORMResponse struct {
	FileName string `json:"file_name"`
	FileSize int    `json:"file_size"`
	ScormURL string `json:"scorm_url"`
	IsValid  bool   `json:"is_valid"`
	Message  string `json:"message"`
}