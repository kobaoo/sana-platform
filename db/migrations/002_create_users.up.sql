CREATE TABLE users (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_user_id  VARCHAR(255) NOT NULL UNIQUE,
    email             VARCHAR(255) NOT NULL,
    role              VARCHAR(20)  NOT NULL DEFAULT 'EMP',
    dzo_id            UUID,
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    is_onboarded      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email  ON users(email);
CREATE INDEX idx_users_dzo_id ON users(dzo_id);
