package notifications

import (
	"context"
	"log"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
)

//encore:api private
func NotifyUserWithEntity(ctx context.Context, req *NotifyRequest) (*NotifyResponse, error) {
	log.Printf("[notifications] processing: user=%s type=%s entity=%s/%s",
		req.UserID, req.Type, req.EntityType, req.EntityID)

	exists, err := notificationExists(ctx, req)
	if err != nil {
		log.Printf("[notifications] duplicate check error: %v", err)
		return nil, err
	}
	if exists {
		log.Printf("[notifications] skipped (duplicate): user=%s type=%s entity=%s/%s",
			req.UserID, req.Type, req.EntityType, req.EntityID)
		return &NotifyResponse{Skipped: true, Message: "duplicate: already notified"}, nil
	}

	emailErr := sendEmail(ctx, req)
	sendRealtime(ctx, req)

	if emailErr != nil {
		log.Printf("[notifications] email dispatch failed: %v", emailErr)
		_ = saveFailedNotification(ctx, req)
		return nil, emailErr
	}

	if err := saveNotification(ctx, req); err != nil {
		return nil, err
	}

	return &NotifyResponse{Skipped: false, Message: "notification sent"}, nil
}

// GetNotifications возвращает список уведомлений текущего пользователя.
//
//encore:api auth method=GET path=/notifications
func (s *Service) GetNotifications(ctx context.Context) (*ListNotificationsResponse, error) {
	userID := auth.Data().(*authhandler.AuthData).KeycloakUserID

	items, err := listNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Notification{} // никогда не возвращаем null на фронт
	}
	return &ListNotificationsResponse{
		Notifications: items,
		Total:         len(items),
	}, nil
}
