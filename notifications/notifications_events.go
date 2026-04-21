package notifications

// CertificateExpiryEvent — событие из домена certificates.
// Генерируется когда сертификат приближается к дате истечения.
//
// Будущие события (добавить по мере роста):
//   - RequestSubmittedEvent
//   - CourseCompletedEvent
type CertificateExpiryEvent struct {
	EmployeeID string
	CertID     string
	Title      string
	ExpiryDate string // формат "2006-01-02"
}

// ToNotifyRequest конвертирует доменное событие в универсальный NotifyRequest.
func (e CertificateExpiryEvent) ToNotifyRequest() NotifyRequest {
	return NotifyRequest{
		UserID:     e.EmployeeID,
		Type:       TypeCertExpiring,
		EntityType: EntityCertificate,
		EntityID:   e.CertID,
	}
}
