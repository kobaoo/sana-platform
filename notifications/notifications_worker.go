package notifications

import (
	"context"
	"log"
)

// NotifyUserWithEntity — основной Encore private API для отправки уведомлений.
// Вызывается доменными сервисами: certificates, requests, courses.
//
// Порядок выполнения:
//  1. Анти-дублирующая проверка → пропуск если уже отправлялось
//  2. Dispatch в email + realtime каналы
//  3. Сохранение записи (SENT / FAILED)
//
//encore:api private
func (s *Service) NotifyUserWithEntity(ctx context.Context, req *NotifyRequest) (*NotifyResponse, error) {
	log.Printf("[notifications] processing: user=%s type=%s entity=%s/%s",
		req.UserID, req.Type, req.EntityType, req.EntityID)

	// 1. Анти-дублирующая проверка
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

	// 2. Dispatch по каналам
	emailErr := sendEmail(ctx, req)
	sendRealtime(ctx, req) // stub — fire-and-forget, никогда не падает

	// 3. Сохранение результата
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
