CREATE TABLE contracts_dzo (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    dzo_id           UUID           NOT NULL REFERENCES organizations(id),
    contract_number  VARCHAR(100)   NOT NULL,
    category         VARCHAR(100)   NOT NULL,
    signed_date      DATE           NOT NULL,
    expiry_date      DATE,
    amount_with_vat  DECIMAL(14, 2) NOT NULL,
    amendment_number VARCHAR(100),
    amendment_date   DATE,
    amendment_amount DECIMAL(14, 2),
    total_amount     DECIMAL(14, 2) NOT NULL,
    spent_amount     DECIMAL(14, 2) NOT NULL DEFAULT 0,
    remaining_amount DECIMAL(14, 2) NOT NULL,
    is_active        BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT contracts_dzo_amount_non_negative CHECK (amount_with_vat >= 0),
    CONSTRAINT contracts_dzo_amendment_non_negative CHECK (amendment_amount IS NULL OR amendment_amount >= 0),
    CONSTRAINT contracts_dzo_spent_non_negative CHECK (spent_amount >= 0),
    CONSTRAINT contracts_dzo_remaining_non_negative CHECK (remaining_amount >= 0)
);

CREATE INDEX idx_contracts_dzo_dzo_id ON contracts_dzo(dzo_id);
CREATE INDEX idx_contracts_dzo_is_active ON contracts_dzo(is_active);
CREATE INDEX idx_contracts_dzo_remaining_amount ON contracts_dzo(remaining_amount);
