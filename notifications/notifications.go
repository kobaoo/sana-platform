package notifications

import "encore.dev/storage/sqldb"

// db — подключение к базе "lms" для хранения истории уведомлений.
//
// NOTE: После запуска `go generate ./db/ent/...`
// (когда Notification-схема зарегистрирована в Ent)
// репозиторный слой можно перевести на Ent-клиент.
var db = sqldb.Named("lms")

// ════ SERVICE ════

//encore:service
type Service struct{}

func initService() (*Service, error) {
	return &Service{}, nil
}
