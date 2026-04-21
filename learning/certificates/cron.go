package certificates

import (
	"context"
	"log"

	"encore.app/learning/certificates/certutil"
	"encore.app/notifications"
	"encore.dev/cron"
)

var _ = cron.NewJob("check-expiring-certs", cron.JobConfig{
	Title:    "Check Expiring Certificates",
	Schedule: "0 5 * * 1",
	Endpoint: CheckExpiringCertificates,
})

//encore:api private
func CheckExpiringCertificates(ctx context.Context) error {
	log.Printf("[cron] checking certificates expiring within 6 months")

	certs, err := queryExpiringCerts(ctx)
	if err != nil {
		return err
	}

	if len(certs) == 0 {
		log.Printf("[cron] no expiring certificates found")
		return nil
	}

	grouped := certutil.GroupByEmployee(toCertutil(certs))
	log.Printf("[cron] found %d employee(s) with expiring certificates", len(grouped))

	for employeeID, empCerts := range grouped {
		for _, cert := range empCerts {
			event := notifications.CertificateExpiryEvent{
				EmployeeID: employeeID,
				CertID:     cert.ID,
				Title:      cert.Title,
			}
			if cert.ExpiryDate != nil {
				event.ExpiryDate = cert.ExpiryDate.Format("2006-01-02")
			}

			req := event.ToNotifyRequest()
			resp, err := notifications.NotifyUserWithEntity(ctx, &req)
			if err != nil {
				log.Printf("[cron] failed to notify user=%s cert=%s: %v", employeeID, cert.ID, err)
				continue // не прерываем — уведомляем остальных
			}
			if resp.Skipped {
				log.Printf("[cron] skipped (duplicate) user=%s cert=%s", employeeID, cert.ID)
			} else {
				log.Printf("[cron] notified user=%s cert=%s", employeeID, cert.ID)
			}
		}
	}

	return nil
}

// toCertutil конвертирует локальный тип Certificate в certutil.Certificate
// для работы с чистыми утилитами без зависимости на Encore.
func toCertutil(certs []Certificate) []certutil.Certificate {
	out := make([]certutil.Certificate, len(certs))
	for i, c := range certs {
		out[i] = certutil.Certificate{
			ID:         c.ID,
			EmployeeID: c.EmployeeID,
			Title:      c.Title,
			IssuedDate: c.IssuedDate,
			ExpiryDate: c.ExpiryDate,
		}
	}
	return out
}
