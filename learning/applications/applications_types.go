package applications

import (
	"time"

	"encore.app/auth/authhandler"
)

type ApplicationKind string

const (
	ApplicationKindRegular  ApplicationKind = "regular"
	ApplicationKindClosed   ApplicationKind = "closed"
	ApplicationKindArchived ApplicationKind = "archived"
)

func (k ApplicationKind) IsValid() bool {
	switch k {
	case "", ApplicationKindRegular, ApplicationKindClosed, ApplicationKindArchived:
		return true
	}
	return false
}

type ApplicationStatus string

const (
	ApplicationStatusDraft     ApplicationStatus = "draft"
	ApplicationStatusSubmitted ApplicationStatus = "submitted"
	ApplicationStatusInProcess ApplicationStatus = "in_process"
	ApplicationStatusCompleted ApplicationStatus = "completed"
	ApplicationStatusCancelled ApplicationStatus = "cancelled"
)

func (s ApplicationStatus) IsValid() bool {
	switch s {
	case ApplicationStatusDraft, ApplicationStatusSubmitted, ApplicationStatusInProcess, ApplicationStatusCompleted, ApplicationStatusCancelled:
		return true
	}
	return false
}

type Caller struct {
	UserID string
	Role   authhandler.UserRole
	DzoID  *string
}

type Application struct {
	ID                  string            `json:"id"`
	Kind                ApplicationKind   `json:"kind"`
	Status              ApplicationStatus `json:"status"`
	DzoID               *string           `json:"dzo_id,omitempty"`
	CreatedByUserID     *string           `json:"created_by_user_id,omitempty"`
	CourseID            *string           `json:"course_id,omitempty"`
	RequestedCourseName string            `json:"requested_course_name"`
	ExpenseCategory     *string           `json:"expense_category,omitempty"`
	Comment             *string           `json:"comment,omitempty"`
	EmployeeIDs         []string          `json:"employee_ids"`
	IsActive            bool              `json:"is_active"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type CreateApplicationRequest struct {
	Kind                ApplicationKind `json:"kind,omitempty"`
	CourseID            *string         `json:"course_id,omitempty"`
	RequestedCourseName string          `json:"requested_course_name"`
	ExpenseCategory     *string         `json:"expense_category,omitempty"`
	Comment             *string         `json:"comment,omitempty"`
	EmployeeIDs         []string        `json:"employee_ids,omitempty"`
}

type UpdateApplicationRequest struct {
	CourseID            *string   `json:"course_id,omitempty"`
	RequestedCourseName *string   `json:"requested_course_name,omitempty"`
	ExpenseCategory     *string   `json:"expense_category,omitempty"`
	Comment             *string   `json:"comment,omitempty"`
	EmployeeIDs         *[]string `json:"employee_ids,omitempty"`
}

type GetApplicationResponse struct {
	Application Application `json:"application"`
}

type ListApplicationsResponse struct {
	Applications []Application `json:"applications"`
	Total        int           `json:"total"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
