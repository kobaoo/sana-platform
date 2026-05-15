package notifications

// ════ CERTIFICATE EVENTS ════

// CertificateExpiryEvent — сертификат приближается к дате истечения.
type CertificateExpiryEvent struct {
	EmployeeID string
	CertID     string
	Title      string
	ExpiryDate string // формат "2006-01-02"
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

// RequestCreatedEvent — новый запрос создан, уведомить следующего апрувера.
type RequestCreatedEvent struct {
	InitiatorID string
	ApproverID  string // первый апрувер (HR на шаге 0→1)
	RequestID   string
	EntityType  string
}

func (e RequestCreatedEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.ApproverID,
		Type:       TypeRequestCreated,
		EntityType: EntityRequest,
		EntityID:   e.RequestID,
	}
}

// RequestStepUpdatedEvent — шаг изменился, уведомить следующего апрувера.
type RequestStepUpdatedEvent struct {
	ApproverID string // следующий апрувер
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

// RequestApprovedEvent — запрос одобрен, уведомить инициатора.
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

// RequestCancelledEvent — запрос отменён, уведомить инициатора.
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
