package notifications

// NotificationType — тип уведомления.
type NotificationType string

const (
	TypeCertExpiring NotificationType = "CERT_EXPIRING"
	TypeCertExpired  NotificationType = "CERT_EXPIRED"
)

// EntityType — доменная сущность, к которой относится уведомление.
type EntityType string

const (
	EntityCertificate EntityType = "CERTIFICATE"
)

// NotificationStatus — статус доставки уведомления.
type NotificationStatus string

const (
	StatusPending NotificationStatus = "PENDING"
	StatusSent    NotificationStatus = "SENT"
	StatusFailed  NotificationStatus = "FAILED"
)

// NotifyRequest — входные данные для создания уведомления.
// Используется как тип запроса Encore private API.
type NotifyRequest struct {
	UserID     string           `json:"user_id"`
	Type       NotificationType `json:"type"`
	EntityType EntityType       `json:"entity_type"`
	EntityID   string           `json:"entity_id"`
}

// NotifyResponse — результат обработки уведомления.
type NotifyResponse struct {
	Skipped bool   `json:"skipped"` // true — дубликат, пропущено
	Message string `json:"message"`
}
