package requests

import "time"

type RequestType string

const (
	RequestTypeMain       RequestType = "MAIN"
	RequestTypeSubrequest RequestType = "SUBREQUEST"
)

func (t RequestType) IsValid() bool {
	switch t {
	case RequestTypeMain, RequestTypeSubrequest:
		return true
	default:
		return false
	}
}

type RequestStatus string

const (
	RequestStatusDraft      RequestStatus = "DRAFT"
	RequestStatusInProgress RequestStatus = "IN_PROGRESS"
	RequestStatusPending    RequestStatus = "PENDING"
	RequestStatusApproved   RequestStatus = "APPROVED"
	RequestStatusRejected   RequestStatus = "REJECTED"
)

func (s RequestStatus) IsValid() bool {
	switch s {
	case RequestStatusDraft, RequestStatusInProgress, RequestStatusPending, RequestStatusApproved, RequestStatusRejected:
		return true
	default:
		return false
	}
}

type CostMode string

const (
	CostModePerEmployee CostMode = "PER_EMPLOYEE"
	CostModeGroup       CostMode = "GROUP"
)

func (m CostMode) IsValid() bool {
	switch m {
	case CostModePerEmployee, CostModeGroup:
		return true
	default:
		return false
	}
}

type CreateAdminRequestRequest struct {
	TrainingEventID string    `json:"training_event_id"`
	Title           *string   `json:"title,omitempty"`
	Category        *string   `json:"category,omitempty"`
	Format          *string   `json:"format,omitempty"`
	EmployeeIDs     []string  `json:"employee_ids,omitempty"`
	DzoIDs          []string  `json:"dzo_ids,omitempty"`
	CostAmount      *float64  `json:"cost_amount,omitempty"`
	CostMode        *CostMode `json:"cost_mode,omitempty"`
	DeadlineAt      *string   `json:"deadline_at,omitempty"`
}

type UpdateHRRequestEmployeesRequest struct {
	EmployeeIDs []string `json:"employee_ids"`
}

type RequestEmployee struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	DzoID    string `json:"dzo_id"`
	DzoName  string `json:"dzo_name"`
}

type RequestTargetDZO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RequestSummary struct {
	ID                 string        `json:"id"`
	InitiatorID        string        `json:"initiator_id"`
	ParentRequestID    *string       `json:"parent_request_id,omitempty"`
	TrainingEventID    string        `json:"training_event_id"`
	EntityType         string        `json:"entity_type"`
	RequestType        RequestType   `json:"request_type"`
	Status             RequestStatus `json:"status"`
	AssignedHRID       *string       `json:"assigned_hr_id,omitempty"`
	TargetDzoID        *string       `json:"target_dzo_id,omitempty"`
	Title              string        `json:"title"`
	Category           *string       `json:"category,omitempty"`
	Format             *string       `json:"format,omitempty"`
	ResponsibleAdminID *string       `json:"responsible_admin_id,omitempty"`
	TrainingDate       *time.Time    `json:"training_date,omitempty"`
	DeadlineAt         *time.Time    `json:"deadline_at,omitempty"`
	CostAmount         *float64      `json:"cost_amount,omitempty"`
	CostMode           *CostMode     `json:"cost_mode,omitempty"`
	EmployeesCount     int           `json:"employees_count"`
	ApprovedChildren   int           `json:"approved_children"`
	TotalChildren      int           `json:"total_children"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type RequestDetail struct {
	Request       RequestSummary     `json:"request"`
	Employees     []RequestEmployee  `json:"employees"`
	TargetDZOs    []RequestTargetDZO `json:"target_dzos"`
	ChildRequests []RequestSummary   `json:"child_requests"`
}

type GetRequestResponse struct {
	Detail RequestDetail `json:"detail"`
}

type ListRequestsResponse struct {
	Items []RequestSummary `json:"items"`
}

type BudgetHistoryItem struct {
	OperationType string    `json:"operation_type"`
	Amount        float64   `json:"amount"`
	CreatedBy     string    `json:"created_by"`
	Reason        *string   `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
}

type GetRequestBudgetHistoryResponse struct {
	Items []BudgetHistoryItem `json:"items"`
}
