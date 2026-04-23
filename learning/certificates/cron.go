package certificates

import (
	"context"
	"log"

	entemployee "encore.app/db/ent/employee"
	entuser "encore.app/db/ent/user"
	"encore.app/learning/certificates/certutil"
	"encore.dev/cron"
	"github.com/google/uuid"
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

	log.Printf("[cron] found %d expiring certificate(s)", len(certs))

	dzoMap, err := queryEmployeeDzoMap(ctx, certs)
	if err != nil {
		return err
	}

	emps, err := queryEmployeeInfos(ctx, certs)
	if err != nil {
		return err
	}

	grouped := certutil.GroupByDzo(toCertutil(certs, dzoMap))
	log.Printf("[cron] grouped into %d DZO(s)", len(grouped))

	for dzoID, dzoCerts := range grouped {
		log.Printf("[cron] DZO %s: %d certificate(s) expiring", dzoID, len(dzoCerts))

		hrEmails, err := queryHREmailsByDzo(ctx, dzoID)
		if err != nil {
			log.Printf("[cron] failed to query HR for DZO %s: %v", dzoID, err)
			continue
		}
		if len(hrEmails) == 0 {
			log.Printf("[cron] no active HR found for DZO %s, skipping", dzoID)
			continue
		}

		for _, email := range hrEmails {
			if err := sendExpiryEmail(email, dzoID, dzoCerts, emps); err != nil {
				log.Printf("[cron] failed to send email to %s (DZO %s): %v", email, dzoID, err)
			} else {
				log.Printf("[cron] email sent to HR %s for DZO %s (%d certs)", email, dzoID, len(dzoCerts))
			}
		}
	}

	return nil
}

func queryHREmailsByDzo(ctx context.Context, dzoID string) ([]string, error) {
	dzoUID, err := uuid.Parse(dzoID)
	if err != nil {
		return nil, err
	}

	rows, err := Client.User.Query().
		Where(
			entuser.RoleEQ("HR"),
			entuser.DzoIDEQ(dzoUID),
			entuser.IsActiveEQ(true),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	emails := make([]string, 0, len(rows))
	for _, r := range rows {
		emails = append(emails, r.Email)
	}
	return emails, nil
}

func queryEmployeeInfos(ctx context.Context, certs []Certificate) (map[string]empInfo, error) {
	seen := make(map[uuid.UUID]struct{}, len(certs))
	for _, c := range certs {
		if uid, err := uuid.Parse(c.EmployeeID); err == nil {
			seen[uid] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return map[string]empInfo{}, nil
	}

	ids := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}

	rows, err := Client.Employee.Query().
		Where(entemployee.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	m := make(map[string]empInfo, len(rows))
	for _, r := range rows {
		m[r.ID.String()] = empInfo{Name: r.FullName, Email: r.Email}
	}
	return m, nil
}

func toCertutil(certs []Certificate, dzoMap map[string]string) []certutil.Certificate {
	out := make([]certutil.Certificate, len(certs))
	for i, c := range certs {
		out[i] = certutil.Certificate{
			ID:         c.ID,
			EmployeeID: c.EmployeeID,
			DzoID:      dzoMap[c.EmployeeID],
			Title:      c.Title,
			IssuedDate: c.IssuedDate,
			ExpiryDate: c.ExpiryDate,
		}
	}
	return out
}
