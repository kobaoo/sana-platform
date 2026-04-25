package certificates

import (
	"context"
	"log"
	"time"

	entemployee "encore.app/db/ent/employee"
	entuser "encore.app/db/ent/user"
	"encore.app/learning/certificates/certutil"
	"encore.app/notifications"
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
			if err := sendExpiryEmailWithRetry(email, dzoID, dzoCerts, emps); err != nil {
				log.Printf("[cron] failed to send email to %s (DZO %s) after retries: %v", email, dzoID, err)
			} else {
				log.Printf("[cron] email sent to HR %s for DZO %s (%d certs)", email, dzoID, len(dzoCerts))
			}
		}
	}

	// Write in-app CERT_EXPIRING notifications for each employee.
	keycloakMap, err := queryEmployeeKeycloakMap(ctx, certs)
	if err != nil {
		log.Printf("[cron] failed to build employee→keycloak map: %v", err)
		// non-fatal: HR emails already sent
		return nil
	}

	for _, cert := range certs {
		keycloakUserID, ok := keycloakMap[cert.EmployeeID]
		if !ok {
			continue // employee has no linked user account
		}
		req := &notifications.NotifyRequest{
			UserID:     keycloakUserID,
			Type:       notifications.TypeCertExpiring,
			EntityType: notifications.EntityCertificate,
			EntityID:   cert.ID,
		}
		if _, notifyErr := notifications.NotifyUserWithEntity(ctx, req); notifyErr != nil {
			log.Printf("[cron] failed to notify employee for cert %s: %v", cert.ID, notifyErr)
		}
	}

	return nil
}

// sendExpiryEmailWithRetry retries SMTP up to 3 times with a short back-off.
func sendExpiryEmailWithRetry(hrEmail, dzoID string, certs []certutil.Certificate, emps map[string]empInfo) error {
	const maxAttempts = 3
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i*5) * time.Second)
		}
		if err := sendExpiryEmail(hrEmail, dzoID, certs, emps); err == nil {
			return nil
		} else {
			lastErr = err
			log.Printf("[cron] SMTP attempt %d/%d failed for %s: %v", i+1, maxAttempts, hrEmail, err)
		}
	}
	return lastErr
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

// queryEmployeeKeycloakMap returns a map of employeeID → keycloak_user_id.
// Primary lookup: employees.user_id FK → users.keycloak_user_id.
// Fallback: for employees where user_id IS NULL, match by email.
func queryEmployeeKeycloakMap(ctx context.Context, certs []Certificate) (map[string]string, error) {
	seen := make(map[uuid.UUID]struct{}, len(certs))
	for _, c := range certs {
		if uid, err := uuid.Parse(c.EmployeeID); err == nil {
			seen[uid] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return map[string]string{}, nil
	}

	empIDs := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		empIDs = append(empIDs, id)
	}

	empRows, err := Client.Employee.Query().
		Where(entemployee.IDIn(empIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Partition employees: those with user_id FK and those needing email fallback.
	var linkedUserIDs []uuid.UUID
	empToUserID := make(map[string]uuid.UUID)
	var fallbackEmails []string
	empByEmail := make(map[string]string) // email → empID

	for _, e := range empRows {
		if e.UserID != nil {
			linkedUserIDs = append(linkedUserIDs, *e.UserID)
			empToUserID[e.ID.String()] = *e.UserID
		} else if e.Email != "" {
			fallbackEmails = append(fallbackEmails, e.Email)
			empByEmail[e.Email] = e.ID.String()
		}
	}

	result := make(map[string]string)

	// Path 1: user_id FK.
	if len(linkedUserIDs) > 0 {
		userRows, err := Client.User.Query().
			Where(entuser.IDIn(linkedUserIDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		userIDToKC := make(map[string]string, len(userRows))
		for _, u := range userRows {
			userIDToKC[u.ID.String()] = u.KeycloakUserID
		}
		for empID, userID := range empToUserID {
			if kc, ok := userIDToKC[userID.String()]; ok {
				result[empID] = kc
			}
		}
	}

	// Path 2: email fallback for employees where user_id IS NULL.
	if len(fallbackEmails) > 0 {
		userRows, err := Client.User.Query().
			Where(entuser.EmailIn(fallbackEmails...), entuser.IsActiveEQ(true)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, u := range userRows {
			if empID, ok := empByEmail[u.Email]; ok {
				result[empID] = u.KeycloakUserID
			}
		}
	}

	return result, nil
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
