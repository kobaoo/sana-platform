# Sana Platform — Backend

Encore-приложение. Go + PostgreSQL + Keycloak.

## Быстрый старт

### 1. Зависимости

- [Encore](https://encore.dev/docs/install) `encore version`
- Go 1.22+
- Docker (для Keycloak и PostgreSQL)

### 2. Локальные секреты

Скопируй шаблон и заполни своими значениями:

```bash
cp .secrets.local.cue.example .secrets.local.cue
```

Файл `.secrets.local.cue` уже в `.gitignore` — не коммитить.

Содержимое шаблона:

```cue
KeycloakIssuerURL:     "http://localhost:8080/realms/sana-lms"
KeycloakAudience:      "account"
KeycloakAdminUser:     "admin"
KeycloakAdminPassword: "admin"

// SMTP — email notifications (cron)
MailServer:   "smtp.gmail.com"
MailPort:     "587"
MailUsername: "your-email@gmail.com"
MailPassword: "your-app-password"
MailFrom:     "your-email@gmail.com"
AppURL:       "http://localhost:3000"
```

> Gmail: используй App Password (не обычный пароль).  
> Создать: Google Account → Security → 2-Step Verification → App passwords.

### 3. Запуск

```bash
encore run
```

Dashboard: http://localhost:9400

---

## Структура проекта

```
auth/           — аутентификация (Keycloak)
learning/
  certificates/ — управление сертификатами + cron-уведомления
notifications/  — система уведомлений
requests/       — заявки
db/
  ent/          — ORM-схемы и сгенерированный код
  migrations/   — SQL-миграции
```

---

## Cron: проверка истекающих сертификатов

Запускается каждый **понедельник в 10:00 Астана (05:00 UTC)**.

Логика: выбирает сертификаты с `expiry_date` в диапазоне `сегодня < дата ≤ сегодня + 6 месяцев`, группирует по ДЗО, отправляет email HR каждого ДЗО.

Ручной запуск через Dashboard: http://localhost:9400 → Cron Jobs → `check-expiring-certs` → Trigger.

### Превью письма (для дизайнера/разработчика)

```bash
go run send_html_email.go -preview
# создаёт ./tmp/certificates-preview.html и ./tmp/certificates-preview.eml
```

---

## Тесты

```bash
encore test ./...
```

Юнит-тесты без базы:

```bash
go test ./learning/certificates/certutil/...
```

---

## Prod-секреты (деплой)

Для стейджинга и прода используй Encore Secrets:

```bash
encore secret set --prod MailUsername
encore secret set --prod MailPassword
# и т.д.
```
