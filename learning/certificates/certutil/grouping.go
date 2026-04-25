package certutil

import "time"

// Certificate — минимальная модель для чистых функций.
// Не импортирует Encore — намеренно.
type Certificate struct {
	ID         string
	EmployeeID string
	DzoID      string
	Title      string
	IssuedDate time.Time
	ExpiryDate *time.Time
}

// GroupByEmployee группирует сертификаты по EmployeeID.
func GroupByEmployee(certs []Certificate) map[string][]Certificate {
	result := make(map[string][]Certificate, len(certs))
	for _, cert := range certs {
		result[cert.EmployeeID] = append(result[cert.EmployeeID], cert)
	}
	return result
}

// GroupByDzo группирует сертификаты по DzoID.
func GroupByDzo(certs []Certificate) map[string][]Certificate {
	result := make(map[string][]Certificate, len(certs))
	for _, cert := range certs {
		result[cert.DzoID] = append(result[cert.DzoID], cert)
	}
	return result
}
