-- Enum types для notifications
CREATE TYPE notification_type        AS ENUM ('CERT_EXPIRING', 'CERT_EXPIRED');
CREATE TYPE notification_entity_type AS ENUM ('CERTIFICATE');
CREATE TYPE notification_status      AS ENUM ('PENDING', 'SENT', 'FAILED');

CREATE TABLE notifications (
    id          UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID                     NOT NULL,
    type        notification_type        NOT NULL,
    entity_type notification_entity_type NOT NULL,
    entity_id   UUID                     NOT NULL,
    status      notification_status      NOT NULL DEFAULT 'PENDING',
    sent_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ              NOT NULL DEFAULT NOW(),

    -- Анти-дублирующий ключ: одно уведомление на (user, type, entity)
    CONSTRAINT notifications_dedup_key
        UNIQUE (user_id, type, entity_type, entity_id)
);

CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_notifications_status  ON notifications (status);