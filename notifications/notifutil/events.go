package notifutil

// NotificationType — тип уведомления.
type NotificationType string

const (
	TypeCertExpiring       NotificationType = "CERT_EXPIRING"
	TypeCertExpired        NotificationType = "CERT_EXPIRED"
	TypeRequestCreated     NotificationType = "REQUEST_CREATED"
	TypeRequestStepUpdated NotificationType = "REQUEST_STEP_UPDATED"
	TypeRequestApproved    NotificationType = "REQUEST_APPROVED"
	TypeRequestCancelled   NotificationType = "REQUEST_CANCELLED"
)

// EntityType — доменная сущность.
type EntityType string

const (
	EntityCertificate EntityType = "CERTIFICATE"
	EntityRequest     EntityType = "REQUEST"
)

// NotifyRequest — универсальный запрос на уведомление.
type NotifyRequest struct {
	UserID     string
	Type       NotificationType
	EntityType EntityType
	EntityID   string
}

// ════ CERTIFICATE EVENTS ════

type CertificateExpiryEvent struct {
	EmployeeID string
	CertID     string
	Title      string
	ExpiryDate string
}

func (e CertificateExpiryEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.EmployeeID,
		Type:       TypeCertExpiring,
		EntityType: EntityCertificate,
		EntityID:   e.CertID,
	}
}

// ════ REQUEST EVENTS ════

type RequestCreatedEvent struct {
	InitiatorID string
	ApproverID  string
	RequestID   string
}

func (e RequestCreatedEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.ApproverID,
		Type:       TypeRequestCreated,
		EntityType: EntityRequest,
		EntityID:   e.RequestID,
	}
}

type RequestStepUpdatedEvent struct {
	ApproverID string
	RequestID  string
	Step       int
}

func (e RequestStepUpdatedEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.ApproverID,
		Type:       TypeRequestStepUpdated,
		EntityType: EntityRequest,
		EntityID:   e.RequestID,
	}
}

type RequestApprovedEvent struct {
	InitiatorID string
	RequestID   string
}

func (e RequestApprovedEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.InitiatorID,
		Type:       TypeRequestApproved,
		EntityType: EntityRequest,
		EntityID:   e.RequestID,
	}
}

type RequestCancelledEvent struct {
	InitiatorID string
	RequestID   string
}

func (e RequestCancelledEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.InitiatorID,
		Type:       TypeRequestCancelled,
		EntityType: EntityRequest,
		EntityID:   e.RequestID,
	}
}
