package notifications

import (
	"context"
	"fmt"
	"time"
)

// notificationExists проверяет анти-дублирующий ключ:
// (user_id, type, entity_type, entity_id).
// Возвращает true если уведомление уже было отправлено.
func notificationExists(ctx context.Context, req *NotifyRequest) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications
		 WHERE user_id = $1 AND type = $2 AND entity_type = $3 AND entity_id = $4`,
		req.UserID, string(req.Type), string(req.EntityType), req.EntityID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("notificationExists: %w", err)
	}
	return count > 0, nil
}

// saveNotification сохраняет запись уведомления со статусом SENT.
// ON CONFLICT DO NOTHING — дополнительная страховка от гонок.
func saveNotification(ctx context.Context, req *NotifyRequest) error {
	now := time.Now()
	_, err := db.Exec(ctx,
		`INSERT INTO notifications
		   (user_id, type, entity_type, entity_id, status, sent_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, type, entity_type, entity_id) DO NOTHING`,
		req.UserID, string(req.Type), string(req.EntityType), req.EntityID,
		string(StatusSent), now, now,
	)
	if err != nil {
		return fmt.Errorf("saveNotification: %w", err)
	}
	return nil
}

// saveFailedNotification сохраняет запись со статусом FAILED для аудита.
func saveFailedNotification(ctx context.Context, req *NotifyRequest) error {
	now := time.Now()
	_, err := db.Exec(ctx,
		`INSERT INTO notifications
		   (user_id, type, entity_type, entity_id, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, type, entity_type, entity_id) DO NOTHING`,
		req.UserID, string(req.Type), string(req.EntityType), req.EntityID,
		string(StatusFailed), now,
	)
	if err != nil {
		return fmt.Errorf("saveFailedNotification: %w", err)
	}
	return nil
}
