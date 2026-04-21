package notifications

// NotificationType — тип уведомления.
type NotificationType string

const (
	// Certificates
	TypeCertExpiring NotificationType = "CERT_EXPIRING"
	TypeCertExpired  NotificationType = "CERT_EXPIRED"

	// Requests
	TypeRequestCreated     NotificationType = "REQUEST_CREATED"
	TypeRequestStepUpdated NotificationType = "REQUEST_STEP_UPDATED"
	TypeRequestApproved    NotificationType = "REQUEST_APPROVED"
	TypeRequestCancelled   NotificationType = "REQUEST_CANCELLED"
)

// EntityType — доменная сущность, к которой относится уведомление.
type EntityType string

const (
	EntityCertificate EntityType = "CERTIFICATE"
	EntityRequest     EntityType = "REQUEST"
)

// NotificationStatus — статус доставки уведомления.
type NotificationStatus string

const (
	StatusPending NotificationStatus = "PENDING"
	StatusSent    NotificationStatus = "SENT"
	StatusFailed  NotificationStatus = "FAILED"
)

// NotifyRequest — входные данные для создания уведомления.
type NotifyRequest struct {
	UserID     string           `json:"user_id"`
	Type       NotificationType `json:"type"`
	EntityType EntityType       `json:"entity_type"`
	EntityID   string           `json:"entity_id"`
}

// NotifyResponse — результат обработки уведомления.
type NotifyResponse struct {
	Skipped bool   `json:"skipped"`
	Message string `json:"message"`
}
