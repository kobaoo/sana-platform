package notifications

import (
	"context"
	"log"
)

// sendEmail отправляет email-уведомление пользователю.
// Stub — реальная интеграция с email-провайдером (SendGrid / AWS SES) позже.
func sendEmail(_ context.Context, req *NotifyRequest) error {
	// TODO: интегрировать email-провайдер (SendGrid / AWS SES)
	log.Printf("[notifications:email] stub → user=%s type=%s entity=%s/%s",
		req.UserID, req.Type, req.EntityType, req.EntityID)
	return nil
}

// sendRealtime отправляет real-time push (WebSocket / SSE).
// Stub — real-time слой (Centrifugo / Pusher) подключается позже.
func sendRealtime(_ context.Context, req *NotifyRequest) {
	// TODO: интегрировать real-time провайдер (Centrifugo / Pusher)
	log.Printf("[notifications:realtime] stub → user=%s type=%s entity=%s/%s",
		req.UserID, req.Type, req.EntityType, req.EntityID)
}
