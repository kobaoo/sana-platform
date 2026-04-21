package certificates

import (
	"context"
	"log"
	"time"

	"encore.app/learning/certificates/certutil"
	"encore.dev/cron"
)

var _ = cron.NewJob("check-expiring-certs", cron.JobConfig{
	Title:    "Check Expiring Certificates",
	Schedule: "0 5 * * 1",
	Endpoint: CheckExpiringCertificates,
})

//encore:api private
func CheckExpiringCertificates(ctx context.Context) error {
	threshold := time.Now().AddDate(0, 6, 0)
	log.Printf("[cron] checking certificates expiring before %s", threshold.Format("2006-01-02"))

	// TODO: uncomment after notifications merge
	// certs, err := queryExpiringCerts(ctx)
	// if err != nil {
	// 	return err
	// }
	// grouped := certutil.GroupByEmployee(toCertutil(certs))
	// for employeeID, empCerts := range grouped { ... }

	certs := []certutil.Certificate{}

	if len(certs) == 0 {
		log.Printf("[cron] no expiring certificates found")
		return nil
	}

	grouped := certutil.GroupByEmployee(certs)
	log.Printf("[cron] found %d employee(s) with expiring certificates", len(grouped))

	return nil
}
