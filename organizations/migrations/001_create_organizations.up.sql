CREATE TABLE organizations (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    code       VARCHAR(100) NOT NULL UNIQUE,
    parent_id  UUID         REFERENCES organizations(id) ON DELETE SET NULL,
    type       VARCHAR(50)  NOT NULL DEFAULT 'subsidiary',
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_parent_id ON organizations(parent_id);
CREATE INDEX idx_organizations_code ON organizations(code);
